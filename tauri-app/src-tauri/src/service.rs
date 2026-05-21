use std::io::{Read, Write};
use std::net::TcpStream;
use std::sync::Mutex;
use std::time::Duration;

use tauri::{AppHandle, Emitter, Manager, State};
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

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
    let client = match Client::new("127.0.0.1:17891", Duration::from_millis(600)) {
        Ok(c) => c,
        Err(_) => return false,
    };
    client.get("/health").is_ok()
}

pub fn ensure_sidecar(app: &AppHandle) -> Result<(), String> {
    if is_server_healthy() {
        return Ok(());
    }

    let state = app.state::<ServiceState>();
    let mut guard = state.child.lock().map_err(|_| "service mutex poisoned")?;
    if guard.is_some() {
        return Ok(());
    }

    let command = app
        .shell()
        .sidecar("agent-notify-server")
        .map_err(|err| format!("failed to create sidecar command: {err}"))?;

    let (mut rx, child) = command
        .env("AGENT_NOTIFY_HTTP_ADDR", "127.0.0.1:17891")
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
    Ok(())
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
        managed_by_tauri: state.child.lock().map(|child| child.is_some()).unwrap_or(false),
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
        let mut stream = TcpStream::connect(&self.addr).map_err(|err| err.to_string())?;
        stream
            .set_read_timeout(Some(self.timeout))
            .map_err(|err| err.to_string())?;
        stream
            .set_write_timeout(Some(self.timeout))
            .map_err(|err| err.to_string())?;
        let req = format!(
            "GET {path} HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n"
        );
        stream.write_all(req.as_bytes()).map_err(|err| err.to_string())?;
        let mut response = String::new();
        stream.read_to_string(&mut response).map_err(|err| err.to_string())?;
        if response.starts_with("HTTP/1.1 200") || response.starts_with("HTTP/1.0 200") {
            Ok(())
        } else {
            Err(response.lines().next().unwrap_or("empty response").to_string())
        }
    }
}