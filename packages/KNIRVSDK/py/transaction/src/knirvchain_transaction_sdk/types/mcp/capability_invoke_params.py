# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing_extensions import Required, Annotated, TypedDict

from ..._utils import PropertyInfo
from ..transaction_param import TransactionParam

__all__ = ["CapabilityInvokeParams"]


class CapabilityInvokeParams(TypedDict, total=False):
    signed_transaction: Required[Annotated[TransactionParam, PropertyInfo(alias="signedTransaction")]]
