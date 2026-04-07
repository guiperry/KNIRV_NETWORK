"""
KNIRV Unified SDK for Python

This package provides a unified interface to all KNIRV Network services:
- Gateway: Economics, Health, Integration, PoAuD services
- Transaction: Blockchain transaction management
- Transmission: URI and data transmission services

Example:
    ```python
    from knirv_sdk import KNIRVClient, ClientConfig
    
    # Create client with default configuration
    client = KNIRVClient()
    
    # Or with custom configuration
    config = ClientConfig(
        gateway_url="https://gateway.knirv.com",
        transaction_url="https://transaction.knirv.com",
        transmission_url="https://transmission.knirv.com",
        api_key="your-api-key"
    )
    client = KNIRVClient(config)
    
    # Use gateway services
    skills = await client.gateway.economics.skills.list()
    
    # Use transaction services
    tx = await client.transaction.create_transaction(...)
    
    # Use transmission services
    uri = client.transmission.parse_uri("knirv://...")
    ```
"""

from __future__ import annotations

from .client import KNIRVClient, create_client, create_development_client, create_testing_client
from .config import ClientConfig, GatewayConfig, TransactionConfig, TransmissionConfig
from .exceptions import (
    KNIRVSDKError,
    KNIRVAPIError,
    KNIRVValidationError,
    KNIRVConnectionError,
    KNIRVTimeoutError,
    KNIRVAuthenticationError,
    KNIRVNotFoundError,
)
from ._version import __version__

# Re-export service-specific classes
from .services.gateway import (
    GatewayService,
    EconomicsService,
    SkillsService,
    LLMService,
    ValidationService,
    HealthService,
    IntegrationService,
    PoAuDService,
)
from .services.transaction import TransactionService
from .services.transmission import TransmissionService

__all__ = [
    # Main client and configuration
    "KNIRVClient",
    "create_client",
    "create_development_client",
    "create_testing_client",
    "ClientConfig",
    "GatewayConfig",
    "TransactionConfig",
    "TransmissionConfig",

    # Services
    "GatewayService",
    "EconomicsService",
    "SkillsService",
    "LLMService",
    "ValidationService",
    "HealthService",
    "IntegrationService",
    "PoAuDService",
    "TransactionService",
    "TransmissionService",

    # Exceptions
    "KNIRVSDKError",
    "KNIRVAPIError",
    "KNIRVValidationError",
    "KNIRVConnectionError",
    "KNIRVTimeoutError",
    "KNIRVAuthenticationError",
    "KNIRVNotFoundError",

    # Version
    "__version__",
]
