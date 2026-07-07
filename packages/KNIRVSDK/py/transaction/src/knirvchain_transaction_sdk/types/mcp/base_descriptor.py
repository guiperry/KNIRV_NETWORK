# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Dict, Optional
from typing_extensions import Literal

from pydantic import Field as FieldInfo

from ..._models import BaseModel

__all__ = ["BaseDescriptor"]


class BaseDescriptor(BaseModel):
    id: Optional[str] = None
    """Unique identifier"""

    capability_type: Optional[Literal["RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE"]] = FieldInfo(
        alias="capabilityType", default=None
    )
    """Type of capability"""

    custom_metadata: Optional[Dict[str, object]] = FieldInfo(alias="customMetadata", default=None)
    """Custom metadata"""

    description: Optional[str] = None
    """Description of the capability"""

    gas_fee_nrn: Optional[int] = FieldInfo(alias="gasFeeNRN", default=None)
    """NRN token gas fee for invoking/using this capability"""

    name: Optional[str] = None
    """Human-readable name"""

    owner: Optional[str] = None
    """Public key or address of the owner/root"""

    timestamp: Optional[int] = None
    """Creation/registration timestamp"""

    version: Optional[str] = None
    """Version string"""
