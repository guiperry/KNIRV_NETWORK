"""
KNIRV Health Service

This module provides health monitoring and status checking services.
"""

from __future__ import annotations
from typing import Any, Dict
import httpx

from ...exceptions import KNIRVGatewayError


class HealthService:
    """Health monitoring service."""
    
    def __init__(self, http_client: httpx.AsyncClient, config: Any) -> None:
        self._client = http_client
        self._config = config
    
    async def check(self) -> Dict[str, Any]:
        """
        Perform a health check.
        
        Returns:
            Health status information
        """
        try:
            response = await self._client.get("/health")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Health check failed: {e}") from e
    
    async def get_status(self) -> Dict[str, Any]:
        """
        Get detailed service status.
        
        Returns:
            Detailed status information
        """
        try:
            response = await self._client.get("/health/status")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get status: {e}") from e
    
    async def get_metrics(self) -> Dict[str, Any]:
        """
        Get health metrics.
        
        Returns:
            Health metrics data
        """
        try:
            response = await self._client.get("/health/metrics")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get health metrics: {e}") from e
