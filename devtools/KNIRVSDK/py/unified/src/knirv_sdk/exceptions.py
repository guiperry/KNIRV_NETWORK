"""
KNIRV SDK Exception Classes

This module defines custom exception classes for the unified KNIRV SDK.
"""

from __future__ import annotations
from typing import Any, Dict, Optional


class KNIRVSDKError(Exception):
    """Base exception for all KNIRV SDK errors."""
    
    def __init__(self, message: str, **kwargs: Any) -> None:
        super().__init__(message)
        self.message = message
        for key, value in kwargs.items():
            setattr(self, key, value)


class KNIRVAPIError(KNIRVSDKError):
    """Base exception for KNIRV API errors."""
    
    def __init__(
        self, 
        message: str, 
        status_code: Optional[int] = None, 
        response: Optional[Dict[str, Any]] = None,
        **kwargs: Any
    ) -> None:
        super().__init__(message, **kwargs)
        self.status_code = status_code
        self.response = response


class KNIRVValidationError(KNIRVSDKError):
    """Exception raised for validation errors."""
    
    def __init__(self, message: str, field: Optional[str] = None, **kwargs: Any) -> None:
        super().__init__(message, **kwargs)
        self.field = field


class KNIRVConnectionError(KNIRVAPIError):
    """Exception raised for connection errors."""
    pass


class KNIRVTimeoutError(KNIRVAPIError):
    """Exception raised for timeout errors."""
    pass


class KNIRVAuthenticationError(KNIRVAPIError):
    """Exception raised for authentication errors."""
    pass


class KNIRVNotFoundError(KNIRVAPIError):
    """Exception raised when a resource is not found."""
    pass


class KNIRVRateLimitError(KNIRVAPIError):
    """Exception raised when rate limit is exceeded."""
    pass


class KNIRVServerError(KNIRVAPIError):
    """Exception raised for server errors (5xx status codes)."""
    pass


class KNIRVBadRequestError(KNIRVAPIError):
    """Exception raised for bad request errors (400 status code)."""
    pass


class KNIRVUnauthorizedError(KNIRVAPIError):
    """Exception raised for unauthorized errors (401 status code)."""
    pass


class KNIRVForbiddenError(KNIRVAPIError):
    """Exception raised for forbidden errors (403 status code)."""
    pass


class KNIRVConflictError(KNIRVAPIError):
    """Exception raised for conflict errors (409 status code)."""
    pass


class KNIRVUnprocessableEntityError(KNIRVAPIError):
    """Exception raised for unprocessable entity errors (422 status code)."""
    pass


# Service-specific exceptions
class KNIRVGatewayError(KNIRVAPIError):
    """Exception specific to Gateway service errors."""
    pass


class KNIRVTransactionError(KNIRVAPIError):
    """Exception specific to Transaction service errors."""
    pass


class KNIRVTransmissionError(KNIRVAPIError):
    """Exception specific to Transmission service errors."""
    pass


# Economics service specific exceptions
class KNIRVSkillError(KNIRVGatewayError):
    """Exception specific to Skills service errors."""
    pass


class KNIRVLLMError(KNIRVGatewayError):
    """Exception specific to LLM service errors."""
    pass


class KNIRVValidationServiceError(KNIRVGatewayError):
    """Exception specific to Validation service errors."""
    pass
