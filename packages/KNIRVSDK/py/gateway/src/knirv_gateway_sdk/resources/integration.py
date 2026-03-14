"""
Integration service resource for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from ._base import BaseResource
from ..types import Integration


class IntegrationResource(BaseResource):
    """Integration service resource."""
    
    async def list(
        self,
        *,
        page: int = 1,
        per_page: int = 20,
        integration_type: Optional[str] = None,
        enabled: Optional[bool] = None,
    ) -> List[Integration]:
        """
        List integrations.
        
        Args:
            page: Page number
            per_page: Items per page
            integration_type: Filter by integration type
            enabled: Filter by enabled status
            
        Returns:
            List of integrations
        """
        params = {"page": page, "per_page": per_page}
        if integration_type:
            params["type"] = integration_type
        if enabled is not None:
            params["enabled"] = enabled
        
        response = await self._get("/integration", params=params)
        return self._parse_response_list(response, Integration)
    
    async def get(self, integration_id: str) -> Integration:
        """
        Get a specific integration.
        
        Args:
            integration_id: Integration ID
            
        Returns:
            Integration
        """
        response = await self._get(f"/integration/{integration_id}")
        return self._parse_response(response, Integration)
    
    async def create(
        self,
        *,
        name: str,
        integration_type: str,
        config: Dict[str, Any],
        enabled: bool = True,
    ) -> Integration:
        """
        Create a new integration.
        
        Args:
            name: Integration name
            integration_type: Integration type
            config: Integration configuration
            enabled: Whether the integration is enabled
            
        Returns:
            Created integration
        """
        data = {
            "name": name,
            "type": integration_type,
            "config": config,
            "enabled": enabled,
        }
        response = await self._post("/integration", json_data=data)
        return self._parse_response(response, Integration)
    
    async def update(
        self,
        integration_id: str,
        *,
        name: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None,
        enabled: Optional[bool] = None,
    ) -> Integration:
        """
        Update an integration.
        
        Args:
            integration_id: Integration ID
            name: Integration name
            config: Integration configuration
            enabled: Whether the integration is enabled
            
        Returns:
            Updated integration
        """
        data = {}
        if name is not None:
            data["name"] = name
        if config is not None:
            data["config"] = config
        if enabled is not None:
            data["enabled"] = enabled
        
        response = await self._patch(f"/integration/{integration_id}", json_data=data)
        return self._parse_response(response, Integration)
    
    async def delete(self, integration_id: str) -> Dict[str, Any]:
        """
        Delete an integration.
        
        Args:
            integration_id: Integration ID
            
        Returns:
            Deletion result
        """
        response = await self._delete(f"/integration/{integration_id}")
        return response.json()
    
    async def test(self, integration_id: str) -> Dict[str, Any]:
        """
        Test an integration.
        
        Args:
            integration_id: Integration ID
            
        Returns:
            Test result
        """
        response = await self._post(f"/integration/{integration_id}/test")
        return response.json()
    
    async def enable(self, integration_id: str) -> Integration:
        """
        Enable an integration.
        
        Args:
            integration_id: Integration ID
            
        Returns:
            Updated integration
        """
        response = await self._post(f"/integration/{integration_id}/enable")
        return self._parse_response(response, Integration)
    
    async def disable(self, integration_id: str) -> Integration:
        """
        Disable an integration.
        
        Args:
            integration_id: Integration ID
            
        Returns:
            Updated integration
        """
        response = await self._post(f"/integration/{integration_id}/disable")
        return self._parse_response(response, Integration)
