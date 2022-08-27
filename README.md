# proxy-wasm-google-metadata-identity-token

A [proxy-wasm](https://github.com/proxy-wasm/spec) compliant WebAssembly module for making proxies [fetch identity tokens from the Google Cloud Metadata Server](https://cloud.google.com/run/docs/securing/service-identity#fetching_identity_and_access_tokens_using_the_metadata_server).

## Overview

This [proxy-wasm](https://github.com/proxy-wasm/spec) compliant WebAssembly module makes proxies fetch identity tokens from the Google Cloud Metadata Server and attach identity tokens to HTTP requests as an `Authorization` Header with the `Bearer` scheme.

For example, we can make proxies achieve the [Service-to-Service authentication for Google Cloud Run](https://cloud.google.com/run/docs/authenticating/service-to-service) with this WebAssembly module.

This WebAssembly module works similarly to [Envoy native GCP Authentication Filter](https://www.envoyproxy.io/docs/envoy/v1.23.1/configuration/http/http_filters/gcp_authn_filter), but has more control capabilities like propagating original `Authorization` to upstreams.

## Usage

1. Download the latest WebAssembly module binary from the [release page](https://github.com/kauche/proxy-wasm-google-metadata-identity-token/releases).

2. Configure the proxy to use the WebAssembly module like below (this assumes [Envoy](https://www.envoyproxy.io/) as the proxy):

```yaml
listeners:
  - name: example
    filter_chains:
      - filters:
          - name: envoy.filters.network.http_connection_manager
            typed_config:
              # ...
              http_filters:
                - name: envoy.filters.http.wasm
                  typed_config:
                    '@type': type.googleapis.com/udpa.type.v1.TypedStruct
                    type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                    value:
                      config:
                        vm_config:
                          runtime: envoy.wasm.runtime.v8
                          code:
                            local:
                              filename: /etc/envoy/proxy-wasm-google-metadata-identity-token.wasm
                        configuration:
                          "@type": type.googleapis.com/google.protobuf.StringValue
                          value: |
                            {
                              "audience": "https://your-audience-service.a.run.app",
                              "metadata_server_cluster": "google-metadata-server",
                              "token_cache_duration": 1800,
                              "original_authorization_propagation_header": "original-authorization"
                            }
                - name: envoy.filters.http.router
                  typed_config:
                    '@type': type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
# ...

clusters:
  - name: google-metadata-server
    connect_timeout: 5000s
    type: strict_dns
    lb_policy: round_robin
    load_assignment:
      cluster_name: google-metadata-server
      endpoints:
        - lb_endpoints:
            - endpoint:
                address:
                  socket_address:
                    address: metadata.google.internal
                    port_value: 80
```

We can also configure this WebAssembly module for the per Route basis by using the [Composite Filter](https://www.envoyproxy.io/docs/envoy/v1.23.1/configuration/http/http_filters/composite_filter). See the [example `envoy.yaml`](https://github.com/kauche/proxy-wasm-google-metadata-identity-token/blob/main/test/envoy.yaml) for more details.

### Plugin Configurations

- `audience` (Required)
    - The `audience` value for identity tokens.
- `metadata_server_cluster` (Required)
    - Cluster name for the Google Cloud Metadata Server.
- `token_cache_duration` (Optional)
    - How long should this WebAssembly module cache the identity token. The default is 1800 (30 min). This must not be greater than 3600 since identity tokens issued by the Google Metadata Server expires in 3600s.
- `original_authorization_propagation_header` (Optional)
    - If this field is configured, this WebAssembly module will propagate the original Authorization HTTP header to the upstreams by modifying the name of the header to be the value of this field.
