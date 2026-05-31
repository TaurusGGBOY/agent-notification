fn main() {
    #[cfg(target_os = "macos")]
    {
        cc::Build::new()
            .file("native/macos_notification.m")
            .flag("-x")
            .flag("objective-c")
            .flag("-fobjc-arc")
            .flag("-Wno-deprecated-declarations")
            .flag("-Wno-unguarded-availability-new")
            .compile("agentnotify_macos_notification");

        println!("cargo:rustc-link-lib=framework=Foundation");
        println!("cargo:rustc-link-lib=framework=Cocoa");
        println!("cargo:rustc-link-lib=framework=UserNotifications");
        println!("cargo:rerun-if-changed=native/macos_notification.m");
    }

    tauri_build::build()
}
