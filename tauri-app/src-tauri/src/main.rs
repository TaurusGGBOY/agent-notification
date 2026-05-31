#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod lifecycle;
mod notification_settings;
mod service;
mod tray;

use tauri::{Manager, Theme};

#[tauri::command]
fn set_app_theme(app: tauri::AppHandle, theme: String) -> Result<(), String> {
    let theme = match theme.as_str() {
        "light" => Theme::Light,
        "dark" => Theme::Dark,
        value => return Err(format!("unsupported theme: {value}")),
    };

    app.set_theme(Some(theme));
    for window in app.webview_windows().values() {
        window
            .set_theme(Some(theme))
            .map_err(|err| err.to_string())?;
    }
    Ok(())
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_shell::init())
        .enable_macos_default_menu(false)
        .manage(service::ServiceState::new())
        .manage(lifecycle::TrayExitState::default())
        .invoke_handler(tauri::generate_handler![
            notification_settings::open_windows_notification_settings,
            notification_settings::windows_notification_status,
            notification_settings::open_macos_notification_settings,
            notification_settings::macos_notification_status,
            service::service_status,
            service::restart_service,
            set_app_theme
        ])
        .setup(|app| {
            #[cfg(target_os = "macos")]
            app.set_activation_policy(tauri::ActivationPolicy::Accessory);
            if let Err(err) = service::ensure_sidecar(app.handle()) {
                eprintln!("AgentNotify sidecar startup warning: {err}");
            }
            tray::build_tray(app.handle())?;
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.set_size(tauri::Size::Logical(tauri::LogicalSize {
                    width: 1200.0,
                    height: 675.0,
                }));
                let _ = window.center();
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                if window.label() == "main" {
                    api.prevent_close();
                    let _ = window.hide();
                }
            }
        })
        .build(tauri::generate_context!())
        .expect("error while building AgentNotify")
        .run(|app, event| {
            if let tauri::RunEvent::ExitRequested { api, .. } = event {
                let tray_exit = app.state::<lifecycle::TrayExitState>();
                if !tray_exit.consume_tray_exit_request() {
                    api.prevent_exit();
                    lifecycle::hide_main_window(app);
                    return;
                }

                let state = app.state::<service::ServiceState>();
                service::stop_sidecar(&state);
            }
        });
}
