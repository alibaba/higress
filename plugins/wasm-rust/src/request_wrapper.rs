use proxy_wasm::hostcalls;

use crate::internal;

fn get_request_head(head: &str, log_flag: &str) -> String {
    if let Some(value) = internal::get_http_request_header(head) {
        value
    } else {
        hostcalls::log(
            proxy_wasm::types::LogLevel::Error,
            &format!("get request {} failed", log_flag),
        )
        .unwrap();
        String::new()
    }
}

pub fn get_request_scheme() -> String {
    get_request_head(":scheme", "head")
}

pub fn get_request_host() -> String {
    get_request_head(":authority", "host")
}

pub fn get_request_path() -> String {
    get_request_head(":path", "path")
}

pub fn get_request_method() -> String {
    get_request_head(":method", "method")
}

pub fn is_binary_request_body() -> bool {
    if let Some(content_type) = internal::get_http_request_header("content-type") {
        if content_type.contains("octet-stream") || content_type.contains("grpc") {
            return true;
        }
    }
    if let Some(encoding) = internal::get_http_request_header("content-encoding") {
        if !encoding.is_empty() {
            return true;
        }
    }
    false
}

pub fn is_binary_response_body() -> bool {
    if let Some(content_type) = internal::get_http_response_header("content-type") {
        if content_type.contains("octet-stream") || content_type.contains("grpc") {
            return true;
        }
    }
    if let Some(encoding) = internal::get_http_response_header("content-encoding") {
        if !encoding.is_empty() {
            return true;
        }
    }
    false
}

pub fn has_request_body() -> bool {
    let content_type = internal::get_http_request_header("content-type");
    let content_length_str = internal::get_http_request_header("content-length");
    let transfer_encoding = internal::get_http_request_header("transfer-encoding");
    hostcalls::log(
        proxy_wasm::types::LogLevel::Debug,
        &format!(
            "check has request body: content_type:{:?}, content_length_str:{:?}, transfer_encoding:{:?}",
            content_type, content_length_str, transfer_encoding
        )
    ).unwrap();
    if content_type.is_some_and(|x| !x.is_empty()) {
        return true;
    }
    if let Some(cl) = content_length_str {
        if let Ok(content_length) = cl.parse::<i32>() {
            if content_length > 0 {
                return true;
            }
        }
    }
    transfer_encoding.is_some_and(|x| x == "chunked")
}
