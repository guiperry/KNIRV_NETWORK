use crate::error::{Error, Result};
use crate::http_transport::{HttpRequest, HttpTransport, TransportErrorKind};
use serde::de::DeserializeOwned;
use serde_json::Value;
use std::{collections::BTreeMap, sync::Arc, time::Duration};
use url::Url;

const USER_AGENT: &str = concat!("KNIRV-Rust-SDK/", env!("CARGO_PKG_VERSION"));

/// The HTTP methods the SDK uses.
#[allow(clippy::upper_case_acronyms)]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Method {
    GET,
    POST,
    PUT,
    DELETE,
}
impl Method {
    pub fn as_str(self) -> &'static str {
        match self {
            Method::GET => "GET",
            Method::POST => "POST",
            Method::PUT => "PUT",
            Method::DELETE => "DELETE",
        }
    }
}

#[derive(Clone, Copy, Debug, Default, Eq, PartialEq)]
pub enum Network {
    PublicTestnet,
    #[default]
    PublicProduction,
    LocalTestnet,
    LocalProduction,
}

#[derive(Clone, Debug)]
pub struct Services {
    pub controller: String,
    pub router: String,
    pub graph: String,
    pub chain: String,
    pub oracle: String,
    pub nexus: String,
    pub gateway: String,
    pub backend: String,
}
#[derive(Clone, Debug)]
pub struct NetworkInfo {
    pub chain_id: String,
    pub name: String,
    pub rpc_url: String,
    pub services: Services,
}
impl NetworkInfo {
    pub fn for_network(network: Network) -> Self {
        let (chain_id, name, rpc, services) = match network {
            Network::PublicProduction => (
                "knirv-1",
                "KNIRV Production Network",
                "https://gateway.knirv.com/rpc",
                Services {
                    controller: "https://gateway.knirv.com".into(),
                    router: "https://gateway.knirv.com".into(),
                    graph: "https://gateway.knirv.com".into(),
                    chain: "https://gateway.knirv.com".into(),
                    oracle: "https://gateway.knirv.com".into(),
                    nexus: "https://gateway.knirv.com".into(),
                    gateway: "https://gateway.knirv.com".into(),
                    backend: "https://gateway.knirv.com".into(),
                },
            ),
            Network::PublicTestnet => (
                "knirv-testnet-1",
                "KNIRV Testnet",
                "https://testnet-gateway.knirv.com/rpc",
                Services {
                    controller: "https://testnet-gateway.knirv.com".into(),
                    router: "https://testnet-gateway.knirv.com".into(),
                    graph: "https://testnet-gateway.knirv.com".into(),
                    chain: "https://testnet-gateway.knirv.com".into(),
                    oracle: "https://testnet-gateway.knirv.com".into(),
                    nexus: "https://testnet-gateway.knirv.com".into(),
                    gateway: "https://testnet-gateway.knirv.com".into(),
                    backend: "https://testnet-gateway.knirv.com".into(),
                },
            ),
            Network::LocalTestnet => (
                "knirv-local-testnet",
                "KNIRV Local Testnet",
                "http://localhost:8080/rpc",
                Services {
                    controller: "http://localhost:8080".into(),
                    router: "http://localhost:8080".into(),
                    graph: "http://localhost:8080".into(),
                    chain: "http://localhost:8080".into(),
                    oracle: "http://localhost:8080".into(),
                    nexus: "http://localhost:8080".into(),
                    gateway: "http://localhost:8080".into(),
                    backend: "http://localhost:8082".into(),
                },
            ),
            Network::LocalProduction => (
                "knirv-local-production",
                "KNIRV Local Production",
                "http://localhost:26657",
                Services {
                    controller: "http://localhost:3000".into(),
                    router: "http://localhost:8085".into(),
                    graph: "http://localhost:8081".into(),
                    chain: "http://localhost:8080".into(),
                    oracle: "http://localhost:8086".into(),
                    nexus: "http://localhost:8090".into(),
                    gateway: "http://localhost:8087".into(),
                    backend: "http://localhost:8082".into(),
                },
            ),
        };
        Self {
            chain_id: chain_id.into(),
            name: name.into(),
            rpc_url: rpc.into(),
            services,
        }
    }
    pub fn service_url(&self, service: &str) -> &str {
        match service {
            "controller" => &self.services.controller,
            "router" => &self.services.router,
            "graph" => &self.services.graph,
            "chain" => &self.services.chain,
            "oracle" => &self.services.oracle,
            "nexus" => &self.services.nexus,
            "gateway" => &self.services.gateway,
            "backend" => &self.services.backend,
            _ => &self.services.gateway,
        }
    }
}

#[derive(Clone, Debug)]
pub struct RetryConfig {
    pub max_retries: u32,
    pub initial_backoff: Duration,
}
impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            max_retries: 2,
            initial_backoff: Duration::from_millis(250),
        }
    }
}
#[derive(Clone, Debug)]
pub struct ClientConfig {
    pub network: Network,
    pub api_key: Option<String>,
    pub timeout: Duration,
    pub retry: RetryConfig,
    pub transaction_url: Option<String>,
    pub gateway_url: Option<String>,
    pub controller_url: Option<String>,
    pub backend_url: Option<String>,
    pub headers: BTreeMap<String, String>,
    pub idempotency_key: Option<String>,
    /// Moves the bytes. `None` uses reqwest; WebAssembly hosts supply their own
    /// (see `http_transport`). Retries, failover and error mapping stay here.
    pub transport: Option<Arc<dyn HttpTransport>>,
}
impl Default for ClientConfig {
    fn default() -> Self {
        Self {
            network: Network::default(),
            api_key: std::env::var("KNIRVCHAIN_TRANSACTION_SDK_API_KEY").ok(),
            timeout: Duration::from_secs(60),
            retry: RetryConfig::default(),
            transaction_url: std::env::var("KNIRVCHAIN_TRANSACTION_SDK_BASE_URL").ok(),
            gateway_url: std::env::var("KNIRVGATEWAY_BASE_URL").ok(),
            controller_url: None,
            backend_url: std::env::var("KNIRVSERVER_BACKEND_URL").ok(),
            headers: BTreeMap::new(),
            idempotency_key: None,
            transport: None,
        }
    }
}

#[derive(Clone, Debug)]
pub struct HttpClient {
    base_url: Url,
    base_urls: Vec<Url>,
    transport: Arc<dyn HttpTransport>,
    config: ClientConfig,
}
impl HttpClient {
    pub fn new(base_url: impl AsRef<str>, config: ClientConfig) -> Result<Self> {
        let base_url = Url::parse(base_url.as_ref())
            .map_err(|e| Error::Configuration(format!("invalid base URL: {e}")))?;
        let transport: Arc<dyn HttpTransport> = match &config.transport {
            Some(transport) => transport.clone(),
            #[cfg(feature = "http-reqwest")]
            None => Arc::new(crate::http_transport::ReqwestTransport::new()?),
            #[cfg(not(feature = "http-reqwest"))]
            None => {
                return Err(Error::Configuration(
                    "no HTTP transport: this build has none built in, so the host must supply one".into(),
                ))
            }
        };
        let base_urls = gateway_candidates(&base_url);
        Ok(Self {
            base_url,
            base_urls,
            transport,
            config,
        })
    }
    /// Waits using the transport's own timer.
    pub async fn sleep(&self, duration: Duration) {
        self.transport.sleep(duration).await
    }
    pub fn base_url(&self) -> &Url {
        &self.base_url
    }
    /// Ordered request targets. Canonical KNIRV gateways receive automatic
    /// failover in the same order as the TypeScript `BaseService`.
    pub fn base_urls(&self) -> &[Url] {
        &self.base_urls
    }
    pub async fn get<T: DeserializeOwned>(
        &self,
        path: &str,
        query: &[(&str, String)],
    ) -> Result<T> {
        self.request(Method::GET, path, query, None).await
    }
    pub async fn delete<T: DeserializeOwned>(&self, path: &str) -> Result<T> {
        self.request(Method::DELETE, path, &[], None).await
    }
    pub async fn post<T: DeserializeOwned>(&self, path: &str, body: Value) -> Result<T> {
        self.request(Method::POST, path, &[], Some(body)).await
    }
    pub async fn put<T: DeserializeOwned>(&self, path: &str, body: Value) -> Result<T> {
        self.request(Method::PUT, path, &[], Some(body)).await
    }
    pub async fn request<T: DeserializeOwned>(
        &self,
        method: Method,
        path: &str,
        query: &[(&str, String)],
        body: Option<Value>,
    ) -> Result<T> {
        let mut last_error = None;
        for base_url in &self.base_urls {
            match self
                .request_from(base_url, method.clone(), path, query, body.clone())
                .await
            {
                Ok(value) => return Ok(value),
                Err(error) if should_fail_over(&error) => last_error = Some(error),
                Err(error) => return Err(error),
            }
        }
        Err(last_error.expect("an HTTP client has at least one base URL"))
    }

    async fn request_from<T: DeserializeOwned>(
        &self,
        base_url: &Url,
        method: Method,
        path: &str,
        query: &[(&str, String)],
        body: Option<Value>,
    ) -> Result<T> {
        let mut url = base_url
            .join(path.trim_start_matches('/'))
            .map_err(|e| Error::Configuration(e.to_string()))?;
        if !query.is_empty() {
            url.query_pairs_mut()
                .extend_pairs(query.iter().map(|(k, v)| (*k, v.as_str())));
        }
        let body_bytes = match &body {
            Some(body) => serde_json::to_vec(body)?,
            None => Vec::new(),
        };
        for attempt in 0..=self.config.retry.max_retries {
            let mut headers = BTreeMap::new();
            headers.insert("User-Agent".to_owned(), USER_AGENT.to_owned());
            if let Some(idempotency_key) = &self.config.idempotency_key {
                headers.insert("Idempotency-Key".to_owned(), idempotency_key.clone());
            }
            for (name, value) in &self.config.headers {
                headers.insert(name.clone(), value.clone());
            }
            if let Some(key) = &self.config.api_key {
                headers.insert("Authorization".to_owned(), format!("Bearer {key}"));
            }
            if body.is_some() {
                headers.insert("Content-Type".to_owned(), "application/json".to_owned());
            }
            let request = HttpRequest {
                method: method.as_str().to_owned(),
                url: url.to_string(),
                headers,
                body: body_bytes.clone(),
                timeout_ms: self.config.timeout.as_millis() as u64,
            };
            match self.transport.send(request).await {
                Ok(response) if (200..300).contains(&response.status) => {
                    return serde_json::from_slice(&response.body).map_err(Error::from)
                }
                Ok(response) => {
                    let status = response.status;
                    let message = String::from_utf8_lossy(&response.body).into_owned();
                    let retryable = (500..600).contains(&status) || status == 408 || status == 409;
                    if retryable && attempt < self.config.retry.max_retries {
                        if let Some(seconds) = response
                            .headers
                            .get("retry-after")
                            .and_then(|v| v.trim().parse::<u64>().ok())
                        {
                            self.transport.sleep(Duration::from_secs(seconds)).await;
                            continue;
                        }
                        self.transport
                            .sleep(self.config.retry.initial_backoff * 2u32.pow(attempt))
                            .await;
                        continue;
                    }
                    if status == 409 {
                        return Err(Error::IdempotencyConflict);
                    }
                    return Err(Error::Api { status, message });
                }
                Err(error) => {
                    // A host abort is a decision, not a fault: do not retry it.
                    if error.kind == TransportErrorKind::Aborted
                        || attempt == self.config.retry.max_retries
                    {
                        return Err(error.into());
                    }
                }
            }
            self.transport
                .sleep(self.config.retry.initial_backoff * 2u32.pow(attempt))
                .await;
        }
        unreachable!("retry loop always returns")
    }
}

const CANONICAL_GATEWAYS: [&str; 3] = [
    "https://gateway.knirv.com",
    "https://testnet-gateway.knirv.com",
    "http://localhost:8080",
];

fn gateway_candidates(base_url: &Url) -> Vec<Url> {
    let normalized = base_url.as_str().trim_end_matches('/');
    let mut candidates = vec![base_url.clone()];
    // This deliberately mirrors the TypeScript BaseService: only canonical
    // gateway URLs opt into failover; a caller's custom endpoint is respected.
    if CANONICAL_GATEWAYS
        .iter()
        .any(|gateway| *gateway == normalized)
    {
        candidates.extend(
            CANONICAL_GATEWAYS
                .iter()
                .filter(|gateway| **gateway != normalized)
                .filter_map(|gateway| Url::parse(gateway).ok()),
        );
    }
    candidates
}

fn should_fail_over(error: &Error) -> bool {
    matches!(
        error,
        Error::Timeout
            | Error::Transport(_)
            | Error::Api {
                status: 500..=599,
                ..
            }
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn canonical_gateways_receive_ordered_failover_candidates() {
        let client = HttpClient::new("https://gateway.knirv.com", ClientConfig::default()).unwrap();
        let urls: Vec<_> = client.base_urls().iter().map(Url::as_str).collect();
        assert_eq!(
            urls,
            [
                "https://gateway.knirv.com/",
                "https://testnet-gateway.knirv.com/",
                "http://localhost:8080/",
            ]
        );
    }

    #[test]
    fn custom_endpoints_do_not_receive_unexpected_failover_targets() {
        let client =
            HttpClient::new("https://gateway.example.test", ClientConfig::default()).unwrap();
        assert_eq!(client.base_urls().len(), 1);
    }

    #[test]
    fn public_networks_use_their_canonical_gateway() {
        let production = NetworkInfo::for_network(Network::PublicProduction);
        let testnet = NetworkInfo::for_network(Network::PublicTestnet);
        assert_eq!(production.services.gateway, "https://gateway.knirv.com");
        assert_eq!(
            testnet.services.gateway,
            "https://testnet-gateway.knirv.com"
        );
    }

    // --- the transport seam: retries, failover and error mapping live here ---

    use crate::http_transport::{
        block_on, BoxFuture, HttpResponse, JsonTransport, TransportError,
    };
    use std::sync::Mutex;

    /// Answers from a script and records every request it sees.
    #[derive(Debug, Default)]
    struct Fake {
        script: Mutex<Vec<std::result::Result<HttpResponse, TransportError>>>,
        seen: Mutex<Vec<HttpRequest>>,
        slept: Mutex<Vec<Duration>>,
    }
    impl HttpTransport for Fake {
        fn send(
            &self,
            request: HttpRequest,
        ) -> BoxFuture<'_, std::result::Result<HttpResponse, TransportError>> {
            self.seen.lock().unwrap().push(request);
            let next = self.script.lock().unwrap().remove(0);
            Box::pin(std::future::ready(next))
        }
        fn sleep(&self, duration: Duration) -> BoxFuture<'_, ()> {
            self.slept.lock().unwrap().push(duration);
            Box::pin(std::future::ready(()))
        }
    }

    fn reply(status: u16, body: &str) -> std::result::Result<HttpResponse, TransportError> {
        Ok(HttpResponse {
            status,
            headers: BTreeMap::new(),
            body: body.as_bytes().to_vec(),
        })
    }
    fn fail(kind: TransportErrorKind) -> std::result::Result<HttpResponse, TransportError> {
        Err(TransportError {
            kind,
            message: "boom".into(),
        })
    }
    fn client(
        base: &str,
        script: Vec<std::result::Result<HttpResponse, TransportError>>,
    ) -> (HttpClient, Arc<Fake>) {
        let fake = Arc::new(Fake {
            script: Mutex::new(script),
            ..Default::default()
        });
        let config = ClientConfig {
            transport: Some(fake.clone()),
            api_key: Some("k".into()),
            ..Default::default()
        };
        (HttpClient::new(base, config).unwrap(), fake)
    }

    #[test]
    fn requests_carry_auth_query_and_json_body() {
        let (c, fake) = client("https://x.example.test", vec![reply(200, r#"{"ok":true}"#)]);
        let v: Value = block_on(c.request(
            Method::POST,
            "/v1/a",
            &[("q", "a b".into())],
            Some(serde_json::json!({"n": 1})),
        ))
        .unwrap();
        assert_eq!(v["ok"], true);
        let seen = fake.seen.lock().unwrap();
        assert_eq!(seen[0].url, "https://x.example.test/v1/a?q=a+b");
        assert_eq!(seen[0].headers["Authorization"], "Bearer k");
        assert_eq!(seen[0].headers["Content-Type"], "application/json");
        assert_eq!(seen[0].body, br#"{"n":1}"#);
        assert_eq!(seen[0].timeout_ms, 60_000);
    }

    #[test]
    fn server_errors_are_retried_with_backoff_then_succeed() {
        let (c, fake) = client(
            "https://x.example.test",
            vec![reply(503, "no"), reply(502, "no"), reply(200, "{}")],
        );
        block_on(c.get::<Value>("/a", &[])).unwrap();
        assert_eq!(fake.seen.lock().unwrap().len(), 3);
        assert_eq!(
            *fake.slept.lock().unwrap(),
            [Duration::from_millis(250), Duration::from_millis(500)]
        );
    }

    #[test]
    fn client_errors_are_not_retried_and_keep_status_and_body() {
        let (c, fake) = client("https://x.example.test", vec![reply(404, "gone")]);
        let err = block_on(c.get::<Value>("/a", &[])).unwrap_err();
        assert!(matches!(err, Error::Api { status: 404, ref message } if message == "gone"), "{err}");
        assert_eq!(fake.seen.lock().unwrap().len(), 1);
    }

    #[test]
    fn exhausted_conflicts_map_to_idempotency_conflict() {
        let (c, _) = client(
            "https://x.example.test",
            vec![reply(409, ""), reply(409, ""), reply(409, "")],
        );
        let err = block_on(c.post::<Value>("/a", Value::Null)).unwrap_err();
        assert!(matches!(err, Error::IdempotencyConflict), "{err}");
    }

    #[test]
    fn retry_after_header_wins_over_backoff() {
        let mut limited = HttpResponse {
            status: 503,
            ..Default::default()
        };
        limited.headers.insert("retry-after".into(), "3".into());
        let (c, fake) = client("https://x.example.test", vec![Ok(limited), reply(200, "{}")]);
        block_on(c.get::<Value>("/a", &[])).unwrap();
        assert_eq!(*fake.slept.lock().unwrap(), [Duration::from_secs(3)]);
    }

    #[test]
    fn timeouts_retry_then_map_to_timeout() {
        let (c, fake) = client(
            "https://x.example.test",
            vec![
                fail(TransportErrorKind::Timeout),
                fail(TransportErrorKind::Timeout),
                fail(TransportErrorKind::Timeout),
            ],
        );
        let err = block_on(c.get::<Value>("/a", &[])).unwrap_err();
        assert!(matches!(err, Error::Timeout), "{err}");
        assert_eq!(fake.seen.lock().unwrap().len(), 3);
    }

    #[test]
    fn a_host_abort_is_final() {
        let (c, fake) = client("https://x.example.test", vec![fail(TransportErrorKind::Aborted)]);
        let err = block_on(c.get::<Value>("/a", &[])).unwrap_err();
        assert!(matches!(err, Error::Transport(ref m) if m.starts_with("aborted")), "{err}");
        assert_eq!(fake.seen.lock().unwrap().len(), 1);
    }

    #[test]
    fn canonical_gateways_fail_over_in_order() {
        let (c, fake) = client(
            "https://gateway.knirv.com",
            vec![
                fail(TransportErrorKind::Refused),
                fail(TransportErrorKind::Refused),
                reply(200, "{}"),
            ],
        );
        // the retry budget is per base URL: one attempt each
        let c = HttpClient {
            config: ClientConfig {
                retry: RetryConfig {
                    max_retries: 0,
                    ..Default::default()
                },
                ..c.config.clone()
            },
            ..c
        };
        block_on(c.get::<Value>("/a", &[])).unwrap();
        let urls: Vec<_> = fake.seen.lock().unwrap().iter().map(|r| r.url.clone()).collect();
        assert_eq!(
            urls,
            [
                "https://gateway.knirv.com/a",
                "https://testnet-gateway.knirv.com/a",
                "http://localhost:8080/a"
            ]
        );
    }

    #[test]
    fn json_transport_round_trips_the_host_wire_format() {
        let transport = JsonTransport::new(|request| {
            let request: Value = serde_json::from_slice(request).unwrap();
            assert_eq!(request["method"], "GET");
            assert_eq!(request["body_base64"], "");
            br#"{"status":200,"headers":{"X-A":"b"},"body_base64":"e30="}"#.to_vec()
        });
        let config = ClientConfig {
            transport: Some(Arc::new(transport)),
            ..Default::default()
        };
        let c = HttpClient::new("https://x.example.test", config).unwrap();
        assert_eq!(block_on(c.get::<Value>("/a", &[])).unwrap(), serde_json::json!({}));
    }

    #[test]
    fn json_transport_maps_host_error_kinds() {
        let transport = JsonTransport::new(|_| br#"{"error":{"kind":"dns","message":"nx"}}"#.to_vec());
        let config = ClientConfig {
            transport: Some(Arc::new(transport)),
            retry: RetryConfig {
                max_retries: 0,
                ..Default::default()
            },
            ..Default::default()
        };
        let c = HttpClient::new("https://x.example.test", config).unwrap();
        let err = block_on(c.get::<Value>("/a", &[])).unwrap_err();
        assert!(matches!(err, Error::Transport(ref m) if m == "dns: nx"), "{err}");
    }
}
