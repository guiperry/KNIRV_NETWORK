"""
URI parser module for the KNIRV Client SDK.

This module provides functions for parsing knirv:// URIs.
"""

from urllib.parse import urlparse, parse_qs
from typing import Dict, List, Optional, Any, Union


class KnirvURIError(Exception):
    """Exception raised for invalid KNIRV URIs."""
    
    def __init__(self, message: str, uri: str):
        self.message = message
        self.uri = uri
        super().__init__(f"Invalid knirv URI ({uri}): {message}")


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
            The first value for the query parameter, or None if it doesn't exist.
        """
        values = self.query.get(key)
        return values[0] if values else None
    
    def has_query_param(self, key: str) -> bool:
        """
        Check if a query parameter exists.
        
        Args:
            key: The query parameter key.
            
        Returns:
            True if the query parameter exists, False otherwise.
        """
        return key in self.query
    
    def get_query_params(self, key: str) -> List[str]:
        """
        Get all values for a query parameter.
        
        Args:
            key: The query parameter key.
            
        Returns:
            A list of values for the query parameter, or an empty list if it doesn't exist.
        """
        return self.query.get(key, [])
    
    def get_all_query_params(self) -> Dict[str, List[str]]:
        """
        Get all query parameters.
        
        Returns:
            A dictionary of all query parameters.
        """
        return self.query


def parse_knirv_uri(uri_string: str) -> KnirvURI:
    """
    Parse a knirv:// URI string into a KnirvURI object.
    
    Args:
        uri_string: The URI string to parse.
        
    Returns:
        A KnirvURI object representing the parsed URI.
        
    Raises:
        KnirvURIError: If the URI is invalid.
    """
    try:
        parsed_url = urlparse(uri_string)
    except Exception as e:
        raise KnirvURIError(f"Failed to parse URI: {e}", uri_string)
    
    # Check scheme
    if parsed_url.scheme != "knirv":
        raise KnirvURIError(f"Invalid scheme: expected 'knirv', got '{parsed_url.scheme}'", uri_string)
    
    # Extract ID and ResourceType from authority
    host = parsed_url.netloc
    host_parts = host.split(".")
    if len(host_parts) != 2:
        raise KnirvURIError("Invalid authority format: expected <ID>.<ResourceType>", uri_string)
    
    id_part = host_parts[0]
    resource_type = host_parts[1]
    
    # Validate ID and ResourceType are non-empty
    if not id_part or not resource_type:
        raise KnirvURIError("ID and ResourceType cannot be empty", uri_string)
    
    # Normalize path
    path = parsed_url.path
    if not path:
        path = "/"  # Default path
    
    # Parse query parameters
    query_params = parse_qs(parsed_url.query)
    
    return KnirvURI(
        id=id_part,
        resource_type=resource_type,
        path=path,
        query=query_params,
        raw=uri_string
    )