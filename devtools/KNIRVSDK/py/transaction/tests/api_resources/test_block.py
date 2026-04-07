# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

import os
from typing import Any, cast

import pytest

from tests.utils import assert_matches_type
from knirvchain_transaction_sdk import KnirvchainTransactionSDK, AsyncKnirvchainTransactionSDK
from knirvchain_transaction_sdk.types import BlockSubmitResponse

base_url = os.environ.get("TEST_API_BASE_URL", "http://127.0.0.1:4010")


class TestBlock:
    parametrize = pytest.mark.parametrize("client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    def test_method_submit(self, client: KnirvchainTransactionSDK) -> None:
        block = client.block.submit()
        assert_matches_type(BlockSubmitResponse, block, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_method_submit_with_all_params(self, client: KnirvchainTransactionSDK) -> None:
        block = client.block.submit(
            block_number=0,
            hash="U3RhaW5sZXNzIHJvY2tz",
            nonce=0,
            prev_hash="U3RhaW5sZXNzIHJvY2tz",
            proposer_address="proposer_address",
            timestamp=0,
            transactions=[
                {
                    "id": "0xabcdef123456...",
                    "fee": "fee",
                    "from": "from",
                    "public_key": "public_key",
                    "signature": "U3RhaW5sZXNzIHJvY2tz",
                    "timestamp": 0,
                    "type": "type",
                    "version": 1,
                    "data": {"foo": "bar"},
                    "status": "status",
                    "to": "to",
                    "transaction_hash": "transaction_hash",
                    "value": "1000000000000000000",
                }
            ],
        )
        assert_matches_type(BlockSubmitResponse, block, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_submit(self, client: KnirvchainTransactionSDK) -> None:
        response = client.block.with_raw_response.submit()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        block = response.parse()
        assert_matches_type(BlockSubmitResponse, block, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_submit(self, client: KnirvchainTransactionSDK) -> None:
        with client.block.with_streaming_response.submit() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            block = response.parse()
            assert_matches_type(BlockSubmitResponse, block, path=["response"])

        assert cast(Any, response.is_closed) is True


class TestAsyncBlock:
    parametrize = pytest.mark.parametrize("async_client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_submit(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        block = await async_client.block.submit()
        assert_matches_type(BlockSubmitResponse, block, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_submit_with_all_params(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        block = await async_client.block.submit(
            block_number=0,
            hash="U3RhaW5sZXNzIHJvY2tz",
            nonce=0,
            prev_hash="U3RhaW5sZXNzIHJvY2tz",
            proposer_address="proposer_address",
            timestamp=0,
            transactions=[
                {
                    "id": "0xabcdef123456...",
                    "fee": "fee",
                    "from": "from",
                    "public_key": "public_key",
                    "signature": "U3RhaW5sZXNzIHJvY2tz",
                    "timestamp": 0,
                    "type": "type",
                    "version": 1,
                    "data": {"foo": "bar"},
                    "status": "status",
                    "to": "to",
                    "transaction_hash": "transaction_hash",
                    "value": "1000000000000000000",
                }
            ],
        )
        assert_matches_type(BlockSubmitResponse, block, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_submit(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.block.with_raw_response.submit()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        block = await response.parse()
        assert_matches_type(BlockSubmitResponse, block, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_submit(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.block.with_streaming_response.submit() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            block = await response.parse()
            assert_matches_type(BlockSubmitResponse, block, path=["response"])

        assert cast(Any, response.is_closed) is True
