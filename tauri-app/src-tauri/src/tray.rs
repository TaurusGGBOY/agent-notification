use std::io::{Read, Write};
use std::net::TcpStream;

use tauri::menu::{Menu, MenuItem, PredefinedMenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Emitter, Manager};

use crate::{lifecycle, service};

pub fn build_tray(app: &AppHandle) -> tauri::Result<()> {
    let open = MenuItem::with_id(app, "open", "打开 AgentNotify", true, None::<&str>)?;
    let test = MenuItem::with_id(app, "test", "发送测试通知", true, None::<&str>)?;
    let pause = MenuItem::with_id(app, "pause", "暂停通知", true, None::<&str>)?;
    let resume = MenuItem::with_id(app, "resume", "恢复通知", true, None::<&str>)?;
    let restart = MenuItem::with_id(app, "restart", "重启服务", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
    let separator = PredefinedMenuItem::separator(app)?;

    let menu = Menu::with_items(
        app,
        &[&open, &test, &pause, &resume, &restart, &separator, &quit],
    )?;

    TrayIconBuilder::with_id("agentnotify")
        .icon(
            app.default_window_icon()
                .cloned()
                .expect("missing default window icon"),
        )
        .icon_as_template(cfg!(target_os = "macos"))
        .tooltip("AgentNotify")
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                show_main_window(tray.app_handle());
            }
        })
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => show_main_window(app),
            "test" => {
                let app = app.clone();
                tauri::async_runtime::spawn(async move {
                    let _ = send_test_notification().await;
                    let _ = app.emit("agentnotify://refresh", ());
                });
            }
            "pause" => {
                let app = app.clone();
                tauri::async_runtime::spawn(async move {
                    let _ = set_events_enabled(false).await;
                    let _ = app.emit("agentnotify://refresh", ());
                });
            }
            "resume" => {
                let app = app.clone();
                tauri::async_runtime::spawn(async move {
                    let _ = set_events_enabled(true).await;
                    let _ = app.emit("agentnotify://refresh", ());
                });
            }
            "restart" => {
                let state = app.state::<service::ServiceState>();
                service::stop_sidecar(&state);
                let _ = service::ensure_sidecar(app);
            }
            "quit" => {
                lifecycle::request_tray_exit(app);
                app.exit(0);
            }
            _ => {}
        })
        .build(app)?;

    Ok(())
}

fn show_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.unminimize();
        let _ = window.show();
        let _ = window.set_focus();
    }
}

async fn send_test_notification() -> Result<(), String> {
    post_json(
        "/notify",
        r#"{"agent":"tauri","event":"start","project":"AgentNotify","message":"来自托盘的测试通知","sourcePayload":{}}"#,
    )
    .await
}

async fn get_config() -> Result<serde_json::Value, String> {
    let addr = "127.0.0.1:17891";
    let request = "GET /config HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n";
    let addr = addr.to_string();
    let request = request.to_string();
    tauri::async_runtime::spawn_blocking(move || {
        let mut stream = TcpStream::connect(addr).map_err(|err| err.to_string())?;
        stream
            .write_all(request.as_bytes())
            .map_err(|err| err.to_string())?;
        let mut response = String::new();
        stream
            .read_to_string(&mut response)
            .map_err(|err| err.to_string())?;
        let body = response.split("\r\n\r\n").nth(1).unwrap_or("{}");
        serde_json::from_str(body).map_err(|err| err.to_string())
    })
    .await
    .map_err(|err| err.to_string())?
}

async fn set_events_enabled(enabled: bool) -> Result<(), String> {
    let config = get_config().await?;
    let notification_style = config
        .get("notificationStyle")
        .and_then(|v| v.as_str())
        .unwrap_or("clean");
    let future_overrides = config
        .get("futureOverrides")
        .and_then(|v| serde_json::to_string(v).ok())
        .unwrap_or_else(|| "{}".to_string());

    let events = if enabled { r#"["start","stop"]"# } else { "[]" };
    let body = format!(
        r#"{{"notificationStyle":"{}","enabledEvents":{},"futureOverrides":{}}}"#,
        notification_style, events, future_overrides
    );
    post_json("/settings", &body).await
}

async fn post_json(path: &str, body: &str) -> Result<(), String> {
    let addr = "127.0.0.1:17891";
    let request = format!(
        "POST {path} HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
        body.len(),
        body
    );
    tokio_like_tcp(addr, &request).await
}

async fn tokio_like_tcp(addr: &str, request: &str) -> Result<(), String> {
    let addr = addr.to_string();
    let request = request.to_string();
    tauri::async_runtime::spawn_blocking(move || {
        let mut stream = TcpStream::connect(addr).map_err(|err| err.to_string())?;
        stream
            .write_all(request.as_bytes())
            .map_err(|err| err.to_string())?;
        let mut response = String::new();
        stream
            .read_to_string(&mut response)
            .map_err(|err| err.to_string())?;
        if response.starts_with("HTTP/1.1 204")
            || response.starts_with("HTTP/1.0 204")
            || response.starts_with("HTTP/1.1 200")
            || response.starts_with("HTTP/1.0 200")
        {
            Ok(())
        } else {
            Err(response
                .lines()
                .next()
                .unwrap_or("empty response")
                .to_string())
        }
    })
    .await
    .map_err(|err| err.to_string())?
}
