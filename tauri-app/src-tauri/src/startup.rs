use serde::Serialize;
#[cfg(windows)]
use std::os::windows::process::CommandExt;
use std::path::{Path, PathBuf};

const APP_LABEL: &str = "com.agentnotify.client";
const APP_NAME: &str = "AgentNotify";
#[cfg(windows)]
const CREATE_NO_WINDOW: u32 = 0x0800_0000;
const WINDOWS_RUN_KEY: &str = r"HKCU\Software\Microsoft\Windows\CurrentVersion\Run";

#[derive(Debug, Clone, Copy, Serialize)]
pub struct StartupStatus {
    pub enabled: bool,
    pub supported: bool,
}

#[tauri::command]
pub fn startup_status() -> StartupStatus {
    let supported = cfg!(any(windows, target_os = "macos"));
    StartupStatus {
        enabled: supported && query_startup_enabled().unwrap_or(false),
        supported,
    }
}

#[tauri::command]
pub fn set_startup_enabled(enabled: bool) -> Result<StartupStatus, String> {
    set_startup_enabled_impl(enabled)?;
    Ok(startup_status())
}

#[cfg(windows)]
fn query_startup_enabled() -> Option<bool> {
    let output = hidden_windows_command("reg")
        .args(["query", WINDOWS_RUN_KEY, "/v", APP_NAME])
        .output()
        .ok()?;
    if !output.status.success() {
        return Some(false);
    }
    let text = String::from_utf8_lossy(&output.stdout);
    Some(text.lines().any(|line| line.contains(APP_NAME)))
}

#[cfg(target_os = "macos")]
fn query_startup_enabled() -> Option<bool> {
    let plist_path = macos_launch_agent_path()?;
    let current_exe = std::env::current_exe().ok()?;
    let plist = std::fs::read_to_string(plist_path).ok()?;
    Some(plist.contains(&xml_escape(&current_exe.to_string_lossy())))
}

#[cfg(not(any(windows, target_os = "macos")))]
fn query_startup_enabled() -> Option<bool> {
    None
}

#[cfg(windows)]
fn set_startup_enabled_impl(enabled: bool) -> Result<(), String> {
    if enabled {
        let exe = std::env::current_exe().map_err(|err| format!("failed to resolve current executable: {err}"))?;
        let args = windows_run_set_args(&exe);
        let refs = args.iter().map(String::as_str).collect::<Vec<_>>();
        let status = hidden_windows_command("reg")
            .args(refs)
            .status()
            .map_err(|err| format!("failed to update Windows startup setting: {err}"))?;
        if status.success() {
            return Ok(());
        }
        return Err("failed to enable Windows startup setting".to_string());
    }

    let status = hidden_windows_command("reg")
        .args(["delete", WINDOWS_RUN_KEY, "/v", APP_NAME, "/f"])
        .status()
        .map_err(|err| format!("failed to remove Windows startup setting: {err}"))?;
    if status.success() {
        Ok(())
    } else {
        Err("failed to disable Windows startup setting".to_string())
    }
}

#[cfg(target_os = "macos")]
fn set_startup_enabled_impl(enabled: bool) -> Result<(), String> {
    let plist_path = macos_launch_agent_path()
        .ok_or_else(|| "failed to resolve macOS LaunchAgents directory".to_string())?;
    if enabled {
        let exe = std::env::current_exe().map_err(|err| format!("failed to resolve current executable: {err}"))?;
        let parent = plist_path
            .parent()
            .ok_or_else(|| "failed to resolve macOS LaunchAgents directory".to_string())?;
        std::fs::create_dir_all(parent)
            .map_err(|err| format!("failed to create macOS LaunchAgents directory: {err}"))?;
        std::fs::write(&plist_path, macos_launch_agent_plist(&exe))
            .map_err(|err| format!("failed to write macOS LaunchAgent: {err}"))?;
        return Ok(());
    }

    match std::fs::remove_file(&plist_path) {
        Ok(()) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(err) => Err(format!("failed to remove macOS LaunchAgent: {err}")),
    }
}

#[cfg(not(any(windows, target_os = "macos")))]
fn set_startup_enabled_impl(_enabled: bool) -> Result<(), String> {
    Err("startup settings are only available on Windows and macOS".to_string())
}

#[cfg(windows)]
fn hidden_windows_command(program: &str) -> std::process::Command {
    let mut cmd = std::process::Command::new(program);
    cmd.creation_flags(CREATE_NO_WINDOW);
    cmd
}

#[cfg_attr(not(any(windows, test)), allow(dead_code))]
fn windows_run_set_args(exe: &Path) -> Vec<String> {
    vec![
        "add".to_string(),
        WINDOWS_RUN_KEY.to_string(),
        "/v".to_string(),
        APP_NAME.to_string(),
        "/t".to_string(),
        "REG_SZ".to_string(),
        "/d".to_string(),
        format!(r#""{}""#, exe.display()),
        "/f".to_string(),
    ]
}

#[cfg(target_os = "macos")]
fn macos_launch_agent_path() -> Option<PathBuf> {
    Some(
        PathBuf::from(std::env::var_os("HOME")?)
            .join("Library")
            .join("LaunchAgents")
            .join(format!("{APP_LABEL}.plist")),
    )
}

#[cfg_attr(not(any(target_os = "macos", test)), allow(dead_code))]
fn macos_launch_agent_plist(exe: &Path) -> String {
    format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>{APP_LABEL}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
"#,
        xml_escape(&exe.to_string_lossy())
    )
}

#[cfg_attr(not(any(target_os = "macos", test)), allow(dead_code))]
fn xml_escape(value: &str) -> String {
    value
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
}

#[cfg(test)]
mod tests {
    use super::{macos_launch_agent_plist, windows_run_set_args};
    use std::path::Path;

    #[test]
    fn macos_launch_agent_plist_runs_current_app_at_login() {
        let plist = macos_launch_agent_plist(Path::new("/Applications/AgentNotify.app/Contents/MacOS/AgentNotify"));

        assert!(plist.contains("<key>Label</key>"));
        assert!(plist.contains("<string>com.agentnotify.client</string>"));
        assert!(plist.contains("<key>RunAtLoad</key>"));
        assert!(plist.contains("<true/>"));
        assert!(plist.contains("/Applications/AgentNotify.app/Contents/MacOS/AgentNotify"));
    }

    #[test]
    fn macos_launch_agent_plist_escapes_xml_paths() {
        let plist = macos_launch_agent_plist(Path::new("/Applications/Agent & Notify.app/Contents/MacOS/Agent<Notify>"));

        assert!(plist.contains("Agent &amp; Notify.app"));
        assert!(plist.contains("Agent&lt;Notify&gt;"));
    }

    #[test]
    fn windows_run_set_args_registers_current_executable() {
        let args = windows_run_set_args(Path::new(r"C:\Program Files\AgentNotify\AgentNotify.exe"));

        assert_eq!(
            args,
            vec![
                "add",
                r"HKCU\Software\Microsoft\Windows\CurrentVersion\Run",
                "/v",
                "AgentNotify",
                "/t",
                "REG_SZ",
                "/d",
                r#""C:\Program Files\AgentNotify\AgentNotify.exe""#,
                "/f",
            ],
        );
    }
}
