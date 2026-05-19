mod service;
mod tray;

use tauri::Manager;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(service::ServiceState::new())
        .invoke_handler(tauri::generate_handler![service::service_status, service::restart_service])
        .setup(|app| {
            service::ensure_sidecar(app.handle())?;
            tray::build_tray(app.handle())?;
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
            if let tauri::RunEvent::ExitRequested { .. } = event {
                let state = app.state::<service::ServiceState>();
                service::stop_sidecar(&state);
            }
        });
}