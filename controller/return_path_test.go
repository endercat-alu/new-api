package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

func TestPaymentReturnPath(t *testing.T) {
	oldAddress := system_setting.ServerAddress
	t.Cleanup(func() { system_setting.ServerAddress = oldAddress })

	tests := []struct {
		name    string
		address string
		suffix  string
		want    string
	}{
		{name: "path", address: "https://example.com", suffix: "/wallet", want: "https://example.com/wallet"},
		{name: "trailing slash", address: "https://example.com/", suffix: "/usage-logs", want: "https://example.com/usage-logs"},
		{name: "query", address: "https://example.com///", suffix: "/wallet?pay=success", want: "https://example.com/wallet?pay=success"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system_setting.ServerAddress = test.address
			if got := paymentReturnPath(test.suffix); got != test.want {
				t.Fatalf("paymentReturnPath() = %q, want %q", got, test.want)
			}
		})
	}
}
