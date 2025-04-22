use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, ContextType, LogLevel};
use serde::{Deserialize, Serialize};

#[cfg(not(all(target_arch = "wasm32", target_os = "unknown")))]

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
        proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> { Box::new(GoogleMetadataIdentityTokenRoot{
            config: Config {
            audience: "".to_string(),
            metadata_server_cluster: "".to_string(),
            token_cache_duration: 1800,
            original_authorization_propagation_header: "".to_string(),
        } })
    });
}}

struct GoogleMetadataIdentityTokenRoot {
    config: Config,
}

impl Context for GoogleMetadataIdentityTokenRoot {}

impl RootContext for GoogleMetadataIdentityTokenRoot {
    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }

    fn on_configure(&mut self, _: usize) -> bool {
        if let Some(config_bytes) = self.get_plugin_configuration() {
            self.config = serde_json::from_reader(config_bytes.as_slice()).unwrap();
        }
        true
    }

    fn create_http_context(&self, _: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(GoogleMetadataIdentityToken {
            config: self.config.clone(),
        }))
    }
}

struct GoogleMetadataIdentityToken {
    config: Config,
}

impl Context for GoogleMetadataIdentityToken {}

impl HttpContext for GoogleMetadataIdentityToken {
    fn on_http_request_headers(&mut self, _: usize, _: bool) -> Action {
        Action::Continue
    }
}

#[derive(Serialize, Deserialize, Clone)]
struct Config {
    audience: String,
    metadata_server_cluster: String,
    token_cache_duration: u64,
    original_authorization_propagation_header: String,
}
