use std::sync::atomic::{AtomicBool, Ordering};

use tauri::{AppHandle, Manager};

#[derive(Default)]
pub struct TrayExitState {
    tray_exit_requested: AtomicBool,
}

impl TrayExitState {
    pub fn request_tray_exit(&self) {
        self.tray_exit_requested.store(true, Ordering::SeqCst);
    }

    pub fn consume_tray_exit_request(&self) -> bool {
        self.tray_exit_requested.swap(false, Ordering::SeqCst)
    }
}

pub fn request_tray_exit(app: &AppHandle) {
    app.state::<TrayExitState>().request_tray_exit();
}

pub fn hide_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.hide();
    }
}

#[cfg(test)]
mod tests {
    use super::TrayExitState;

    #[test]
    fn tray_exit_request_is_single_use() {
        let state = TrayExitState::default();

        assert!(!state.consume_tray_exit_request());
        state.request_tray_exit();
        assert!(state.consume_tray_exit_request());
        assert!(!state.consume_tray_exit_request());
    }
}
