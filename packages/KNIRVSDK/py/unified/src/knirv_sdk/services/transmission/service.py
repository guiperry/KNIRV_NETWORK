"""
KNIRV Transmission Service

This module provides URI and data transmission services.
"""

from __future__ import annotations
from typing import Any, Dict, List, Optional
from urllib.parse import urlparse, parse_qs
import httpx

from ...config import TransmissionConfig
from ...exceptions import KNIRVTransmissionError, KNIRVValidationError
from ..base import BaseService


class KnirvURI:
    """Represents a parsed knirv:// URI."""
    
    def __init__(self, id: str, resource_type: str, path: str, query: Dict[str, List[str]], raw: str):
        """
        Initialize a KnirvURI object.
        
        Args:
            id: The ID part of the URI.
            resource_type: The resource type part of the URI.
            path: The path part of the URI.
            query: The query parameters as a dictionary of lists.
            raw: The original URI string.
        """
        self.id = id
        self.resource_type = resource_type
        self.path = path
        self.query = query
        self.raw = raw
    
    def __str__(self) -> str:
        """Return the string representation of the URI."""
        return self.raw
    
    def get_query_param(self, key: str) -> Optional[str]:
        """
        Get the first value for a query parameter.
        
        Args:
            key: The query parameter key.
            
        Returns:
            The first value for the key, or None if not found.
        """
        values = self.query.get(key, [])
        return values[0] if values else None
    
    def get_query_params(self, key: str) -> List[str]:
        """
        Get all values for a query parameter.
        
        Args:
            key: The query parameter key.
            
        Returns:
            List of values for the key.
        """
        return self.query.get(key, [])


class TransmissionService(BaseService):
    """
    Service for URI and data transmission.
    
    This service provides:
    - KNIRV URI parsing and generation
    - Data transmission capabilities
    - URI validation and resolution
    
    Example:
        ```python
        from knirv_sdk.services.transmission import TransmissionService
        from knirv_sdk.config import TransmissionConfig
        
        config = TransmissionConfig(
            base_url="https://transmission.knirv.com",
            api_key="your-api-key"
        )
        service = TransmissionService(config)
        
        # Parse a KNIRV URI
        uri = service.parse_uri("knirv://skill123/execute?param=value")
        
        # Generate a new URI
        new_uri = await service.generate_uri({
            "resource_type": "skill",
            "id": "skill123",
            "action": "execute"
        })
        ```
    """
    
    def __init__(self, config: Optional[TransmissionConfig] = None) -> None:
        """
        Initialize the Transmission service.
        
        Args:
            config: Transmission configuration. If None, uses default configuration.
        """
        self._config = config or TransmissionConfig()
        super().__init__(self._config.base_url, self._config)
    
    def parse_uri(self, uri: str) -> KnirvURI:
        """
        Parse a knirv:// URI.
        
        Args:
            uri: The URI string to parse
            
        Returns:
            Parsed KnirvURI object
            
        Raises:
            KNIRVValidationError: If the URI is invalid
        """
        if not uri:
            raise KNIRVValidationError("URI is required")
        
        if not uri.startswith("knirv://"):
            raise KNIRVValidationError("URI must start with 'knirv://'")
        
        try:
            parsed = urlparse(uri)
            
            # Extract components
            netloc = parsed.netloc  # This contains the ID and resource type
            path = parsed.path
            query = parse_qs(parsed.query)
            
            # Parse netloc to extract ID and resource type
            # Format: knirv://id/resource_type/path?query
            if not netloc:
                raise KNIRVValidationError("URI must contain an ID")
            
            # Split path to get resource type
            path_parts = path.strip('/').split('/') if path else []
            resource_type = path_parts[0] if path_parts else ""
            remaining_path = '/'.join(path_parts[1:]) if len(path_parts) > 1 else ""
            
            return KnirvURI(
                id=netloc,
                resource_type=resource_type,
                path=remaining_path,
                query=query,
                raw=uri
            )
            
        except Exception as e:
            raise KNIRVValidationError(f"Failed to parse URI: {e}") from e
    
    async def generate_uri(self, uri_data: Dict[str, Any]) -> str:
        """
        Generate a new KNIRV URI.
        
        Args:
            uri_data: URI generation data
            
        Returns:
            Generated URI string
        """
        if not uri_data.get("resource_type"):
            raise KNIRVValidationError("Resource type is required")
        if not uri_data.get("id"):
            raise KNIRVValidationError("ID is required")
        
        try:
            response = await self._request("POST", "/uri/generate", json=uri_data)
            result = response.json()
            return result.get("uri", "")
        except httpx.HTTPError as e:
            raise KNIRVTransmissionError(f"Failed to generate URI: {e}") from e
    
    async def resolve_uri(self, uri: str) -> Dict[str, Any]:
        """
        Resolve a KNIRV URI to get resource information.
        
        Args:
            uri: URI to resolve
            
        Returns:
            Resource information
        """
        if not uri:
            raise KNIRVValidationError("URI is required")
        
        try:
            response = await self._request("POST", "/uri/resolve", json={"uri": uri})
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransmissionError(f"Failed to resolve URI: {e}") from e
    
    async def validate_uri(self, uri: str) -> Dict[str, Any]:
        """
        Validate a KNIRV URI.
        
        Args:
            uri: URI to validate
            
        Returns:
            Validation result
        """
        if not uri:
            raise KNIRVValidationError("URI is required")
        
        try:
            response = await self._request("POST", "/uri/validate", json={"uri": uri})
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransmissionError(f"Failed to validate URI: {e}") from e
    
    async def transmit_data(self, data: Dict[str, Any], destination: str) -> Dict[str, Any]:
        """
        Transmit data to a destination.
        
        Args:
            data: Data to transmit
            destination: Destination URI or address
            
        Returns:
            Transmission result
        """
        if not data:
            raise KNIRVValidationError("Data is required")
        if not destination:
            raise KNIRVValidationError("Destination is required")
        
        transmission_data = {
            "data": data,
            "destination": destination
        }
        
        try:
            response = await self._request("POST", "/transmit", json=transmission_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransmissionError(f"Failed to transmit data: {e}") from e
    
    async def get_transmission_status(self, transmission_id: str) -> Dict[str, Any]:
        """
        Get the status of a data transmission.
        
        Args:
            transmission_id: Transmission identifier
            
        Returns:
            Transmission status
        """
        if not transmission_id:
            raise KNIRVValidationError("Transmission ID is required")
        
        try:
            response = await self._request("GET", f"/transmit/{transmission_id}/status")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransmissionError(f"Failed to get transmission status: {e}") from e
