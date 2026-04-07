"""
KNIRV Gateway SDK for Python

This package provides access to KNIRV Gateway services including:
- Economics API (Skills, LLM, Validation, Fees, Metrics, Transactions, Burn, Rules)
- Gateway API (Routes, Health, Status)
- Health Service
- Integration Service
- PoAuD (Proof of Audit) Service
"""

from __future__ import annotations

from ._client import KNIRVGateway
from ._version import __version__
from .types import *
from ._exceptions import (
    KNIRVGatewayError,
    APIError,
    APIConnectionError,
    APIResponseValidationError,
    APIStatusError,
    APITimeoutError,
    BadRequestError,
    AuthenticationError,
    PermissionDeniedError,
    NotFoundError,
    ConflictError,
    UnprocessableEntityError,
    RateLimitError,
    InternalServerError,
    KNIRVAPIError,
    KNIRVValidationError,
)

__all__ = [
    "__version__",
    "KNIRVGateway",
    "KNIRVGatewayError",
    "APIError",
    "APIConnectionError",
    "APIResponseValidationError",
    "APIStatusError",
    "APITimeoutError",
    "BadRequestError",
    "AuthenticationError",
    "PermissionDeniedError",
    "NotFoundError",
    "ConflictError",
    "UnprocessableEntityError",
    "RateLimitError",
    "InternalServerError",
    "KNIRVAPIError",
    "KNIRVValidationError",
]
