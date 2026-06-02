use std::io::{Read, Write};
use std::net::{TcpStream, ToSocketAddrs};
use std::process;
use std::sync::Mutex;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

#[cfg(windows)]
use std::os::windows::process::CommandExt;

use tauri::{AppHandle, Emitter, Manager, State};
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

use crate::native_notification;

const CONTROL_ADDR: &str = "127.0.0.1:17891";
// This LAN-only tool intentionally exposes the entire unauthenticated Agent Notify HTTP API on the LAN.
const SIDECAR_LISTEN_ADDR: &str = "0.0.0.0:17891";
const NOTIFICATION_STDOUT_PREFIX: &str = "AGENT_NOTIFY_TAURI_NOTIFICATION ";
#[cfg(windows)]
const CREATE_NO_WINDOW: u32 = 0x0800_0000;

pub fn control_addr() -> &'static str {
    CONTROL_ADDR
}

pub fn sidecar_listen_addr() -> &'static str {
    SIDECAR_LISTEN_ADDR
}

pub struct ServiceState {
    child: Mutex<Option<CommandChild>>,
    instance_token: String,
}

impl ServiceState {
    pub fn new() -> Self {
        Self {
            child: Mutex::new(None),
            instance_token: new_instance_token(),
        }
    }

    fn instance_token(&self) -> &str {
        &self.instance_token
    }
}

#[derive(serde::Serialize)]
pub struct ServiceStatus {
    pub healthy: bool,
    pub managed_by_tauri: bool,
}

pub fn is_server_healthy() -> bool {
    let client = match Client::new(control_addr(), Duration::from_millis(600)) {
        Ok(c) => c,
        Err(_) => return false,
    };
    client.get("/health").is_ok()
}

fn new_instance_token() -> String {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or_default();
    format!("{}-{nanos}", process::id())
}

fn health_instance_token_matches(body: &str, expected_token: &str) -> bool {
    if expected_token.trim().is_empty() {
        return false;
    }
    let Ok(health) = serde_json::from_str::<serde_json::Value>(body) else {
        return false;
    };
    health.get("instanceToken").and_then(|value| value.as_str()) == Some(expected_token)
}

fn is_managed_server_healthy(instance_token: &str) -> bool {
    let client = match Client::new(control_addr(), Duration::from_millis(600)) {
        Ok(c) => c,
        Err(_) => return false,
    };
    match client.get_text("/health") {
        Ok(body) => health_instance_token_matches(&body, instance_token),
        Err(_) => false,
    }
}

fn wait_for_managed_server_ready(instance_token: &str, timeout: Duration) -> bool {
    wait_for_managed_server_ready_with(timeout, || is_managed_server_healthy(instance_token))
}

fn wait_for_managed_server_ready_with<F>(timeout: Duration, mut is_managed_healthy: F) -> bool
where
    F: FnMut() -> bool,
{
    let deadline = Instant::now() + timeout;
    while Instant::now() < deadline {
        if is_managed_healthy() {
            return true;
        }
        std::thread::sleep(Duration::from_millis(100));
    }
    is_managed_healthy()
}

#[cfg(windows)]
fn wait_for_server_to_stop(timeout: Duration) -> bool {
    let deadline = Instant::now() + timeout;
    while Instant::now() < deadline {
        if !is_server_healthy() {
            return true;
        }
        std::thread::sleep(Duration::from_millis(100));
    }
    !is_server_healthy()
}

fn cleanup_windows_legacy_server_autostart() {
    #[cfg(windows)]
    run_hidden_windows_command("schtasks", &["/delete", "/tn", "AgentNotifyServer", "/f"]);
}

#[cfg(windows)]
fn stop_windows_standalone_server_processes() {
    run_hidden_windows_command("taskkill", &["/F", "/T", "/IM", "agent-notify-server.exe"]);
}

#[cfg(windows)]
fn run_hidden_windows_command(program: &str, args: &[&str]) {
    let _ = std::process::Command::new(program)
        .args(args)
        .creation_flags(CREATE_NO_WINDOW)
        .status();
}

pub fn ensure_sidecar(app: &AppHandle) -> Result<(), String> {
    let state = app.state::<ServiceState>();
    let instance_token = state.instance_token().to_string();

    cleanup_windows_legacy_server_autostart();

    if is_managed_server_healthy(&instance_token) {
        return Ok(());
    }

    let mut guard = state.child.lock().map_err(|_| "service mutex poisoned")?;
    if guard.is_some() {
        drop(guard);
        if wait_for_managed_server_ready(&instance_token, Duration::from_secs(5)) {
            return Ok(());
        }
        return Err(format!(
            "server running but did not become healthy on {}",
            sidecar_listen_addr()
        ));
    }

    #[cfg(windows)]
    {
        if is_server_healthy() {
            stop_windows_standalone_server_processes();
            let _ = wait_for_server_to_stop(Duration::from_secs(2));
        }
    }

    if is_server_healthy() {
        return Ok(());
    }

    let mut command = app
        .shell()
        .sidecar("agent-notify-server")
        .map_err(|err| format!("failed to create sidecar command: {err}"))?
        .env("AGENT_NOTIFY_HTTP_ADDR", sidecar_listen_addr())
        .env("AGENT_NOTIFY_INSTANCE_TOKEN", &instance_token);

    #[cfg(target_os = "macos")]
    {
        command = command.env("AGENT_NOTIFY_TAURI_STDOUT", "1");
    }

    let (mut rx, child) = command
        .spawn()
        .map_err(|err| format!("failed to spawn sidecar: {err}"))?;

    let app_for_events = app.clone();
    tauri::async_runtime::spawn(async move {
        let mut stdout_router = SidecarStdoutRouter::default();
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(bytes) => {
                    for notification in stdout_router.push(&bytes) {
                        show_forwarded_notification(app_for_events.clone(), notification);
                    }
                    let line = String::from_utf8_lossy(&bytes).to_string();
                    let _ = app_for_events.emit("agentnotify://server-stdout", line);
                }
                CommandEvent::Stderr(bytes) => {
                    let line = String::from_utf8_lossy(&bytes).to_string();
                    let _ = app_for_events.emit("agentnotify://server-stderr", line);
                }
                _ => {}
            }
        }
    });

    *guard = Some(child);
    drop(guard);

    if wait_for_managed_server_ready(&instance_token, Duration::from_secs(5)) {
        Ok(())
    } else {
        let mut guard = state.child.lock().map_err(|_| "service mutex poisoned")?;
        if let Some(child) = guard.take() {
            let _ = child.kill();
        }
        Err(format!(
            "server started but did not become healthy on {}",
            sidecar_listen_addr()
        ))
    }
}

pub fn stop_sidecar(state: &State<ServiceState>) {
    if let Ok(mut guard) = state.child.lock() {
        if let Some(child) = guard.take() {
            let _ = child.kill();
        }
    }
}

#[tauri::command]
pub fn restart_service(app: AppHandle) -> Result<(), String> {
    let state = app.state::<ServiceState>();
    stop_sidecar(&state);
    ensure_sidecar(&app)
}

#[tauri::command]
pub fn service_status(state: State<ServiceState>) -> ServiceStatus {
    ServiceStatus {
        healthy: is_server_healthy(),
        managed_by_tauri: is_managed_server_healthy(state.instance_token()),
    }
}

struct Client {
    addr: String,
    timeout: Duration,
}

impl Client {
    fn new(addr: &str, timeout: Duration) -> Result<Self, String> {
        Ok(Self {
            addr: addr.to_string(),
            timeout,
        })
    }

    fn get(&self, path: &str) -> Result<(), String> {
        self.get_text(path).map(|_| ())
    }

    fn get_text(&self, path: &str) -> Result<String, String> {
        let socket_addr = self
            .addr
            .to_socket_addrs()
            .map_err(|err| format!("failed to resolve {}: {err}", self.addr))?
            .next()
            .ok_or_else(|| format!("failed to resolve {}: no socket addresses", self.addr))?;
        let mut stream = TcpStream::connect_timeout(&socket_addr, self.timeout)
            .map_err(|err| err.to_string())?;
        stream
            .set_read_timeout(Some(self.timeout))
            .map_err(|err| err.to_string())?;
        stream
            .set_write_timeout(Some(self.timeout))
            .map_err(|err| err.to_string())?;
        let req = format!(
            "GET {path} HTTP/1.1\r\nHost: {}\r\nConnection: close\r\n\r\n",
            self.addr
        );
        stream
            .write_all(req.as_bytes())
            .map_err(|err| err.to_string())?;
        let mut response = String::new();
        stream
            .read_to_string(&mut response)
            .map_err(|err| err.to_string())?;
        if response.starts_with("HTTP/1.1 200") || response.starts_with("HTTP/1.0 200") {
            Ok(response
                .split_once("\r\n\r\n")
                .map(|(_, body)| body.to_string())
                .unwrap_or_default())
        } else {
            Err(response
                .lines()
                .next()
                .unwrap_or("empty response")
                .to_string())
        }
    }
}

#[derive(Clone, Debug, PartialEq, serde::Deserialize)]
struct ForwardedNotification {
    title: String,
    body: String,
}

#[derive(Default)]
struct SidecarStdoutRouter {
    pending: Vec<u8>,
}

impl SidecarStdoutRouter {
    fn push(&mut self, chunk: &[u8]) -> Vec<ForwardedNotification> {
        self.pending.extend_from_slice(chunk);
        let mut notifications = Vec::new();

        while let Some(newline) = self.pending.iter().position(|byte| *byte == b'\n') {
            let mut line = self.pending.drain(..=newline).collect::<Vec<_>>();
            while matches!(line.last(), Some(b'\r' | b'\n')) {
                line.pop();
            }
            let line = String::from_utf8_lossy(&line);
            if let Some(notification) = parse_forwarded_notification_line(&line) {
                notifications.push(notification);
            }
        }

        notifications
    }
}

fn parse_forwarded_notification_line(line: &str) -> Option<ForwardedNotification> {
    let payload = line.strip_prefix(NOTIFICATION_STDOUT_PREFIX)?;
    serde_json::from_str(payload).ok()
}

fn show_forwarded_notification(app: AppHandle, notification: ForwardedNotification) {
    let emit_app = app.clone();
    let result = app.run_on_main_thread(move || {
        if let Err(err) = native_notification::show(&notification.title, &notification.body) {
            let _ = emit_app.emit(
                "agentnotify://server-stderr",
                format!("native macOS notification failed: {err}\n"),
            );
        }
    });
    if let Err(err) = result {
        let _ = app.emit(
            "agentnotify://server-stderr",
            format!("schedule native macOS notification failed: {err}\n"),
        );
    }
}

#[cfg(test)]
mod tests {
    use std::time::Duration;

    use super::{
        control_addr, health_instance_token_matches, wait_for_managed_server_ready_with,
        parse_forwarded_notification_line, sidecar_listen_addr, SidecarStdoutRouter,
    };

    #[test]
    fn sidecar_listens_on_all_interfaces_for_lan_agents() {
        assert_eq!(sidecar_listen_addr(), "0.0.0.0:17891");
    }

    #[test]
    fn tauri_controls_server_through_loopback() {
        assert_eq!(control_addr(), "127.0.0.1:17891");
    }

    #[test]
    fn health_instance_token_must_match_managed_sidecar() {
        let managed = r#"{"status":"ok","version":"1.0.1","instanceToken":"token-123"}"#;
        let external = r#"{"status":"ok","version":"1.0.1"}"#;
        let other_instance = r#"{"status":"ok","version":"1.0.1","instanceToken":"other"}"#;

        assert!(health_instance_token_matches(managed, "token-123"));
        assert!(!health_instance_token_matches(external, "token-123"));
        assert!(!health_instance_token_matches(other_instance, "token-123"));
        assert!(!health_instance_token_matches(managed, ""));
    }

    #[test]
    fn managed_service_readiness_does_not_require_lan_self_connect() {
        assert!(wait_for_managed_server_ready_with(Duration::ZERO, || true));
    }

    #[test]
    fn parses_forwarded_tauri_notification_stdout_line() {
        let line = r#"AGENT_NOTIFY_TAURI_NOTIFICATION {"title":"Codex Started","body":"hello"}"#;

        let notification = parse_forwarded_notification_line(line).expect("notification");

        assert_eq!(notification.title, "Codex Started");
        assert_eq!(notification.body, "hello");
    }

    #[test]
    fn routes_forwarded_notifications_from_split_stdout_chunks() {
        let mut router = SidecarStdoutRouter::default();

        assert!(router
            .push(r#"AGENT_NOTIFY_TAURI_NOTIFICATION {"title":"Codex"#.as_bytes())
            .is_empty());
        let notifications =
            router.push(" Started\",\"body\":\"hello\"}\nplain log line\n".as_bytes());

        assert_eq!(notifications.len(), 1);
        assert_eq!(notifications[0].title, "Codex Started");
        assert_eq!(notifications[0].body, "hello");
    }

    #[test]
    fn routes_forwarded_notifications_with_utf8_split_between_chunks() {
        let mut router = SidecarStdoutRouter::default();
        let line = r#"AGENT_NOTIFY_TAURI_NOTIFICATION {"title":"开始通知","body":"agent-notification 已启动"}"#;
        let bytes = line.as_bytes();
        let split = bytes
            .windows("通知".len())
            .position(|window| window == "通知".as_bytes())
            .expect("notification text in test line")
            + 1;

        assert!(router.push(&bytes[..split]).is_empty());
        let notifications = router.push(&[&bytes[split..], b"\n"].concat());

        assert_eq!(notifications.len(), 1);
        assert_eq!(notifications[0].title, "开始通知");
        assert_eq!(notifications[0].body, "agent-notification 已启动");
    }
}
