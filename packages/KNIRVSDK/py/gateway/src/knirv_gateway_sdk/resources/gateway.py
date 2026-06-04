"""
Gateway service resource for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from ._base import BaseResource
from ..types import Route, HealthStatus, ServiceStatus


class RoutesResource(BaseResource):
    """Routes management resource."""
    
    async def list(self) -> List[Route]:
        """
        List all routes.
        
        Returns:
            List of routes
        """
        response = await self._get("/gateway/routes")
        return self._parse_response_list(response, Route)
    
    async def get(self, path: str) -> Route:
        """
        Get a specific route.
        
        Args:
            path: Route path
            
        Returns:
            Route information
        """
        response = await self._get(f"/gateway/routes/{path}")
        return self._parse_response(response, Route)
    
    async def create(
        self,
        *,
        path: str,
        method: str,
        handler: str,
        middleware: Optional[List[str]] = None,
        enabled: bool = True,
    ) -> Route:
        """
        Create a new route.
        
        Args:
            path: Route path
            method: HTTP method
            handler: Handler function
            middleware: Middleware list
            enabled: Whether the route is enabled
            
        Returns:
            Created route
        """
        data = {
            "path": path,
            "method": method,
            "handler": handler,
            "enabled": enabled,
        }
        if middleware:
            data["middleware"] = middleware
        
        response = await self._post("/gateway/routes", json_data=data)
        return self._parse_response(response, Route)
    
    async def update(
        self,
        path: str,
        *,
        method: Optional[str] = None,
        handler: Optional[str] = None,
        middleware: Optional[List[str]] = None,
        enabled: Optional[bool] = None,
    ) -> Route:
        """
        Update a route.
        
        Args:
            path: Route path
            method: HTTP method
            handler: Handler function
            middleware: Middleware list
            enabled: Whether the route is enabled
            
        Returns:
            Updated route
        """
        data = {}
        if method is not None:
            data["method"] = method
        if handler is not None:
            data["handler"] = handler
        if middleware is not None:
            data["middleware"] = middleware
        if enabled is not None:
            data["enabled"] = enabled
        
        response = await self._patch(f"/gateway/routes/{path}", json_data=data)
        return self._parse_response(response, Route)
    
    async def delete(self, path: str) -> Dict[str, Any]:
        """
        Delete a route.
        
        Args:
            path: Route path
            
        Returns:
            Deletion result
        """
        response = await self._delete(f"/gateway/routes/{path}")
        return response.json()


class GatewayHealthResource(BaseResource):
    """Gateway health resource."""
    
    async def check(self) -> HealthStatus:
        """
        Check gateway health.
        
        Returns:
            Health status
        """
        response = await self._get("/gateway/health")
        return self._parse_response(response, HealthStatus)


class StatusResource(BaseResource):
    """Status resource."""
    
    async def get(self) -> ServiceStatus:
        """
        Get service status.
        
        Returns:
            Service status
        """
        response = await self._get("/gateway/status")
        return self._parse_response(response, ServiceStatus)
    
    async def get_dependencies(self) -> List[ServiceStatus]:
        """
        Get status of all dependencies.
        
        Returns:
            List of dependency statuses
        """
        response = await self._get("/gateway/status/dependencies")
        return self._parse_response_list(response, ServiceStatus)


class GatewayResource(BaseResource):
    """Gateway service resource."""
    
    def __init__(self, client) -> None:
        super().__init__(client)
        self.routes = RoutesResource(client)
        self.health = GatewayHealthResource(client)
        self.status = StatusResource(client)
