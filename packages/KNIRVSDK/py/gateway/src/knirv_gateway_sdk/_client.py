"""
Main client for the KNIRV Gateway SDK
"""

from __future__ import annotations

import os
from typing import Optional, Dict, Any, Union
import httpx

from ._base_client import BaseClient
from ._utils import NotGiven, NOT_GIVEN
from .resources.economics import EconomicsResource
from .resources.gateway import GatewayResource
from .resources.health import HealthResource
from .resources.integration import IntegrationResource
from .resources.poaud import PoAuDResource
from .types import ClientConfig


class KNIRVGateway(BaseClient):
    """
    Main client for interacting with KNIRV Gateway services.
    
    This client provides access to all KNIRV Gateway services:
    - Economics: Skills, LLM, Validation, Fees, Metrics, Transactions, Burn, Rules
    - Gateway: Routes, Health, Status
    - Health: Service health checks
    - Integration: External service integrations
    - PoAuD: Proof of Audit services
    
    Example:
        ```python
        from knirv_gateway_sdk import KNIRVGateway
        
        client = KNIRVGateway(
            base_url="https://gateway.knirv.com",
            api_key="your-api-key"
        )
        
        # Use economics service
        skills = await client.economics.skills.list()
        
        # Use health service
        status = await client.health.check()
        ```
    """
    
    economics: EconomicsResource
    gateway: GatewayResource
    health: HealthResource
    integration: IntegrationResource
    poaud: PoAuDResource
    
    def __init__(
        self,
        *,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        timeout: Union[float, httpx.Timeout, None] = None,
        max_retries: int = 2,
        default_headers: Optional[Dict[str, str]] = None,
        default_query: Optional[Dict[str, Any]] = None,
        debug: bool = False,
        **kwargs: Any,
    ) -> None:
        """
        Initialize the KNIRV Gateway client.
        
        Args:
            base_url: Base URL for the KNIRV Gateway API
            api_key: API key for authentication
            timeout: Request timeout in seconds
            max_retries: Maximum number of retries for failed requests
            default_headers: Default headers to include with all requests
            default_query: Default query parameters to include with all requests
            debug: Enable debug logging
            **kwargs: Additional arguments passed to the base client
        """
        if base_url is None:
            base_url = os.environ.get("KNIRV_GATEWAY_BASE_URL", "https://gateway.knirv.com")
        
        if api_key is None:
            api_key = os.environ.get("KNIRV_GATEWAY_API_KEY")
        
        if timeout is None:
            timeout = 60.0
        
        super().__init__(
            base_url=base_url,
            timeout=timeout,
            max_retries=max_retries,
            default_headers=default_headers,
            default_query=default_query,
            **kwargs,
        )
        
        self._api_key = api_key
        self._debug = debug
        
        # Initialize service resources
        self.economics = EconomicsResource(self)
        self.gateway = GatewayResource(self)
        self.health = HealthResource(self)
        self.integration = IntegrationResource(self)
        self.poaud = PoAuDResource(self)
    
    @property
    def auth_headers(self) -> Dict[str, str]:
        """Get authentication headers."""
        headers = {}
        if self._api_key:
            headers["Authorization"] = f"Bearer {self._api_key}"
        return headers
    
    @classmethod
    def create_economics_client(
        cls,
        *,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        **kwargs: Any,
    ) -> "KNIRVGateway":
        """
        Create a client specifically configured for the Economics Service.
        
        Args:
            base_url: Base URL for the Economics Service
            api_key: API key for authentication
            **kwargs: Additional arguments passed to the client
            
        Returns:
            KNIRVGateway client configured for Economics Service
        """
        if base_url is None:
            base_url = os.environ.get("ECONOMICS_SERVICE_URL", "http://localhost:8090")
        
        return cls(base_url=base_url, api_key=api_key, **kwargs)
    
    @classmethod
    def create_gateway_client(
        cls,
        *,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        **kwargs: Any,
    ) -> "KNIRVGateway":
        """
        Create a client specifically configured for the API Gateway.
        
        Args:
            base_url: Base URL for the API Gateway
            api_key: API key for authentication
            **kwargs: Additional arguments passed to the client
            
        Returns:
            KNIRVGateway client configured for API Gateway
        """
        if base_url is None:
            base_url = os.environ.get("GATEWAY_SERVICE_URL", "http://localhost:8000")
        
        return cls(base_url=base_url, api_key=api_key, **kwargs)
    
    def copy(
        self,
        *,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        timeout: Union[float, httpx.Timeout, None, NotGiven] = NOT_GIVEN,
        max_retries: Union[int, NotGiven] = NOT_GIVEN,
        default_headers: Union[Dict[str, str], NotGiven] = NOT_GIVEN,
        default_query: Union[Dict[str, Any], NotGiven] = NOT_GIVEN,
        **kwargs: Any,
    ) -> "KNIRVGateway":
        """
        Create a copy of this client with potentially different configuration.
        
        Returns:
            A new KNIRVGateway client instance
        """
        if base_url is None:
            base_url = self.base_url
        
        if api_key is None:
            api_key = self._api_key
        
        return self.__class__(
            base_url=base_url,
            api_key=api_key,
            timeout=self.timeout if isinstance(timeout, NotGiven) else timeout,
            max_retries=self.max_retries if isinstance(max_retries, NotGiven) else max_retries,
            default_headers=self._default_headers if isinstance(default_headers, NotGiven) else default_headers,
            default_query=self._default_query if isinstance(default_query, NotGiven) else default_query,
            **kwargs,
        )
