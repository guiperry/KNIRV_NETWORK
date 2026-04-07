// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../../core/resource';
import * as CapabilityAPI from './capability';
import { Capability, } from './capability';
import { path } from '../../internal/utils/path';
export class Mcp extends APIResource {
    constructor() {
        super(...arguments);
        this.capability = new CapabilityAPI.Capability(this._client);
    }
    /**
     * Retrieves a specific context record by ID
     */
    retrieve(contextID, options) {
        return this._client.get(path `/mcp/context/${contextID}`, options);
    }
    /**
     * Lists all registered capabilities
     */
    retrieveCapabilities(query = {}, options) {
        return this._client.get('/mcp/capabilities', { query, ...options });
    }
    /**
     * Lists all context records
     */
    retrieveContexts(query = {}, options) {
        return this._client.get('/mcp/contexts', { query, ...options });
    }
}
Mcp.Capability = Capability;
