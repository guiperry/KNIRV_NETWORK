

---

**Source**: KNIRVROOT/CLI/docs/test_report.md

Here's a summary of the other failing tests and likely causes/fixes:

TestAPIClient/SubmitTransaction (core/api_client_test.go):

Error: Not equal: expected: "test-txn" actual: ""
Likely Cause: The mock HTTP server in your test is returning {"txn_hash": "test-txn", ...}, but your API client's response struct might be expecting a field tagged as json:"transactionHash". Ensure the JSON field name in the mock response matches the struct tag in your client's code for unmarshalling the transaction hash. The log for TestTransactionFlow shows a response with "transactionHash", suggesting this might be the expected field name.
TestFileManager/GenerateFileReference (core/file_manager_test.go):

Error: Not equal: expected: 47 actual: 54 (length mismatch)
Likely Cause: The logic for generating file references (e.g., format string, hash length, encoding of components) might have changed, leading to a different output length. Update the expected length in the test or verify if the generation logic change was intended.
TestWalletManager/ParsePrivateKey (core/wallet_manager_test.go):

Error: An error is expected but got nil.
Likely Cause: The test input for an "invalid" private key might not be invalid enough to cause an error from the underlying parsing functions (e.g., crypto.HexToECDSA), or your ParsePrivateKey function is missing some validation. Review the input or enhance validation in ParsePrivateKey.
TestSplitIntoChunks (pkg/inference/context_manager_test.go):

Error: Expected multiple chunks, got 1
Likely Cause: The input text might be too short for the configured chunk size, the chunking strategy (logged as 1) might imply a single chunk, or there's a bug in the splitting logic. Verify the test setup and chunking strategy behavior.
TestProcessLargePrompt (pkg/inference/context_manager_test.go):

Error: Not all chunks were processed correctly in sequential mode... (content mismatch)
Likely Cause: Differences in newlines or formatting between the actual combined output and the expected output. This could be an issue in the summary generation, chunk combination logic, or the test's string comparison.
TestProcessLargePromptWithError (pkg/inference/context_manager_test.go):

Errors: Failures in error propagation and partial result formatting for both parallel and sequential modes.
Likely Cause: The error handling within ContextManager when a chunk processing fails needs review. It might not be returning errors or formatting error placeholders as expected.
TestAPIClientIntegration/PrepareCapabilityRegistration (test/integration/api_client_integration_test.go):

Error: plugin file validation failed: file does not exist: mock-plugin.so
Likely Cause: The test setup needs to ensure the mock-plugin.so file is created or available in the expected path during the test run.
TestAPIClientTimeout (test/unit/api_client_test.go:107):

Error: "...context deadline exceeded (Client.Timeout exceeded while awaiting headers)" does not contain "timeout"
Likely Cause: The assertion assert.Contains(t, err.Error(), "timeout") is case-sensitive. The error string contains "Timeout" (capital 'T').
Fix: Change the assertion to assert.Contains(t, err.Error(), "Timeout") or use a case-insensitive comparison like assert.True(t, strings.Contains(strings.ToLower(err.Error()), "timeout")).
TestUpdate (ui/app_test.go):

Errors: Multiple Should be true failures and view content mismatch ("No screen set" does not contain "Mock Screen Content").
Likely Cause: Issues in the Bubble Tea application's Update logic. Key presses or messages might not be updating the model correctly, or the View isn't rendering the expected state. The "Mock Screen Content" error suggests a screen isn't being set/rendered as intended.
TestNewPasswordField (ui/components/forms_test.go):

Error: Not equal: expected: int(1) actual: textinput.EchoMode(1)
Likely Cause: Type mismatch. textinput.EchoMode is a distinct type.
Fix: Compare with the correct type and value, e.g., assert.Equal(t, textinput.EchoPassword, field.Input.EchoMode) (assuming 1 corresponds to EchoPassword).
TestSetValidator (ui/components/forms_test.go):

Error: Invalid operation: ... (cannot take func type as argument)
Likely Cause: Functions in Go cannot be compared directly with ==.
Fix: Assert that the validator field is not nil (e.g., assert.NotNil(t, formField.Validator)) or test the validator's behavior by calling it.
TestAddField (ui/components/forms_test.go):

Error: Struct field mismatches for focus, Blink, blinkTag.
Likely Cause: The default state (especially focus) of a newly added form field differs from the test's expectation. Often, the first field added gets focus. Adjust the test's expected struct to match the actual behavior or modify the AddField logic if the current behavior is incorrect. The diff indicates the actual field has focus: true while expected was focus: false.
Addressing the initConfig issue should clear up a significant number of your test failures. For the remaining ones, you'll need to investigate each based on the specific error messages and the context of the test and code involved, using the pointers above. 

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
