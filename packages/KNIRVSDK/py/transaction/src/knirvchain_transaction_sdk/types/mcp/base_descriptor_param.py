# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Dict
from typing_extensions import Literal, Annotated, TypedDict

from ..._utils import PropertyInfo

__all__ = ["BaseDescriptorParam"]


class BaseDescriptorParam(TypedDict, total=False):
    id: str
    """Unique identifier"""

    capability_type: Annotated[
        Literal["RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE"], PropertyInfo(alias="capabilityType")
    ]
    """Type of capability"""

    custom_metadata: Annotated[Dict[str, object], PropertyInfo(alias="customMetadata")]
    """Custom metadata"""

    description: str
    """Description of the capability"""

    gas_fee_nrn: Annotated[int, PropertyInfo(alias="gasFeeNRN")]
    """NRN token gas fee for invoking/using this capability"""

    name: str
    """Human-readable name"""

    owner: str
    """Public key or address of the owner/root"""

    timestamp: int
    """Creation/registration timestamp"""

    version: str
    """Version string"""
