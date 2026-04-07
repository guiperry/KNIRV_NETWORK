# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Optional

from pydantic import Field as FieldInfo

from ..._models import BaseModel

__all__ = ["CapabilityInvokeResponse"]


class CapabilityInvokeResponse(BaseModel):
    context_id: Optional[str] = FieldInfo(alias="contextID", default=None)

    message: Optional[str] = None

    transaction_hash: Optional[str] = FieldInfo(alias="transactionHash", default=None)
