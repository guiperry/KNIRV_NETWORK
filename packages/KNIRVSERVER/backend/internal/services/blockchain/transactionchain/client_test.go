package transactionchain

import "testing"

type stubBalanceReader struct {
	balance int64
	err     error
}

func (s *stubBalanceReader) GetAccountBalance(string) (int64, error) {
	return s.balance, s.err
}

func TestClientGetAccountBalanceUsesInjectedReader(t *testing.T) {
	client := &Client{}
	client.SetBalanceReader(&stubBalanceReader{balance: 4242})

	balance, err := client.GetAccountBalance("0x1111111111111111111111111111111111111111")
	if err != nil {
		t.Fatalf("expected no error from injected balance reader, got %v", err)
	}
	if balance != 4242 {
		t.Fatalf("expected injected balance 4242, got %d", balance)
	}
}
