package query

import (
	"testing"
)

func TestNewQueryProcessor(t *testing.T) {
	qp := NewQueryProcessor(nil, nil, nil, nil, nil, nil)
	if qp == nil {
		t.Fatal("Expected non-nil query processor")
	}
}
