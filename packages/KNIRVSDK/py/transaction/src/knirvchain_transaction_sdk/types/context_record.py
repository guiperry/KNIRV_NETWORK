# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Dict, Optional
from datetime import datetime
from typing_extensions import Literal

from pydantic import Field as FieldInfo

from .._models import BaseModel

__all__ = ["ContextRecord"]


class ContextRecord(BaseModel):
    id: Optional[str] = None
    """Unique identifier for the context record"""

    capability_id: Optional[str] = FieldInfo(alias="capabilityID", default=None)
    """Identifier of the MCP capability involved"""

    created_at: Optional[datetime] = FieldInfo(alias="createdAt", default=None)

    details: Optional[Dict[str, object]] = None
    """Primitive-specific information"""

    initiator: Optional[str] = None
    """Public key or address of the entity initiating the context"""

    input_hash: Optional[str] = FieldInfo(alias="inputHash", default=None)
    """Optional hash of the input data to the interaction"""

    interaction_type: Optional[
        Literal[
            "CAPABILITY_REGISTRATION",
            "CAPABILITY_INVOCATION",
            "CAPABILITY_UPDATE",
            "GENERAL_MESSAGE",
            "TOOL_INVOCATION",
            "PROMPT_USAGE",
            "RESOURCE_ACCESS",
            "PLUGIN_EXECUTION",
            "SAMPLING_REQUEST_SENT",
            "SAMPLING_RESPONSE_RECEIVED",
            "MEMORY_WRITE",
        ]
    ] = FieldInfo(alias="interactionType", default=None)
    """Type of interaction"""

    output_hash: Optional[str] = FieldInfo(alias="outputHash", default=None)
    """Optional hash of the output data from the interaction"""

    signature: Optional[str] = None
    """Cryptographic signature of the ContextRecord content by the initiator"""

    status: Optional[Literal["pending", "success", "failed", "processing"]] = None

    timestamp: Optional[int] = None
    """Timestamp of the event"""

    updated_at: Optional[datetime] = FieldInfo(alias="updatedAt", default=None)
