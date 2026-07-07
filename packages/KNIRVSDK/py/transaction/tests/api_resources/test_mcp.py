# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

import os
from typing import Any, cast

import pytest

from tests.utils import assert_matches_type
from knirvchain_transaction_sdk import KnirvchainTransactionSDK, AsyncKnirvchainTransactionSDK
from knirvchain_transaction_sdk.types import (
    ContextRecord,
    McpRetrieveContextsResponse,
    McpRetrieveCapabilitiesResponse,
)

base_url = os.environ.get("TEST_API_BASE_URL", "http://127.0.0.1:4010")


class TestMcp:
    parametrize = pytest.mark.parametrize("client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        mcp = client.mcp.retrieve(
            "context_id",
        )
        assert_matches_type(ContextRecord, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.with_raw_response.retrieve(
            "context_id",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        mcp = response.parse()
        assert_matches_type(ContextRecord, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.with_streaming_response.retrieve(
            "context_id",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            mcp = response.parse()
            assert_matches_type(ContextRecord, mcp, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    def test_path_params_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        with pytest.raises(ValueError, match=r"Expected a non-empty value for `context_id` but received ''"):
            client.mcp.with_raw_response.retrieve(
                "",
            )

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve_capabilities(self, client: KnirvchainTransactionSDK) -> None:
        mcp = client.mcp.retrieve_capabilities()
        assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve_capabilities_with_all_params(self, client: KnirvchainTransactionSDK) -> None:
        mcp = client.mcp.retrieve_capabilities(
            owner="owner",
            type="RESOURCE",
        )
        assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_retrieve_capabilities(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.with_raw_response.retrieve_capabilities()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        mcp = response.parse()
        assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_retrieve_capabilities(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.with_streaming_response.retrieve_capabilities() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            mcp = response.parse()
            assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve_contexts(self, client: KnirvchainTransactionSDK) -> None:
        mcp = client.mcp.retrieve_contexts()
        assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve_contexts_with_all_params(self, client: KnirvchainTransactionSDK) -> None:
        mcp = client.mcp.retrieve_contexts(
            capability_id="capabilityId",
            initiator="initiator",
            interaction_type="TOOL_INVOCATION",
        )
        assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_retrieve_contexts(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.with_raw_response.retrieve_contexts()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        mcp = response.parse()
        assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_retrieve_contexts(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.with_streaming_response.retrieve_contexts() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            mcp = response.parse()
            assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

        assert cast(Any, response.is_closed) is True


class TestAsyncMcp:
    parametrize = pytest.mark.parametrize("async_client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        mcp = await async_client.mcp.retrieve(
            "context_id",
        )
        assert_matches_type(ContextRecord, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.with_raw_response.retrieve(
            "context_id",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        mcp = await response.parse()
        assert_matches_type(ContextRecord, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.with_streaming_response.retrieve(
            "context_id",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            mcp = await response.parse()
            assert_matches_type(ContextRecord, mcp, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    async def test_path_params_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        with pytest.raises(ValueError, match=r"Expected a non-empty value for `context_id` but received ''"):
            await async_client.mcp.with_raw_response.retrieve(
                "",
            )

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve_capabilities(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        mcp = await async_client.mcp.retrieve_capabilities()
        assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve_capabilities_with_all_params(
        self, async_client: AsyncKnirvchainTransactionSDK
    ) -> None:
        mcp = await async_client.mcp.retrieve_capabilities(
            owner="owner",
            type="RESOURCE",
        )
        assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_retrieve_capabilities(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.with_raw_response.retrieve_capabilities()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        mcp = await response.parse()
        assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_retrieve_capabilities(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.with_streaming_response.retrieve_capabilities() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            mcp = await response.parse()
            assert_matches_type(McpRetrieveCapabilitiesResponse, mcp, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve_contexts(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        mcp = await async_client.mcp.retrieve_contexts()
        assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve_contexts_with_all_params(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        mcp = await async_client.mcp.retrieve_contexts(
            capability_id="capabilityId",
            initiator="initiator",
            interaction_type="TOOL_INVOCATION",
        )
        assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_retrieve_contexts(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.with_raw_response.retrieve_contexts()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        mcp = await response.parse()
        assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_retrieve_contexts(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.with_streaming_response.retrieve_contexts() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            mcp = await response.parse()
            assert_matches_type(McpRetrieveContextsResponse, mcp, path=["response"])

        assert cast(Any, response.is_closed) is True
