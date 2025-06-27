use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, ContextType, LogLevel};
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(GoogleMetadataIdentityTokenRoot {
            config: None,
        })
    });
}}

const SHARED_DATA_TOKEN_KEY: &str = "proxy-wasm-google-metadata-identity-token-identity-token";
const SHARED_DATA_TOKEN_ISSUED_AT_KEY: &str =
    "proxy-wasm-google-metadata-identity-token-identity-token-issued-at";
const DEFAULT_TOKEN_CACHE_DURATION: u64 = 1800; // 30 minutes

#[derive(Serialize, Deserialize, Clone)]
struct Config {
    audience: String,
    metadata_server_cluster: String,
    #[serde(default)]
    token_cache_duration: Option<u64>,
    #[serde(default)]
    original_authorization_propagation_header: Option<String>,
}

struct GoogleMetadataIdentityTokenRoot {
    config: Option<Config>,
}

impl Context for GoogleMetadataIdentityTokenRoot {}

impl RootContext for GoogleMetadataIdentityTokenRoot {
    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }

    fn on_configure(&mut self, _: usize) -> bool {
        if let Some(config_bytes) = self.get_plugin_configuration() {
            match serde_json::from_slice::<Config>(&config_bytes) {
                Ok(mut config) => {
                    if config.audience.is_empty() {
                        proxy_wasm::hostcalls::log(
                            LogLevel::Error,
                            "the plugin configuration does not include `audience`",
                        )
                        .unwrap();
                        return false;
                    }

                    if config.metadata_server_cluster.is_empty() {
                        proxy_wasm::hostcalls::log(
                            LogLevel::Error,
                            "the plugin configuration does not include `metadata_server_cluster`",
                        )
                        .unwrap();
                        return false;
                    }

                    if config.token_cache_duration.is_none()
                        || config.token_cache_duration == Some(0)
                    {
                        config.token_cache_duration = Some(DEFAULT_TOKEN_CACHE_DURATION);
                    }

                    self.config = Some(config);
                }
                Err(e) => {
                    proxy_wasm::hostcalls::log(
                        LogLevel::Error,
                        &format!("failed to parse plugin configuration: {}", e),
                    )
                    .unwrap();
                    return false;
                }
            }
        } else {
            proxy_wasm::hostcalls::log(LogLevel::Error, "the plugin configuration is not found")
                .unwrap();
            return false;
        }
        true
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        self.config.as_ref().map(|config| {
            Box::new(GoogleMetadataIdentityToken {
                config: config.clone(),
            }) as Box<dyn HttpContext>
        })
    }
}

struct GoogleMetadataIdentityToken {
    config: Config,
}

impl Context for GoogleMetadataIdentityToken {
    fn on_http_call_response(
        &mut self,
        _token_id: u32,
        _num_headers: usize,
        body_size: usize,
        _num_trailers: usize,
    ) {
        let headers = self.get_http_call_response_headers();
        let status = headers
            .iter()
            .find(|(name, _)| name == ":status")
            .map(|(_, value)| value.as_str());

        if status.is_none() {
            self.send_error_response(
                "failed to get the http response status from the metadata server",
            );
            return;
        }

        let status = status.unwrap();

        match self.get_http_call_response_body(0, body_size) {
            Some(body) => {
                if status != "200" {
                    self.send_error_response(&format!(
                        "failed to call the metadata server, status:{}, body:{}",
                        status,
                        String::from_utf8_lossy(&body)
                    ));
                    return;
                }

                if let Err(e) = self.set_shared_data(SHARED_DATA_TOKEN_KEY, Some(&body), None) {
                    self.send_error_response(&format!(
                        "failed to set the identity token to the shared data: {:?}",
                        e
                    ));
                    return;
                }

                let now = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap()
                    .as_secs();
                let iat_bytes = now.to_le_bytes();

                if let Err(e) =
                    self.set_shared_data(SHARED_DATA_TOKEN_ISSUED_AT_KEY, Some(&iat_bytes), None)
                {
                    self.send_error_response(&format!("failed to set the issued at for the identity token to the shared data: {:?}", e));
                    return;
                }

                if let Ok(token) = std::str::from_utf8(&body) {
                    self.set_http_request_header(
                        "authorization",
                        Some(&format!("Bearer {}", token)),
                    );
                    self.resume_http_request();
                } else {
                    self.send_error_response("failed to parse identity token as UTF-8");
                }
            }
            None => {
                self.send_error_response(
                    "failed to get the http response from the metadata server",
                );
            }
        }
    }
}

impl HttpContext for GoogleMetadataIdentityToken {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        if let Some(header) = &self.config.original_authorization_propagation_header {
            if !header.is_empty() {
                if let Some(original_auth) = self.get_http_request_header("authorization") {
                    self.set_http_request_header(header, Some(&original_auth));
                }
            }
        }

        let token_result = self.get_shared_data(SHARED_DATA_TOKEN_KEY);
        let iat_result = self.get_shared_data(SHARED_DATA_TOKEN_ISSUED_AT_KEY);

        if let (Some(token_bytes), Some(_)) = &token_result {
            if let (Some(iat_bytes), Some(_)) = &iat_result {
                if iat_bytes.len() >= 8 {
                    let iat = u64::from_le_bytes(iat_bytes[..8].try_into().unwrap());
                    let now = SystemTime::now()
                        .duration_since(UNIX_EPOCH)
                        .unwrap()
                        .as_secs();

                    if now - iat
                        < self
                            .config
                            .token_cache_duration
                            .unwrap_or(DEFAULT_TOKEN_CACHE_DURATION)
                    {
                        if let Ok(token) = String::from_utf8(token_bytes.clone()) {
                            self.set_http_request_header(
                                "authorization",
                                Some(&format!("Bearer {}", token)),
                            );
                            return Action::Continue;
                        }
                    }
                }
            }
        }

        let path = format!(
            "/computeMetadata/v1/instance/service-accounts/default/identity?audience={}",
            self.config.audience
        );

        let headers = vec![
            (":path", path.as_str()),
            (":method", "GET"),
            (":authority", "metadata.google.internal"),
            (":scheme", "http"),
            ("Metadata-Flavor", "Google"),
        ];

        match self.dispatch_http_call(
            &self.config.metadata_server_cluster,
            headers,
            None,
            vec![],
            std::time::Duration::from_millis(5000),
        ) {
            Ok(_) => Action::Pause,
            Err(e) => {
                self.send_error_response(&format!(
                    "failed to dispatch the http call for the metadata server: {:?}",
                    e
                ));
                Action::Pause
            }
        }
    }
}

impl GoogleMetadataIdentityToken {
    fn send_error_response(&self, message: &str) {
        proxy_wasm::hostcalls::log(LogLevel::Error, message).unwrap();
        self.send_http_response(
            500,
            vec![("content-type", "application/json")],
            Some(b"{\"error\": \"internal server error\"}"),
        );
    }
}
