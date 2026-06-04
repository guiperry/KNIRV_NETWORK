# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Dict, Union, Optional

import httpx

from ..types import transaction_submit_params
from .._types import NOT_GIVEN, Body, Query, Headers, NotGiven, Base64FileInput
from .._utils import maybe_transform, async_maybe_transform
from .._compat import cached_property
from .._resource import SyncAPIResource, AsyncAPIResource
from .._response import (
    to_raw_response_wrapper,
    to_streamed_response_wrapper,
    async_to_raw_response_wrapper,
    async_to_streamed_response_wrapper,
)
from .._base_client import make_request_options
from ..types.transaction_submit_response import TransactionSubmitResponse

__all__ = ["TransactionResource", "AsyncTransactionResource"]


class TransactionResource(SyncAPIResource):
    @cached_property
    def with_raw_response(self) -> TransactionResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return TransactionResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> TransactionResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return TransactionResourceWithStreamingResponse(self)

    def submit(
        self,
        *,
        id: str,
        fee: str,
        from_: str,
        public_key: str,
        signature: Union[str, Base64FileInput],
        timestamp: int,
        type: str,
        version: int,
        data: Dict[str, object] | NotGiven = NOT_GIVEN,
        status: str | NotGiven = NOT_GIVEN,
        to: Optional[str] | NotGiven = NOT_GIVEN,
        transaction_hash: str | NotGiven = NOT_GIVEN,
        value: str | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> TransactionSubmitResponse:
        """Submits a pre-signed transaction to the blockchain network.

        This endpoint is
        used for various transaction types, including standard transfers and MCP
        operations like capability registration (after preparation), invocation, or
        updates. The `Transaction.type` field and `Transaction.data` structure will
        determine how it's processed.

        Args:
          id: Transaction hash/ID.

          fee: NRN token gas fee paid for this transaction

          from_: Sender address

          public_key: Public key of the sender

          signature: Cryptographic signature

          timestamp: Unix timestamp (nanoseconds) when the transaction was created

          type: Transaction type for MCP transactions

          data: Transaction payload, structure depends on transaction type. For MCP ops, this
              will be JSON of MCPRegisterCapabilityData, MCPInvokeCapabilityData, etc.

          status: Transaction status (PENDING, SUCCESS, FAILED)

          to: Recipient address

          transaction_hash: Hash of the transaction

          value: Amount of NRN transferred.

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._post(
            "/transaction",
            body=maybe_transform(
                {
                    "id": id,
                    "fee": fee,
                    "from_": from_,
                    "public_key": public_key,
                    "signature": signature,
                    "timestamp": timestamp,
                    "type": type,
                    "version": version,
                    "data": data,
                    "status": status,
                    "to": to,
                    "transaction_hash": transaction_hash,
                    "value": value,
                },
                transaction_submit_params.TransactionSubmitParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=TransactionSubmitResponse,
        )


class AsyncTransactionResource(AsyncAPIResource):
    @cached_property
    def with_raw_response(self) -> AsyncTransactionResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return AsyncTransactionResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> AsyncTransactionResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return AsyncTransactionResourceWithStreamingResponse(self)

    async def submit(
        self,
        *,
        id: str,
        fee: str,
        from_: str,
        public_key: str,
        signature: Union[str, Base64FileInput],
        timestamp: int,
        type: str,
        version: int,
        data: Dict[str, object] | NotGiven = NOT_GIVEN,
        status: str | NotGiven = NOT_GIVEN,
        to: Optional[str] | NotGiven = NOT_GIVEN,
        transaction_hash: str | NotGiven = NOT_GIVEN,
        value: str | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> TransactionSubmitResponse:
        """Submits a pre-signed transaction to the blockchain network.

        This endpoint is
        used for various transaction types, including standard transfers and MCP
        operations like capability registration (after preparation), invocation, or
        updates. The `Transaction.type` field and `Transaction.data` structure will
        determine how it's processed.

        Args:
          id: Transaction hash/ID.

          fee: NRN token gas fee paid for this transaction

          from_: Sender address

          public_key: Public key of the sender

          signature: Cryptographic signature

          timestamp: Unix timestamp (nanoseconds) when the transaction was created

          type: Transaction type for MCP transactions

          data: Transaction payload, structure depends on transaction type. For MCP ops, this
              will be JSON of MCPRegisterCapabilityData, MCPInvokeCapabilityData, etc.

          status: Transaction status (PENDING, SUCCESS, FAILED)

          to: Recipient address

          transaction_hash: Hash of the transaction

          value: Amount of NRN transferred.

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._post(
            "/transaction",
            body=await async_maybe_transform(
                {
                    "id": id,
                    "fee": fee,
                    "from_": from_,
                    "public_key": public_key,
                    "signature": signature,
                    "timestamp": timestamp,
                    "type": type,
                    "version": version,
                    "data": data,
                    "status": status,
                    "to": to,
                    "transaction_hash": transaction_hash,
                    "value": value,
                },
                transaction_submit_params.TransactionSubmitParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=TransactionSubmitResponse,
        )


class TransactionResourceWithRawResponse:
    def __init__(self, transaction: TransactionResource) -> None:
        self._transaction = transaction

        self.submit = to_raw_response_wrapper(
            transaction.submit,
        )


class AsyncTransactionResourceWithRawResponse:
    def __init__(self, transaction: AsyncTransactionResource) -> None:
        self._transaction = transaction

        self.submit = async_to_raw_response_wrapper(
            transaction.submit,
        )


class TransactionResourceWithStreamingResponse:
    def __init__(self, transaction: TransactionResource) -> None:
        self._transaction = transaction

        self.submit = to_streamed_response_wrapper(
            transaction.submit,
        )


class AsyncTransactionResourceWithStreamingResponse:
    def __init__(self, transaction: AsyncTransactionResource) -> None:
        self._transaction = transaction

        self.submit = async_to_streamed_response_wrapper(
            transaction.submit,
        )
