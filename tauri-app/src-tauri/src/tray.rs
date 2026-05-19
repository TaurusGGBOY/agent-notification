use tauri::menu::{Menu, MenuItem, PredefinedMenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Manager};

use crate::service;

pub fn build_tray(app: &AppHandle) -> tauri::Result<()> {
    let open = MenuItem::with_id(app, "open", "Open AgentNotify", true, None::<&str>)?;
    let test = MenuItem::with_id(app, "test", "Send Test Notification", true, None::<&str>)?;
    let pause = MenuItem::with_id(app, "pause", "Pause Notifications", true, None::<&str>)?;
    let resume = MenuItem::with_id(app, "resume", "Resume Notifications", true, None::<&str>)?;
    let restart = MenuItem::with_id(app, "restart", "Restart Service", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
    let separator = PredefinedMenuItem::separator(app)?;

    let menu = Menu::with_items(
        app,
        &[&open, &test, &pause, &resume, &restart, &separator, &quit],
    )?;

    TrayIconBuilder::with_id("agentnotify")
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
            "quit" => app.exit(0),
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
        r#"{"agent":"tauri","event":"start","project":"AgentNotify","message":"Test notification from tray","sourcePayload":{}}"#,
    )
    .await
}

async fn set_events_enabled(enabled: bool) -> Result<(), String> {
    let events = if enabled {
        r#"["start","stop"]"#
    } else {
        "[]"
    };
    let body = format!(
        r#"{{"notificationStyle":"custom-card","enabledEvents":{events},"futureOverrides":{{}}}}"#
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
    use std::io::{Read, Write};
    use std::net::TcpStream;

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