"""
Health service resource for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from ._base import BaseResource
from ..types import HealthStatus


class HealthResource(BaseResource):
    """Health service resource."""
    
    async def check(self, service: Optional[str] = None) -> HealthStatus:
        """
        Check health status.
        
        Args:
            service: Specific service to check (optional)
            
        Returns:
            Health status
        """
        if service:
            response = await self._get(f"/health/{service}")
        else:
            response = await self._get("/health")
        
        return self._parse_response(response, HealthStatus)
    
    async def check_all(self) -> List[HealthStatus]:
        """
        Check health status of all services.
        
        Returns:
            List of health statuses
        """
        response = await self._get("/health/all")
        return self._parse_response_list(response, HealthStatus)
    
    async def ping(self) -> Dict[str, Any]:
        """
        Simple ping endpoint.
        
        Returns:
            Ping response
        """
        response = await self._get("/health/ping")
        return response.json()
    
    async def ready(self) -> Dict[str, Any]:
        """
        Check if the service is ready.
        
        Returns:
            Readiness status
        """
        response = await self._get("/health/ready")
        return response.json()
    
    async def live(self) -> Dict[str, Any]:
        """
        Check if the service is live.
        
        Returns:
            Liveness status
        """
        response = await self._get("/health/live")
        return response.json()
