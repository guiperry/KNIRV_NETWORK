"""
KNIRV Gateway Service Implementation

This module provides the main GatewayService class that integrates all gateway services.
"""

from __future__ import annotations
from typing import Optional
import httpx

from ...config import GatewayConfig
from ...exceptions import KNIRVGatewayError
from ..base import BaseService
from .economics import EconomicsService
from .health import HealthService
from .integration import IntegrationService
from .poaud import PoAuDService


class GatewayService(BaseService):
    """
    Main service for interacting with KNIRV Gateway services.
    
    This service provides access to all KNIRV Gateway services:
    - Economics: Skills, LLM, Validation, Fees, Metrics, Transactions, Burn, Rules
    - Health: Service health checks
    - Integration: External service integrations
    - PoAuD: Proof of Audit services
    
    Example:
        ```python
        from knirv_sdk.services.gateway import GatewayService
        from knirv_sdk.config import GatewayConfig
        
        config = GatewayConfig(
            base_url="https://gateway.knirv.com",
            api_key="your-api-key"
        )
        service = GatewayService(config)
        
        # Use economics service
        skills = await service.economics.skills.list()
        
        # Use health service
        status = await service.health.check()
        ```
    """
    
    def __init__(self, config: Optional[GatewayConfig] = None) -> None:
        """
        Initialize the Gateway service.
        
        Args:
            config: Gateway configuration. If None, uses default configuration.
        """
        self._config = config or GatewayConfig()
        super().__init__(self._config.base_url, self._config)
        
        # Initialize sub-services
        self._economics: Optional[EconomicsService] = None
        self._health: Optional[HealthService] = None
        self._integration: Optional[IntegrationService] = None
        self._poaud: Optional[PoAuDService] = None
    
    @property
    def economics(self) -> EconomicsService:
        """Access to Economics services (Skills, LLM, Validation)."""
        if self._economics is None:
            self._economics = EconomicsService(self._http_client, self._config)
        return self._economics
    
    @property
    def health(self) -> HealthService:
        """Access to Health monitoring services."""
        if self._health is None:
            self._health = HealthService(self._http_client, self._config)
        return self._health
    
    @property
    def integration(self) -> IntegrationService:
        """Access to Integration services."""
        if self._integration is None:
            self._integration = IntegrationService(self._http_client, self._config)
        return self._integration
    
    @property
    def poaud(self) -> PoAuDService:
        """Access to PoAuD (Proof of Audit) services."""
        if self._poaud is None:
            self._poaud = PoAuDService(self._http_client, self._config)
        return self._poaud
    
    async def get_status(self) -> dict:
        """
        Get the overall gateway status.
        
        Returns:
            Gateway status information
        """
        try:
            response = await self._http_client.get("/status")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get gateway status: {e}") from e
    
    async def get_version(self) -> dict:
        """
        Get the gateway version information.
        
        Returns:
            Gateway version information
        """
        try:
            response = await self._http_client.get("/version")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get gateway version: {e}") from e
    
    async def ping(self) -> dict:
        """
        Ping the gateway to check connectivity.
        
        Returns:
            Ping response
        """
        try:
            response = await self._http_client.get("/ping")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to ping gateway: {e}") from e
