package lean

import (
	"fmt"
	"os/exec"
)

// FakeRunner is a test double that returns canned stdout/stderr for Lean
// invocations. It is used only in unit tests.
type FakeRunner struct {
	Responses []FakeResponse
	Index     int
}

// FakeResponse describes one canned invocation result.
type FakeResponse struct {
	Stdout []byte
	Stderr []byte
	Err    error
}

// Next returns the next canned response, cycling if there are fewer calls than
// responses.
func (f *FakeRunner) Next() FakeResponse {
	if len(f.Responses) == 0 {
		return FakeRunnerSuccess()
	}
	resp := f.Responses[f.Index%len(f.Responses)]
	f.Index++
	return resp
}

// Run returns the next canned response.
func (f *FakeRunner) Run(cmd *exec.Cmd) ([]byte, []byte, error) {
	resp := f.Next()
	return resp.Stdout, resp.Stderr, resp.Err
}

// FakeRunnerSuccess returns a FakeResponse that simulates a successful Lean
// verification with status FORMALLY_VERIFIED.
func FakeRunnerSuccess() FakeResponse {
	return FakeResponse{
		Stdout: []byte("KNIRV_STATUS=FORMALLY_VERIFIED\n"),
		Stderr: []byte(""),
		Err:    nil,
	}
}

// FakeRunnerRejected returns a FakeResponse that simulates a Lean rejection.
func FakeRunnerRejected(diagnostic string) FakeResponse {
	return FakeResponse{
		Stdout: []byte(fmt.Sprintf("KNIRV_STATUS=FORMALLY_REJECTED\n%s\n", diagnostic)),
		Stderr: []byte(diagnostic),
		Err:    fmt.Errorf("lean exited with code 1"),
	}
}

// FakeRunnerParseError returns a FakeResponse with malformed output.
func FakeRunnerParseError() FakeResponse {
	return FakeResponse{
		Stdout: []byte("some unrelated lean output\n"),
		Stderr: []byte(""),
		Err:    nil,
	}
}
