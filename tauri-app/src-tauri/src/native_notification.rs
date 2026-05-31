#[cfg(target_os = "macos")]
mod imp {
    use std::ffi::CString;
    use std::os::raw::{c_char, c_int};

    extern "C" {
        fn agentnotify_show_app_notification(
            title: *const c_char,
            body: *const c_char,
            errbuf: *mut c_char,
            errbuflen: c_int,
        ) -> c_int;
    }

    pub fn show(title: &str, body: &str) -> Result<(), String> {
        let title = CString::new(title.replace('\0', "")).map_err(|err| err.to_string())?;
        let body = CString::new(body.replace('\0', "")).map_err(|err| err.to_string())?;
        let mut errbuf = vec![0 as c_char; 512];

        let result = unsafe {
            agentnotify_show_app_notification(
                title.as_ptr(),
                body.as_ptr(),
                errbuf.as_mut_ptr(),
                errbuf.len() as c_int,
            )
        };
        if result == 0 {
            return Ok(());
        }

        let message = unsafe { std::ffi::CStr::from_ptr(errbuf.as_ptr()) }
            .to_string_lossy()
            .trim()
            .to_string();
        if message.is_empty() {
            Err("unknown native notification error".to_string())
        } else {
            Err(message)
        }
    }
}

#[cfg(not(target_os = "macos"))]
mod imp {
    pub fn show(_title: &str, _body: &str) -> Result<(), String> {
        Ok(())
    }
}

pub use imp::show;
