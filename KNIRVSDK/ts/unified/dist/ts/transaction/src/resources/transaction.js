// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class TransactionResource extends APIResource {
    /**
     * Submits a pre-signed transaction to the blockchain network. This endpoint is
     * used for various transaction types, including standard transfers and MCP
     * operations like capability registration (after preparation), invocation, or
     * updates. The `Transaction.type` field and `Transaction.data` structure will
     * determine how it's processed.
     *
     * @example
     * ```ts
     * const response = await client.transaction.submit({
     *   id: '0xabcdef123456...',
     *   fee: 'fee',
     *   from: 'from',
     *   public_key: 'public_key',
     *   signature: 'U3RhaW5sZXNzIHJvY2tz',
     *   timestamp: 0,
     *   type: 'type',
     *   version: 1,
     * });
     * ```
     */
    submit(body, options) {
        return this._client.post('/transaction', { body, ...options });
    }
}
