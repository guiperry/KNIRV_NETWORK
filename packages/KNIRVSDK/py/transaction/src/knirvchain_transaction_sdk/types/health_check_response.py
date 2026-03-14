# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Optional
from typing_extensions import Literal

from .._models import BaseModel

__all__ = ["HealthCheckResponse"]


class HealthCheckResponse(BaseModel):
    blockchain: Optional[bool] = None
    """Blockchain subsystem status"""

    database: Optional[bool] = None
    """Database subsystem status"""

    message: Optional[str] = None
    """Additional status information"""

    network: Optional[bool] = None
    """Network subsystem status"""

    status: Optional[Literal["healthy", "unhealthy"]] = None
    """Overall health status"""
