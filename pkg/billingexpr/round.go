package billingexpr

import "github.com/QuantumNous/new-api/common"

// QuotaRound delegates to common.QuotaRound (tiered billing must use this).
func QuotaRound(f float64) int {
	return common.QuotaRound(f)
}

// QuotaRoundStrict rejects an unrepresentable pre-consume estimate.
func QuotaRoundStrict(f float64) (int, error) {
	return common.QuotaRoundStrict(f)
}
