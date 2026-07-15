package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

func TestPaymentReturnURL(t *testing.T) {
	oldAddress := system_setting.ServerAddress
	t.Cleanup(func() { system_setting.ServerAddress = oldAddress })

	system_setting.ServerAddress = "https://example.com/"
	if got, want := PaymentReturnURL("/wallet?source=quota"), "https://example.com/wallet?source=quota"; got != want {
		t.Fatalf("PaymentReturnURL() = %q, want %q", got, want)
	}
}
