"""
KNIRV Unified SDK Client

This module provides the main KNIRVClient class that integrates all KNIRV services.
"""

from __future__ import annotations
from typing import Optional, Any
import asyncio

from .config import ClientConfig
from .exceptions import KNIRVSDKError
from .services.gateway import GatewayService
from .services.transaction import TransactionService
from .services.transmission import TransmissionService


class KNIRVClient:
    """
    Unified client for all KNIRV Network services.
    
    This client provides access to:
    - Gateway: Economics, Health, Integration, PoAuD services
    - Transaction: Blockchain transaction management
    - Transmission: URI and data transmission services
    
    Example:
        ```python
        from knirv_sdk import KNIRVClient, ClientConfig
        
        # Create client with default configuration
        client = KNIRVClient()
        
        # Or with custom configuration
        config = ClientConfig(api_key="your-api-key")
        client = KNIRVClient(config)
        
        # Use gateway services
        skills = await client.gateway.economics.skills.list()
        
        # Use transaction services
        tx = await client.transaction.create_transaction(...)
        
        # Use transmission services
        uri = client.transmission.parse_uri("knirv://...")
        ```
    """
    
    def __init__(self, config: Optional[ClientConfig] = None) -> None:
        """
        Initialize the KNIRV client.
        
        Args:
            config: Client configuration. If None, uses default configuration
                   with environment variables.
        """
        self._config = config or ClientConfig()
        
        # Initialize services
        self._gateway: Optional[GatewayService] = None
        self._transaction: Optional[TransactionService] = None
        self._transmission: Optional[TransmissionService] = None
        
        # Track if client is closed
        self._closed = False
    
    @property
    def gateway(self) -> GatewayService:
        """Access to Gateway services (Economics, Health, Integration, PoAuD)."""
        if self._closed:
            raise KNIRVSDKError("Client has been closed")
        
        if self._gateway is None:
            self._gateway = GatewayService(self._config.gateway)
        
        return self._gateway
    
    @property
    def transaction(self) -> TransactionService:
        """Access to Transaction services."""
        if self._closed:
            raise KNIRVSDKError("Client has been closed")
        
        if self._transaction is None:
            self._transaction = TransactionService(self._config.transaction)
        
        return self._transaction
    
    @property
    def transmission(self) -> TransmissionService:
        """Access to Transmission services."""
        if self._closed:
            raise KNIRVSDKError("Client has been closed")
        
        if self._transmission is None:
            self._transmission = TransmissionService(self._config.transmission)
        
        return self._transmission
    
    @property
    def config(self) -> ClientConfig:
        """Get the client configuration."""
        return self._config
    
    @property
    def is_closed(self) -> bool:
        """Check if the client has been closed."""
        return self._closed
    
    async def close(self) -> None:
        """
        Close the client and clean up resources.
        
        This will close all underlying HTTP connections and mark the client
        as closed. After calling this method, the client cannot be used.
        """
        if self._closed:
            return
        
        # Close all services
        close_tasks = []
        
        if self._gateway is not None:
            close_tasks.append(self._gateway.close())
        
        if self._transaction is not None:
            close_tasks.append(self._transaction.close())
        
        if self._transmission is not None:
            close_tasks.append(self._transmission.close())
        
        if close_tasks:
            await asyncio.gather(*close_tasks, return_exceptions=True)
        
        self._closed = True
    
    async def __aenter__(self) -> KNIRVClient:
        """Async context manager entry."""
        return self
    
    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Async context manager exit."""
        await self.close()
    
    def __enter__(self) -> KNIRVClient:
        """Context manager entry (for sync usage)."""
        return self
    
    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Context manager exit (for sync usage)."""
        # For sync usage, we can't await close(), so we just mark as closed
        self._closed = True
    
    def __repr__(self) -> str:
        """String representation of the client."""
        status = "closed" if self._closed else "open"
        return f"KNIRVClient(environment={self._config.environment}, status={status})"


# Convenience functions for quick access
def create_client(
    api_key: Optional[str] = None,
    environment: str = "production",
    **kwargs: Any
) -> KNIRVClient:
    """
    Create a KNIRV client with simple configuration.
    
    Args:
        api_key: API key for authentication
        environment: Environment (production, staging, development)
        **kwargs: Additional configuration options
    
    Returns:
        Configured KNIRVClient instance
    """
    config = ClientConfig(
        api_key=api_key,
        environment=environment,
        **kwargs
    )
    return KNIRVClient(config)


def create_development_client(**kwargs: Any) -> KNIRVClient:
    """
    Create a KNIRV client configured for development.
    
    Args:
        **kwargs: Additional configuration options
    
    Returns:
        KNIRVClient configured for development
    """
    config = ClientConfig.for_development(**kwargs)
    return KNIRVClient(config)


def create_testing_client(**kwargs: Any) -> KNIRVClient:
    """
    Create a KNIRV client configured for testing.
    
    Args:
        **kwargs: Additional configuration options
    
    Returns:
        KNIRVClient configured for testing
    """
    config = ClientConfig.for_testing(**kwargs)
    return KNIRVClient(config)
