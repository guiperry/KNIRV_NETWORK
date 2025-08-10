# KNIRVTESTNET User Guide

## Getting Started

Before you begin, ensure you have the following prerequisites installed:

* Docker & Docker Compose
* Go 1.23+
* Rust 1.70+
* Node.js 18+
* Python 3.9+

### Prerequisites Checklist

To ensure a smooth setup, please verify that you have the necessary tools installed:

* Docker: `docker --version`
* Docker Compose: `docker-compose --version`
* Go: `go version`
* Rust: `rustc --version`
* Node.js: `node --version`
* Python: `python3 --version`

### Installing Dependencies

If you encounter any issues during the installation process, refer to the troubleshooting section below.

## Setting up KNIRVTESTNET

1. **Clone the repository:** Obtain the KNIRVTESTNET code.
2. **Navigate to the directory:** Open your terminal and navigate to the `KNIRVTESTNET` directory.
3. **Build all components:** Run `./scripts/build-all.sh`. This script compiles all necessary components.

### KNIRVTESTNET Setup Checklist

To ensure a successful setup, please follow these steps:

* Clone the repository
* Navigate to the directory
* Build all components using `./scripts/build-all.sh`

## Starting the Testnet

Run `./scripts/start-testnet.sh` to start all services.  This script handles dependency checks and port conflicts.  The script will output logs indicating the status of each service.

### Starting the Testnet Checklist

To start the testnet, please run the following command:

* `./scripts/start-testnet.sh`

## Verifying Health

Use `./scripts/health-check.sh` to verify that all services are running correctly.  This script provides a summary of service health and response times.  For continuous monitoring, use `./scripts/health-check.sh --watch`.

### Verifying Health Checklist

To verify the health of your testnet, please run the following command:

* `./scripts/health-check.sh`

## Running Tests

The KNIRVTESTNET includes several test suites:

* **Integration Tests:** `./scripts/run-tests.sh` - Tests communication between services.
* **Economic Loop Testing:** `./scripts/test-economics.sh` - Tests the economic model (note: may use mocked values in testnet).
* **Agent Simulation:** `./scripts/simulate-agents.sh` - Simulates agent interactions.

### Running Tests Checklist

To run tests, please use the following commands:

* `./scripts/run-tests.sh` (integration tests)
* `./scripts/test-economics.sh` (economic loop testing)
* `./scripts/simulate-agents.sh` (agent simulation)

## Stopping the Testnet

When finished, run `./scripts/stop-testnet.sh` to gracefully shut down all services.

### Stopping the Testnet Checklist

To stop the testnet, please run the following command:

* `./scripts/stop-testnet.sh`

## Service Endpoints

The following endpoints are available once the testnet is running:

| Service          | Endpoint             | Purpose                                      |
|-----------------|----------------------|----------------------------------------------|
| KNIRV-ROOT       | http://localhost:1317 | NRN blockchain & oracle                      |
| KNIRVCHAIN       | http://localhost:8080 | Base LLM & skill registry                    |
| KNIRVGRAPH       | http://localhost:8081 | Knowledge graphchain                         |
| KNIRV-NEXUS-1    | http://localhost:8082 | Validation environment                       |
| KNIRV-NEXUS-2    | http://localhost:8083 | Validation environment                       |
| KNIRV-ROUTER     | http://localhost:8086 | Network routing                              |
| KNIRV-GATEWAY    | http://localhost:8087 | Unified API gateway                          |
| IPFS API         | http://localhost:5001 | Decentralized storage                         |
| Gateway Health   | http://localhost:8888/gateway/health | Checks the health of the API Gateway        |
| Service Discovery| http://localhost:8888/gateway/services | Lists all available services                |
| Testnet Status   | http://localhost:8888/gateway/testnet/status | Provides overall testnet status             |
| Auth Tokens      | http://localhost:8888/auth/testnet-tokens | Access testnet authentication tokens       |
| Mock LLM         | http://localhost:8080/testnet/llm/validate | Tests the mock Large Language Model (LLM)  |
| TEE Simulation   | http://localhost:8182/testnet/validate/skill | Tests the Trusted Execution Environment (TEE) simulation |

## Troubleshooting

### Common Issues

* **Port Conflicts:** Use `netstat -tulpn | grep :<port>` (replace `<port>` with the conflicting port number) to identify processes using the port.
* **Service Startup Failures:** Check the logs in the `./logs/` directory for error messages.  The specific log file will depend on the failing service.
* **Database Corruption:** Remove the relevant data directory (`rm -rf data/service-name/`) and restart the service.  This will reset the database.

### Getting Help

1. Check the logs in `./logs/`.
2. Run the health check: `./scripts/health-check.sh`.
3. Review the configuration files in the `./config/` directory.
4. Check the process status using `ps aux | grep knirv`.

## Docker Deployment (Alternative)

For a Docker-based deployment, use the provided `docker-compose.yml` file:

1. **Start:** `docker-compose up -d`
2. **Monitor Logs:** `docker-compose logs -f`
3. **Stop:** `docker-compose down`

**Note:** This is a simplified testnet implementation.  Some features are simulated, and economic operations may be mocked.  Real TEE is replaced with a simulation.  All tokens are valueless test tokens.  For production deployment, refer to the "Production Migration" section in the original README.

Improvements made:

* Added a prerequisites checklist to ensure users have the necessary tools installed.
* Created a setup checklist to guide users through the process of setting up KNIRVTESTNET.
* Improved the formatting and organization of the content for better readability.
* Added a troubleshooting section with common issues and solutions.
* Included a getting help section with steps to resolve issues.
* Updated the Docker deployment section to include a start, monitor, and stop checklist.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
