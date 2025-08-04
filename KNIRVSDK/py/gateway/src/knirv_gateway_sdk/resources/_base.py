"""
Base resource class for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any, Dict, List, Optional, Type, TypeVar
import httpx

if TYPE_CHECKING:
    from .._client import KNIRVGateway

T = TypeVar("T")


class BaseResource:
    """Base class for all API resources."""
    
    def __init__(self, client: "KNIRVGateway") -> None:
        """
        Initialize the resource.
        
        Args:
            client: The KNIRV Gateway client instance
        """
        self._client = client
    
    async def _get(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        **kwargs: Any,
    ) -> httpx.Response:
        """Make a GET request."""
        return await self._client.get(path, params=params, headers=headers, **kwargs)
    
    async def _post(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        json_data: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> httpx.Response:
        """Make a POST request."""
        return await self._client.post(
            path, params=params, headers=headers, json_data=json_data, **kwargs
        )
    
    async def _put(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        json_data: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> httpx.Response:
        """Make a PUT request."""
        return await self._client.put(
            path, params=params, headers=headers, json_data=json_data, **kwargs
        )
    
    async def _patch(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        json_data: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> httpx.Response:
        """Make a PATCH request."""
        return await self._client.patch(
            path, params=params, headers=headers, json_data=json_data, **kwargs
        )
    
    async def _delete(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        **kwargs: Any,
    ) -> httpx.Response:
        """Make a DELETE request."""
        return await self._client.delete(path, params=params, headers=headers, **kwargs)
    
    def _parse_response(
        self,
        response: httpx.Response,
        model_class: Type[T],
    ) -> T:
        """
        Parse a response into a model instance.
        
        Args:
            response: HTTP response
            model_class: Pydantic model class to parse into
            
        Returns:
            Parsed model instance
        """
        data = response.json()
        return model_class.model_validate(data)
    
    def _parse_response_list(
        self,
        response: httpx.Response,
        model_class: Type[T],
    ) -> List[T]:
        """
        Parse a response into a list of model instances.
        
        Args:
            response: HTTP response
            model_class: Pydantic model class to parse into
            
        Returns:
            List of parsed model instances
        """
        data = response.json()
        if isinstance(data, dict) and "data" in data:
            # Handle paginated responses
            items = data["data"]
        else:
            items = data
        
        return [model_class.model_validate(item) for item in items]
