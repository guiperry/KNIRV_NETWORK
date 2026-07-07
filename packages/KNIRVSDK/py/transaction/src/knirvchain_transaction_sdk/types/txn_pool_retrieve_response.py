# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import List
from typing_extensions import TypeAlias

from .transaction import Transaction

__all__ = ["TxnPoolRetrieveResponse"]

TxnPoolRetrieveResponse: TypeAlias = List[Transaction]
