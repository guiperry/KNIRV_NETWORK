"""
KNIRV Integration Service

This module provides external service integration capabilities.
"""

from __future__ import annotations
from typing import Any, Dict, List, Optional
import httpx

from ...exceptions import KNIRVGatewayError, KNIRVValidationError


class IntegrationService:
    """External service integration service."""
    
    def __init__(self, http_client: httpx.AsyncClient, config: Any) -> None:
        self._client = http_client
        self._config = config
    
    async def list_integrations(self) -> List[Dict[str, Any]]:
        """
        List available integrations.
        
        Returns:
            List of available integrations
        """
        try:
            response = await self._client.get("/integration")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to list integrations: {e}") from e
    
    async def get_integration(self, integration_id: str) -> Dict[str, Any]:
        """
        Get details of a specific integration.
        
        Args:
            integration_id: Integration identifier
            
        Returns:
            Integration details
        """
        if not integration_id:
            raise KNIRVValidationError("Integration ID is required")
        
        try:
            response = await self._client.get(f"/integration/{integration_id}")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get integration {integration_id}: {e}") from e
    
    async def create_integration(self, integration_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new integration.
        
        Args:
            integration_data: Integration configuration data
            
        Returns:
            Created integration
        """
        if not integration_data.get("name"):
            raise KNIRVValidationError("Integration name is required")
        if not integration_data.get("type"):
            raise KNIRVValidationError("Integration type is required")
        
        try:
            response = await self._client.post("/integration", json=integration_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to create integration: {e}") from e
    
    async def update_integration(
        self, 
        integration_id: str, 
        integration_data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Update an existing integration.
        
        Args:
            integration_id: Integration identifier
            integration_data: Updated integration data
            
        Returns:
            Updated integration
        """
        if not integration_id:
            raise KNIRVValidationError("Integration ID is required")
        
        try:
            response = await self._client.put(f"/integration/{integration_id}", json=integration_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to update integration {integration_id}: {e}") from e
    
    async def delete_integration(self, integration_id: str) -> None:
        """
        Delete an integration.
        
        Args:
            integration_id: Integration identifier
        """
        if not integration_id:
            raise KNIRVValidationError("Integration ID is required")
        
        try:
            await self._client.delete(f"/integration/{integration_id}")
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to delete integration {integration_id}: {e}") from e
    
    async def test_integration(self, integration_id: str) -> Dict[str, Any]:
        """
        Test an integration connection.
        
        Args:
            integration_id: Integration identifier
            
        Returns:
            Test result
        """
        if not integration_id:
            raise KNIRVValidationError("Integration ID is required")
        
        try:
            response = await self._client.post(f"/integration/{integration_id}/test")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to test integration {integration_id}: {e}") from e
