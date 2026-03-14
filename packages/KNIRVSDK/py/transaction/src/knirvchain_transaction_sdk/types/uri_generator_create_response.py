# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Optional

from .._models import BaseModel

__all__ = ["UriGeneratorCreateResponse"]


class UriGeneratorCreateResponse(BaseModel):
    message: Optional[str] = None
    """Additional information"""

    resource_id: Optional[str] = None
    """Resource ID"""

    success: Optional[bool] = None
    """Whether the operation was successful"""

    uri: Optional[str] = None
    """Generated URI"""
