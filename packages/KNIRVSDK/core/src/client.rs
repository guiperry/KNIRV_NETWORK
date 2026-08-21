use crate::error::{Error, Result};
use reqwest::Method;
use serde::de::DeserializeOwned;
use serde_json::Value;
use std::{collections::BTreeMap, time::Duration};
use tokio::time::sleep;
use url::Url;

const USER_AGENT: &str = concat!("KNIRV-Rust-SDK/", env!("CARGO_PKG_VERSION"));

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
                "https://gateway.knirv.network/rpc",
                Services {
                    controller: "https://gateway.knirv.network".into(),
                    router: "https://gateway.knirv.network".into(),
                    graph: "https://gateway.knirv.network".into(),
                    chain: "https://gateway.knirv.network".into(),
                    oracle: "https://gateway.knirv.network".into(),
                    nexus: "https://gateway.knirv.network".into(),
                    gateway: "https://gateway.knirv.network".into(),
                },
            ),
            Network::PublicTestnet => (
                "knirv-testnet-1",
                "KNIRV Testnet",
                "https://testnet-gateway.knirv.network/rpc",
                Services {
                    controller: "https://testnet-gateway.knirv.network".into(),
                    router: "https://testnet-gateway.knirv.network".into(),
                    graph: "https://testnet-gateway.knirv.network".into(),
                    chain: "https://testnet-gateway.knirv.network".into(),
                    oracle: "https://testnet-gateway.knirv.network".into(),
                    nexus: "https://testnet-gateway.knirv.network".into(),
                    gateway: "https://testnet-gateway.knirv.network".into(),
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
    pub headers: BTreeMap<String, String>,
    pub idempotency_key: Option<String>,
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
            headers: BTreeMap::new(),
            idempotency_key: None,
        }
    }
}

#[derive(Clone, Debug)]
pub struct HttpClient {
    base_url: Url,
    base_urls: Vec<Url>,
    client: reqwest::Client,
    config: ClientConfig,
}
impl HttpClient {
    pub fn new(base_url: impl AsRef<str>, config: ClientConfig) -> Result<Self> {
        let base_url = Url::parse(base_url.as_ref())
            .map_err(|e| Error::Configuration(format!("invalid base URL: {e}")))?;
        let client = reqwest::Client::builder().timeout(config.timeout).build()?;
        let base_urls = gateway_candidates(&base_url);
        Ok(Self {
            base_url,
            base_urls,
            client,
            config,
        })
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
        let url = base_url
            .join(path.trim_start_matches('/'))
            .map_err(|e| Error::Configuration(e.to_string()))?;
        for attempt in 0..=self.config.retry.max_retries {
            let mut request = self
                .client
                .request(method.clone(), url.clone())
                .query(query);
            request = request.header("User-Agent", USER_AGENT);
            if let Some(idempotency_key) = &self.config.idempotency_key {
                request = request.header("Idempotency-Key", idempotency_key);
            }
            for (name, value) in &self.config.headers {
                request = request.header(name, value);
            }
            if let Some(key) = &self.config.api_key {
                request = request.bearer_auth(key);
            }
            if let Some(body) = &body {
                request = request.json(body);
            }
            match request.send().await {
                Ok(response) if response.status().is_success() => {
                    return response.json().await.map_err(Error::from)
                }
                Ok(response) => {
                    let status = response.status();
                    let headers = response.headers().clone();
                    let message = response.text().await.unwrap_or_default();
                    let retryable = status.is_server_error()
                        || status == reqwest::StatusCode::REQUEST_TIMEOUT
                        || status == reqwest::StatusCode::CONFLICT;
                    if retryable && attempt < self.config.retry.max_retries {
                        if let Some(retry_after) = headers.get("Retry-After") {
                            if let Ok(retry_str) = retry_after.to_str() {
                                if let Ok(seconds) = retry_str.parse::<u64>() {
                                    sleep(Duration::from_secs(seconds)).await;
                                    continue;
                                }
                            }
                        }
                        sleep(self.config.retry.initial_backoff * 2u32.pow(attempt)).await;
                        continue;
                    }
                    if status == reqwest::StatusCode::CONFLICT {
                        return Err(Error::IdempotencyConflict);
                    }
                    return Err(Error::Api {
                        status: status.as_u16(),
                        message,
                    });
                }
                Err(error) if error.is_timeout() => {
                    if attempt == self.config.retry.max_retries {
                        return Err(Error::Timeout);
                    }
                }
                Err(error) => {
                    if attempt == self.config.retry.max_retries {
                        return Err(Error::Transport(error));
                    }
                }
            }
            sleep(self.config.retry.initial_backoff * 2u32.pow(attempt)).await;
        }
        unreachable!("retry loop always returns")
    }
}

const CANONICAL_GATEWAYS: [&str; 3] = [
    "https://gateway.knirv.network",
    "https://testnet-gateway.knirv.network",
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
        let client =
            HttpClient::new("https://gateway.knirv.network", ClientConfig::default()).unwrap();
        let urls: Vec<_> = client.base_urls().iter().map(Url::as_str).collect();
        assert_eq!(
            urls,
            [
                "https://gateway.knirv.network/",
                "https://testnet-gateway.knirv.network/",
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
        assert_eq!(production.services.gateway, "https://gateway.knirv.network");
        assert_eq!(
            testnet.services.gateway,
            "https://testnet-gateway.knirv.network"
        );
    }
}
