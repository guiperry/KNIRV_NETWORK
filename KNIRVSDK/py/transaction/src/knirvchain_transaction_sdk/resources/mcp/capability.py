# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Any, cast

import httpx

from ..._types import NOT_GIVEN, Body, Query, Headers, NotGiven
from ..._utils import maybe_transform, async_maybe_transform
from ..._compat import cached_property
from ..._resource import SyncAPIResource, AsyncAPIResource
from ..._response import (
    to_raw_response_wrapper,
    to_streamed_response_wrapper,
    async_to_raw_response_wrapper,
    async_to_streamed_response_wrapper,
)
from ...types.mcp import (
    CapabilityDescriptor,
    capability_invoke_params,
    capability_update_params,
    capability_prepare_registration_params,
)
from ..._base_client import make_request_options
from ...types.transaction_param import TransactionParam
from ...types.mcp.capability_descriptor import CapabilityDescriptor
from ...types.mcp.capability_invoke_response import CapabilityInvokeResponse
from ...types.mcp.capability_update_response import CapabilityUpdateResponse
from ...types.mcp.capability_descriptor_param import CapabilityDescriptorParam
from ...types.mcp.capability_prepare_registration_response import CapabilityPrepareRegistrationResponse
from ...types.mcp.capability_retrieve_invocations_response import CapabilityRetrieveInvocationsResponse

__all__ = ["CapabilityResource", "AsyncCapabilityResource"]


class CapabilityResource(SyncAPIResource):
    @cached_property
    def with_raw_response(self) -> CapabilityResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return CapabilityResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> CapabilityResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return CapabilityResourceWithStreamingResponse(self)

    def retrieve(
        self,
        capability_id: str,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityDescriptor:
        """
        Retrieves a specific capability by ID

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        if not capability_id:
            raise ValueError(f"Expected a non-empty value for `capability_id` but received {capability_id!r}")
        return cast(
            CapabilityDescriptor,
            self._get(
                f"/mcp/capability/{capability_id}",
                options=make_request_options(
                    extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
                ),
                cast_to=cast(
                    Any, CapabilityDescriptor
                ),  # Union types cannot be passed in as arguments in the type system
            ),
        )

    def update(
        self,
        *,
        signed_transaction: TransactionParam,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityUpdateResponse:
        """Submits a pre-signed transaction to update an existing capability.

        The sender
        must be the owner of the capability. The `Transaction.data` within the signed
        transaction should contain `MCPUpdateCapabilityData`.

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._post(
            "/mcp/capability/update",
            body=maybe_transform(
                {"signed_transaction": signed_transaction}, capability_update_params.CapabilityUpdateParams
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityUpdateResponse,
        )

    def invoke(
        self,
        *,
        signed_transaction: TransactionParam,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityInvokeResponse:
        """Submits a pre-signed transaction to invoke an existing capability.

        The
        `Transaction.data` within the signed transaction should contain
        `MCPInvokeCapabilityData`.

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._post(
            "/mcp/capability/invoke",
            body=maybe_transform(
                {"signed_transaction": signed_transaction}, capability_invoke_params.CapabilityInvokeParams
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityInvokeResponse,
        )

    def prepare_registration(
        self,
        *,
        fee: int,
        owner_address: str,
        descriptor: CapabilityDescriptorParam | NotGiven = NOT_GIVEN,
        message: str | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityPrepareRegistrationResponse:
        """
        Allows a client to get the necessary data (e.g., a hash or structured unsigned
        transaction) that needs to be signed to register a new MCP capability. The
        client signs this data locally and then submits the complete, signed transaction
        via the general /transaction endpoint.

        Args:
          fee: Network fee for the registration transaction.

          owner_address: Address of the account that will own the capability.

          message: Additional information

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._post(
            "/mcp/capability/prepare_registration",
            body=maybe_transform(
                {
                    "fee": fee,
                    "owner_address": owner_address,
                    "descriptor": descriptor,
                    "message": message,
                },
                capability_prepare_registration_params.CapabilityPrepareRegistrationParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityPrepareRegistrationResponse,
        )

    def retrieve_invocations(
        self,
        capability_id: str,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityRetrieveInvocationsResponse:
        """
        Lists all invocations of a specific capability

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        if not capability_id:
            raise ValueError(f"Expected a non-empty value for `capability_id` but received {capability_id!r}")
        return self._get(
            f"/mcp/capability/{capability_id}/invocations",
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityRetrieveInvocationsResponse,
        )


class AsyncCapabilityResource(AsyncAPIResource):
    @cached_property
    def with_raw_response(self) -> AsyncCapabilityResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return AsyncCapabilityResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> AsyncCapabilityResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return AsyncCapabilityResourceWithStreamingResponse(self)

    async def retrieve(
        self,
        capability_id: str,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityDescriptor:
        """
        Retrieves a specific capability by ID

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        if not capability_id:
            raise ValueError(f"Expected a non-empty value for `capability_id` but received {capability_id!r}")
        return cast(
            CapabilityDescriptor,
            await self._get(
                f"/mcp/capability/{capability_id}",
                options=make_request_options(
                    extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
                ),
                cast_to=cast(
                    Any, CapabilityDescriptor
                ),  # Union types cannot be passed in as arguments in the type system
            ),
        )

    async def update(
        self,
        *,
        signed_transaction: TransactionParam,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityUpdateResponse:
        """Submits a pre-signed transaction to update an existing capability.

        The sender
        must be the owner of the capability. The `Transaction.data` within the signed
        transaction should contain `MCPUpdateCapabilityData`.

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._post(
            "/mcp/capability/update",
            body=await async_maybe_transform(
                {"signed_transaction": signed_transaction}, capability_update_params.CapabilityUpdateParams
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityUpdateResponse,
        )

    async def invoke(
        self,
        *,
        signed_transaction: TransactionParam,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityInvokeResponse:
        """Submits a pre-signed transaction to invoke an existing capability.

        The
        `Transaction.data` within the signed transaction should contain
        `MCPInvokeCapabilityData`.

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._post(
            "/mcp/capability/invoke",
            body=await async_maybe_transform(
                {"signed_transaction": signed_transaction}, capability_invoke_params.CapabilityInvokeParams
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityInvokeResponse,
        )

    async def prepare_registration(
        self,
        *,
        fee: int,
        owner_address: str,
        descriptor: CapabilityDescriptorParam | NotGiven = NOT_GIVEN,
        message: str | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityPrepareRegistrationResponse:
        """
        Allows a client to get the necessary data (e.g., a hash or structured unsigned
        transaction) that needs to be signed to register a new MCP capability. The
        client signs this data locally and then submits the complete, signed transaction
        via the general /transaction endpoint.

        Args:
          fee: Network fee for the registration transaction.

          owner_address: Address of the account that will own the capability.

          message: Additional information

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._post(
            "/mcp/capability/prepare_registration",
            body=await async_maybe_transform(
                {
                    "fee": fee,
                    "owner_address": owner_address,
                    "descriptor": descriptor,
                    "message": message,
                },
                capability_prepare_registration_params.CapabilityPrepareRegistrationParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityPrepareRegistrationResponse,
        )

    async def retrieve_invocations(
        self,
        capability_id: str,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> CapabilityRetrieveInvocationsResponse:
        """
        Lists all invocations of a specific capability

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        if not capability_id:
            raise ValueError(f"Expected a non-empty value for `capability_id` but received {capability_id!r}")
        return await self._get(
            f"/mcp/capability/{capability_id}/invocations",
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=CapabilityRetrieveInvocationsResponse,
        )


class CapabilityResourceWithRawResponse:
    def __init__(self, capability: CapabilityResource) -> None:
        self._capability = capability

        self.retrieve = to_raw_response_wrapper(
            capability.retrieve,
        )
        self.update = to_raw_response_wrapper(
            capability.update,
        )
        self.invoke = to_raw_response_wrapper(
            capability.invoke,
        )
        self.prepare_registration = to_raw_response_wrapper(
            capability.prepare_registration,
        )
        self.retrieve_invocations = to_raw_response_wrapper(
            capability.retrieve_invocations,
        )


class AsyncCapabilityResourceWithRawResponse:
    def __init__(self, capability: AsyncCapabilityResource) -> None:
        self._capability = capability

        self.retrieve = async_to_raw_response_wrapper(
            capability.retrieve,
        )
        self.update = async_to_raw_response_wrapper(
            capability.update,
        )
        self.invoke = async_to_raw_response_wrapper(
            capability.invoke,
        )
        self.prepare_registration = async_to_raw_response_wrapper(
            capability.prepare_registration,
        )
        self.retrieve_invocations = async_to_raw_response_wrapper(
            capability.retrieve_invocations,
        )


class CapabilityResourceWithStreamingResponse:
    def __init__(self, capability: CapabilityResource) -> None:
        self._capability = capability

        self.retrieve = to_streamed_response_wrapper(
            capability.retrieve,
        )
        self.update = to_streamed_response_wrapper(
            capability.update,
        )
        self.invoke = to_streamed_response_wrapper(
            capability.invoke,
        )
        self.prepare_registration = to_streamed_response_wrapper(
            capability.prepare_registration,
        )
        self.retrieve_invocations = to_streamed_response_wrapper(
            capability.retrieve_invocations,
        )


class AsyncCapabilityResourceWithStreamingResponse:
    def __init__(self, capability: AsyncCapabilityResource) -> None:
        self._capability = capability

        self.retrieve = async_to_streamed_response_wrapper(
            capability.retrieve,
        )
        self.update = async_to_streamed_response_wrapper(
            capability.update,
        )
        self.invoke = async_to_streamed_response_wrapper(
            capability.invoke,
        )
        self.prepare_registration = async_to_streamed_response_wrapper(
            capability.prepare_registration,
        )
        self.retrieve_invocations = async_to_streamed_response_wrapper(
            capability.retrieve_invocations,
        )
