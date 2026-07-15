package common

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// Quota columns are int32; conversions must saturate, never wrap into a credit.
const (
	MaxQuota = math.MaxInt32
	MinQuota = math.MinInt32
)

type QuotaClampKind string

const (
	QuotaClampOverflow  QuotaClampKind = "overflow"
	QuotaClampUnderflow QuotaClampKind = "underflow"
	QuotaClampNaN       QuotaClampKind = "nan"
)

// QuotaClamp is a saturated int32 conversion for admin_info audit (nil if in range).
type QuotaClamp struct {
	Op       string         `json:"op"`       // "QuotaFromFloat" | "QuotaRound" | "QuotaFromDecimal"
	Kind     QuotaClampKind `json:"kind"`     // "overflow" | "underflow" | "nan"
	Original float64        `json:"original"` // best-effort pre-clamp value (decimal -> float64 approx)
	Clamped  int            `json:"clamped"`  // the saturated result actually used
}

func (c *QuotaClamp) Error() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("quota conversion (%s) %s: original=%g, clamped=%d", c.Op, c.Kind, c.Original, c.Clamped)
}

// AuditMap is the admin_info.quota_saturation marker shape shared by all billing paths.
func (c *QuotaClamp) AuditMap() map[string]interface{} {
	if c == nil {
		return nil
	}
	return map[string]interface{}{
		"op":       c.Op,
		"kind":     c.Kind,
		"original": c.Original,
		"clamped":  c.Clamped,
	}
}

// saturateQuota clamps to int32, logs on clamp/NaN, and returns *QuotaClamp when saturated.
func saturateQuota(value float64, op string) (int, *QuotaClamp) {
	var clamp *QuotaClamp
	switch {
	case math.IsNaN(value):
		clamp = &QuotaClamp{Op: op, Kind: QuotaClampNaN, Original: value, Clamped: 0}
	case value >= MaxQuota:
		clamp = &QuotaClamp{Op: op, Kind: QuotaClampOverflow, Original: value, Clamped: MaxQuota}
	case value <= MinQuota:
		clamp = &QuotaClamp{Op: op, Kind: QuotaClampUnderflow, Original: value, Clamped: MinQuota}
	default:
		return int(value), nil
	}
	SysError(clamp.Error())
	return clamp.Clamped, clamp
}

func strictQuota(quota int, clamp *QuotaClamp) (int, error) {
	if clamp != nil {
		return 0, clamp
	}
	return quota, nil
}

// QuotaFromFloat truncates toward zero with int32 saturation.
func QuotaFromFloat(value float64) int {
	quota, _ := QuotaFromFloatChecked(value)
	return quota
}

func QuotaFromFloatChecked(value float64) (int, *QuotaClamp) {
	return saturateQuota(value, "QuotaFromFloat")
}

// QuotaFromFloatStrict fails instead of returning a saturated pre-consume value.
func QuotaFromFloatStrict(value float64) (int, error) {
	return strictQuota(QuotaFromFloatChecked(value))
}

// QuotaRound is half-away-from-zero with int32 saturation (required for tiered billing).
func QuotaRound(value float64) int {
	quota, _ := QuotaRoundChecked(value)
	return quota
}

func QuotaRoundChecked(value float64) (int, *QuotaClamp) {
	return saturateQuota(math.Round(value), "QuotaRound")
}

// QuotaRoundStrict fails instead of returning a saturated pre-consume value.
func QuotaRoundStrict(value float64) (int, error) {
	return strictQuota(QuotaRoundChecked(value))
}

// QuotaFromDecimal rounds half-away-from-zero then saturates to int32.
func QuotaFromDecimal(d decimal.Decimal) int {
	quota, _ := QuotaFromDecimalChecked(d)
	return quota
}

func QuotaFromDecimalChecked(d decimal.Decimal) (int, *QuotaClamp) {
	f, _ := d.Round(0).Float64()
	return saturateQuota(f, "QuotaFromDecimal")
}
