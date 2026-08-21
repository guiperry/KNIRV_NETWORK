//! KNIRV's official asynchronous Rust SDK.
//!
//! The crate mirrors the Transaction, Gateway, unified-service, controller-wallet,
//! and canonical direct-signing APIs supplied by the Go and TypeScript SDKs.

pub mod client;
pub mod crypto;
pub mod error;
pub mod gateway;
pub mod governance;
pub mod oracled;
pub mod services;
pub mod signing;
pub mod transaction;
pub mod transmission;
pub mod types;
pub mod wallet;

pub use client::{ClientConfig, HttpClient, Network, NetworkInfo, RetryConfig};
pub use error::{Error, Result};
pub use gateway::GatewayClient;
pub use governance::GovernanceClient;
pub use oracled::OracleClient;
pub use services::*;
pub use transaction::TransactionClient;
pub use transmission::*;
pub use wallet::KnirvWallet;

/// A unified client for all KNIRV HTTP services.
#[derive(Clone, Debug)]
pub struct KnirvClient {
    pub transaction: TransactionClient,
    pub gateway: GatewayClient,
    pub wallet: KnirvWallet,
    pub oracled: OracleClient,
    pub governance: GovernanceClient,
    pub crypto: crypto::CryptoService,
    pub badges: BadgeService,
    pub dve: DVEService,
    pub treasury: TreasuryService,
    pub agents: AgentService,
    /// KNIRVROUTER connectivity proofs, routes, and network statistics.
    pub network: NetworkService,
    pub factuality: FactualityService,
    pub health: HealthService,
    pub config: ConfigService,
    pub transmission: TransmissionClient,
    /// Static configuration for the selected KNIRV environment.
    pub network_info: NetworkInfo,
}

impl KnirvClient {
    /// Creates a client for a KNIRV environment. Custom service URLs override
    /// the selected network's defaults.
    pub fn new(config: ClientConfig) -> Result<Self> {
        let mut network = NetworkInfo::for_network(config.network);
        let transaction_url = config
            .transaction_url
            .clone()
            .unwrap_or_else(|| network.services.chain.clone());
        let gateway_url = config
            .gateway_url
            .clone()
            .unwrap_or_else(|| network.services.gateway.clone());
        let controller_url = config
            .controller_url
            .clone()
            .unwrap_or_else(|| network.services.controller.clone());
        network.services.chain = transaction_url.clone();
        network.services.gateway = gateway_url.clone();
        network.services.controller = controller_url.clone();
        Ok(Self {
            transaction: TransactionClient::new(HttpClient::new(transaction_url, config.clone())?),
            gateway: GatewayClient::new(HttpClient::new(&gateway_url, config.clone())?),
            wallet: KnirvWallet::new(HttpClient::new(&controller_url, config.clone())?),
            oracled: OracleClient::new_with_http(HttpClient::new(
                &network.services.oracle,
                config.clone(),
            )?),
            governance: GovernanceClient::new_with_http(
                HttpClient::new(&network.services.oracle, config.clone())?,
                HttpClient::new(&controller_url, config.clone())?,
            ),
            crypto: crypto::CryptoService,
            badges: BadgeService::new(HttpClient::new(&network.services.oracle, config.clone())?),
            dve: DVEService::new(HttpClient::new(&network.services.nexus, config.clone())?),
            treasury: TreasuryService::new(HttpClient::new(
                &network.services.oracle,
                config.clone(),
            )?),
            agents: AgentService::new(HttpClient::new(&controller_url, config.clone())?),
            network: NetworkService::new(HttpClient::new(
                &network.services.router,
                config.clone(),
            )?),
            factuality: FactualityService::new(HttpClient::new(&controller_url, config.clone())?),
            health: HealthService::new(HttpClient::new(&gateway_url, config.clone())?),
            config: ConfigService::new(HttpClient::new(&gateway_url, config.clone())?),
            transmission: TransmissionClient::new(HttpClient::new(
                &network.services.router,
                config,
            )?),
            network_info: network,
        })
    }

    /// A successful health response means the selected KNIRV network is reachable.
    pub async fn is_connected(&self) -> bool {
        self.transaction.health().await.is_ok()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn unified_client_wires_all_service_clients() {
        let sdk = KnirvClient::new(ClientConfig {
            network: Network::LocalTestnet,
            ..Default::default()
        })
        .unwrap();
        assert_eq!(sdk.network_info.chain_id, "knirv-local-testnet");
        assert_eq!(
            sdk.crypto.sha256_string("knirv"),
            crypto::sha256_string("knirv")
        );
        let _ = (
            &sdk.oracled,
            &sdk.governance,
            &sdk.badges,
            &sdk.dve,
            &sdk.treasury,
            &sdk.agents,
            &sdk.network,
            &sdk.factuality,
            &sdk.health,
            &sdk.config,
            &sdk.transmission,
        );
    }
}
