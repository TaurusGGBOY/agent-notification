use serde::Serialize;

const APP_NOTIFICATION_IDS: [&str; 2] = ["AgentNotify", "com.agentnotify.client"];
const NOTIFICATION_SETTINGS_ROOT: &str =
    r"HKCU\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings";

#[derive(Debug, Clone, Copy, Serialize)]
pub struct WindowsNotificationStatus {
    pub enabled: bool,
}

#[tauri::command]
pub fn windows_notification_status() -> WindowsNotificationStatus {
    WindowsNotificationStatus {
        enabled: query_windows_notification_enabled().unwrap_or(false),
    }
}

#[tauri::command]
pub fn open_windows_notification_settings() -> Result<(), String> {
    open_notification_settings()
}

pub fn parse_notification_enabled(output: &str) -> Option<bool> {
    output.lines().find_map(|line| {
        let mut parts = line.split_whitespace();
        let name = parts.next()?;
        if !name.eq_ignore_ascii_case("Enabled") {
            return None;
        }
        let _kind = parts.next()?;
        let value = parts.next()?;
        parse_registry_dword(value)
    })
}

fn parse_registry_dword(value: &str) -> Option<bool> {
    let parsed = if let Some(hex) = value.strip_prefix("0x").or_else(|| value.strip_prefix("0X")) {
        u32::from_str_radix(hex, 16).ok()?
    } else {
        value.parse::<u32>().ok()?
    };
    Some(parsed != 0)
}

#[cfg(windows)]
fn query_windows_notification_enabled() -> Option<bool> {
    use std::process::Command;

    for app_id in APP_NOTIFICATION_IDS {
        let key = format!(r"{NOTIFICATION_SETTINGS_ROOT}\{app_id}");
        let output = Command::new("reg")
            .args(["query", &key, "/v", "Enabled"])
            .output()
            .ok()?;
        if !output.status.success() {
            continue;
        }
        let text = String::from_utf8_lossy(&output.stdout);
        if let Some(enabled) = parse_notification_enabled(&text) {
            return Some(enabled);
        }
    }
    None
}

#[cfg(not(windows))]
fn query_windows_notification_enabled() -> Option<bool> {
    let _ = APP_NOTIFICATION_IDS;
    let _ = NOTIFICATION_SETTINGS_ROOT;
    None
}

#[cfg(windows)]
fn open_notification_settings() -> Result<(), String> {
    std::process::Command::new("explorer.exe")
        .arg("ms-settings:notifications")
        .spawn()
        .map_err(|err| format!("failed to open Windows notification settings: {err}"))?;
    Ok(())
}

#[cfg(not(windows))]
fn open_notification_settings() -> Result<(), String> {
    Err("Windows notification settings are only available on Windows".to_string())
}

#[cfg(test)]
mod tests {
    use super::{parse_notification_enabled, parse_registry_dword};

    #[test]
    fn parses_enabled_registry_value_as_on() {
        let output = r#"
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\AgentNotify
    Enabled    REG_DWORD    0x1
"#;

        assert_eq!(parse_notification_enabled(output), Some(true));
    }

    #[test]
    fn parses_disabled_registry_value_as_off() {
        let output = r#"
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\AgentNotify
    Enabled    REG_DWORD    0x0
"#;

        assert_eq!(parse_notification_enabled(output), Some(false));
    }

    #[test]
    fn treats_missing_registry_value_as_unknown() {
        let output = r#"
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\AgentNotify
"#;

        assert_eq!(parse_notification_enabled(output), None);
    }

    #[test]
    fn parses_decimal_registry_value() {
        assert_eq!(parse_registry_dword("1"), Some(true));
        assert_eq!(parse_registry_dword("0"), Some(false));
    }
}
