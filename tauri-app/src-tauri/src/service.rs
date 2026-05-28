use std::io::{Read, Write};
use std::net::{TcpStream, ToSocketAddrs};
use std::sync::Mutex;
use std::time::{Duration, Instant};

use tauri::{AppHandle, Emitter, Manager, State};
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

const CONTROL_ADDR: &str = "127.0.0.1:17891";
// This LAN-only tool intentionally exposes the entire unauthenticated Agent Notify HTTP API on the LAN.
const SIDECAR_LISTEN_ADDR: &str = "0.0.0.0:17891";

pub fn control_addr() -> &'static str {
    CONTROL_ADDR
}

pub fn sidecar_listen_addr() -> &'static str {
    SIDECAR_LISTEN_ADDR
}

pub struct ServiceState {
    child: Mutex<Option<CommandChild>>,
}

impl ServiceState {
    pub fn new() -> Self {
        Self {
            child: Mutex::new(None),
        }
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

fn manifest_lan_addr() -> Result<String, String> {
    let client = Client::new(control_addr(), Duration::from_millis(600))?;
    let body = client.get_text("/manifest")?;
    let manifest: serde_json::Value = serde_json::from_str(&body).map_err(|err| err.to_string())?;
    let url = manifest
        .get("url")
        .and_then(|value| value.as_str())
        .ok_or_else(|| "manifest missing url".to_string())?;
    http_url_host_port(url).ok_or_else(|| format!("manifest url is not LAN reachable: {url}"))
}

pub fn http_url_host_port(url: &str) -> Option<String> {
    let rest = url.strip_prefix("http://")?;
    let authority = rest.split('/').next().unwrap_or(rest);
    let (host, port) = authority.rsplit_once(':')?;
    let host = host.trim();
    let port = port.trim();
    let bare_host = host.trim_matches(['[', ']']);
    let lower_host = bare_host.to_ascii_lowercase();

    if host.is_empty()
        || port.is_empty()
        || authority.contains('@')
        || port.parse::<u16>().ok()? == 0
        || lower_host == "localhost"
        || lower_host == "::1"
        || lower_host == "0.0.0.0"
        || lower_host.starts_with("127.")
    {
        return None;
    }

    Some(format!("{host}:{port}"))
}

pub fn is_server_lan_reachable() -> bool {
    let addr = match manifest_lan_addr() {
        Ok(addr) => addr,
        Err(_) => return false,
    };
    let client = match Client::new(&addr, Duration::from_millis(600)) {
        Ok(c) => c,
        Err(_) => return false,
    };
    client.get("/health").is_ok()
}

fn wait_for_server_lan_reachable(timeout: Duration) -> bool {
    let deadline = Instant::now() + timeout;
    while Instant::now() < deadline {
        if is_server_lan_reachable() {
            return true;
        }
        std::thread::sleep(Duration::from_millis(100));
    }
    is_server_lan_reachable()
}

pub fn ensure_sidecar(app: &AppHandle) -> Result<(), String> {
    if is_server_healthy() && is_server_lan_reachable() {
        return Ok(());
    }

    let state = app.state::<ServiceState>();
    let mut guard = state.child.lock().map_err(|_| "service mutex poisoned")?;
    if guard.is_some() {
        drop(guard);
        if wait_for_server_lan_reachable(Duration::from_secs(5)) {
            return Ok(());
        }
        return Err(format!(
            "server running but did not become LAN reachable on {}",
            sidecar_listen_addr()
        ));
    }

    let command = app
        .shell()
        .sidecar("agent-notify-server")
        .map_err(|err| format!("failed to create sidecar command: {err}"))?;

    let (mut rx, child) = command
        .env("AGENT_NOTIFY_HTTP_ADDR", sidecar_listen_addr())
        .spawn()
        .map_err(|err| format!("failed to spawn sidecar: {err}"))?;

    let app_for_events = app.clone();
    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(bytes) => {
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

    if wait_for_server_lan_reachable(Duration::from_secs(5)) {
        Ok(())
    } else {
        let mut guard = state.child.lock().map_err(|_| "service mutex poisoned")?;
        if let Some(child) = guard.take() {
            let _ = child.kill();
        }
        Err(format!(
            "server started but did not become LAN reachable on {}",
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
        managed_by_tauri: state
            .child
            .lock()
            .map(|child| child.is_some())
            .unwrap_or(false),
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

#[cfg(test)]
mod tests {
    use super::{control_addr, http_url_host_port, sidecar_listen_addr};

    #[test]
    fn sidecar_listens_on_all_interfaces_for_lan_agents() {
        assert_eq!(sidecar_listen_addr(), "0.0.0.0:17891");
    }

    #[test]
    fn tauri_controls_server_through_loopback() {
        assert_eq!(control_addr(), "127.0.0.1:17891");
    }

    #[test]
    fn parses_lan_manifest_url_host_port() {
        assert_eq!(
            http_url_host_port("http://192.168.1.10:17891/manifest"),
            Some("192.168.1.10:17891".to_string())
        );
    }

    #[test]
    fn rejects_loopback_manifest_urls_for_lan_reuse() {
        assert_eq!(http_url_host_port("http://127.0.0.1:17891/manifest"), None);
        assert_eq!(http_url_host_port("http://localhost:17891/manifest"), None);
    }

    #[test]
    fn rejects_non_lan_manifest_urls() {
        assert_eq!(http_url_host_port("http://0.0.0.0:17891/manifest"), None);
        assert_eq!(http_url_host_port("http://192.168.1.10:0/manifest"), None);
        assert_eq!(
            http_url_host_port("http://user@192.168.1.10:17891/manifest"),
            None
        );
        assert_eq!(
            http_url_host_port("https://192.168.1.10:17891/manifest"),
            None
        );
        assert_eq!(http_url_host_port("http://[::1]:17891/manifest"), None);
        assert_eq!(
            http_url_host_port("http://192.168.1.10:notaport/manifest"),
            None
        );
    }
}
