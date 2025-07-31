

---

**Source**: KNIRVROUTER/docs/URI_Generation.md

# KNIRVCHAIN Root Chain - URI Generation API Specification

This document outlines the process for peer chains (verifyers) to interact with a KNIRVCHAIN root chain node to request and register new unique resource identifiers (URIs).

## Endpoint

The primary endpoint for requesting URIs is:

`/uriGenerator`

## Method

Requests to this endpoint **MUST** use the `POST` HTTP method. Other methods (e.g., `GET`) will result in a `405 Method Not Allowed` error.

## Request Body

The request body **MUST** be in JSON format. It can contain an optional field:

*   `desired_id` (string, optional): If the requesting chain wishes to use a specific identifier for the resource.
    *   **Constraints:**
        *   Length must be between 3 and 64 characters (inclusive).
        *   Must not contain special characters like `/`, `.`, `?`, `&`, or spaces.
    *   If this field is provided, the root chain will check its availability.
*   **Empty Body / No `desired_id`:** If the request body is empty (`{}`), `null`, or the `desired_id` field is missing or empty (`""`), the root chain will generate a unique UUID v4 as the identifier.

**Example Request (Requesting a specific ID):**

```json
{
  "desired_id": "my-unique-resource-123"
}


---

<div class="footer-links">


© 2025 KNIRV Network
</div>
