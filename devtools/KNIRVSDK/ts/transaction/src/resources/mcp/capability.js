// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../../core/resource';
import { path } from '../../internal/utils/path';
export class Capability extends APIResource {
    /**
     * Retrieves a specific capability by ID
     */
    retrieve(capabilityID, options) {
        return this._client.get(path `/mcp/capability/${capabilityID}`, options);
    }
    /**
     * Submits a pre-signed transaction to update an existing capability. The sender
     * must be the owner of the capability. The `Transaction.data` within the signed
     * transaction should contain `MCPUpdateCapabilityData`.
     */
    update(body, options) {
        return this._client.post('/mcp/capability/update', { body, ...options });
    }
    /**
     * Submits a pre-signed transaction to invoke an existing capability. The
     * `Transaction.data` within the signed transaction should contain
     * `MCPInvokeCapabilityData`.
     */
    invoke(body, options) {
        return this._client.post('/mcp/capability/invoke', { body, ...options });
    }
    /**
     * Allows a client to get the necessary data (e.g., a hash or structured unsigned
     * transaction) that needs to be signed to register a new MCP capability. The
     * client signs this data locally and then submits the complete, signed transaction
     * via the general /transaction endpoint.
     */
    prepareRegistration(body, options) {
        return this._client.post('/mcp/capability/prepare_registration', { body, ...options });
    }
    /**
     * Lists all invocations of a specific capability
     */
    retrieveInvocations(capabilityID, options) {
        return this._client.get(path `/mcp/capability/${capabilityID}/invocations`, options);
    }
}
