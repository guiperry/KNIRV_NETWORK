# KNIRVENGINE Tests

This directory contains integration tests and test utilities for the KNIRVENGINE.

## Integration Tests

### Agent Builder Integration Test

**File:** `test_agent_builder_integration.go`

This integration test validates the complete agent building pipeline including:

1. **Template Management**: Tests the `/templates` endpoint to retrieve available agent templates
2. **Agent Creation**: Creates a new agent via the `/agents` endpoint
3. **Plugin Building**: Builds an agent plugin using the `/agents/{id}/build` endpoint
4. **Build Status**: Checks build status and progress
5. **Sub-Agent Functionality**: Tests spawning and managing sub-agents
6. **Plugin Retrieval**: Lists compiled plugins via the `/plugins` endpoint

#### Running the Integration Test

1. Start the KNIRVENGINE server:
   ```bash
   make run
   # or
   go run .
   ```

2. In a separate terminal, run the integration test:
   ```bash
   cd tests
   go run test_agent_builder_integration.go
   ```

#### Expected Output

The test will output status codes and responses for each API endpoint tested:

```
Testing Agent Builder Integration...

1. Testing GET /templates
Status: 200
Response: {"templates":[...]}

2. Testing agent creation and plugin building
Create Agent Status: 201
Create Agent Response: {"agent":{"id":"...","name":"Test Agent",...}}
Build Agent Status: 202
Build Agent Response: {"build_id":"...","status":"started"}
Build Status: 200
Build Status Response: {"status":"completed","progress":100}

3. Testing sub-agent functionality
Spawn Sub-Agent Status: 201
Spawn Sub-Agent Response: {"sub_agent":{"id":"...","template":"python"}}
Get Sub-Agents Status: 200
Get Sub-Agents Response: {"sub_agents":[...]}

4. Testing GET /plugins
Status: 200
Response: {"plugins":[...]}

Agent Builder Integration Test Complete!
```

#### Test Configuration

The test is configured to connect to:
- **Base URL**: `http://localhost:8081/api/v1`
- **Default Agent**: Creates a "Test Agent" with standard template
- **Sub-Agent Template**: Uses Python template for sub-agent testing

#### Troubleshooting

- **Connection Refused**: Ensure the KNIRVENGINE server is running on port 8081
- **404 Errors**: Verify the API endpoints are properly implemented
- **Build Failures**: Check that the AgentBuilder service is properly configured
- **Timeout Issues**: The test includes small delays between operations; increase if needed

## Adding New Tests

When adding new integration tests:

1. Create a new `.go` file in this directory
2. Use `package main` and include a `main()` function
3. Follow the naming convention: `test_<feature>_integration.go`
4. Document the test purpose and usage in this README
5. Include proper error handling and status reporting

## Test Data

Test data and fixtures should be placed in subdirectories:
- `fixtures/` - Static test data files
- `mocks/` - Mock data for testing
- `configs/` - Test-specific configuration files

## Best Practices

- Always check HTTP status codes
- Include meaningful error messages
- Test both success and failure scenarios
- Clean up any test data created during testing
- Use timeouts for operations that might hang
- Document expected behavior and outputs
