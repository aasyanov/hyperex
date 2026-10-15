package hyperex

import (
	"context"
	"testing"
	"time"
)

// SCADA / PDM hot-path benches (Epic F). Prefer Program reuse + map[string]any.
// Run: go test -count=1 -run '^$' -bench 'BenchmarkSCADA_' -benchmem -benchtime=200ms .

func BenchmarkSCADA_SimpleArith_Compiled(b *testing.B) {
	eng := New(WithNumericMode(NumericFloat64))
	prog := eng.MustCompile(`(pv - sp) * gain + bias`)
	env := make(map[string]any, 4)
	env["pv"] = 100.0
	env["sp"] = 95.0
	env["gain"] = 1.5
	env["bias"] = 0.0
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, err := prog.EvalValue(ctx, env)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := v.TryFloat(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSCADA_Secure_Simple(b *testing.B) {
	// Timeout 0: measure Secure coerce + MaxOps path; wall-clock preset (25ms) flakes under GC STW.
	eng := NewSecureCalc(WithTimeout(0))
	prog := eng.MustCompile(`(pv - sp) * gain + bias`)
	env := make(map[string]any, 4)
	env["pv"] = 100.0
	env["sp"] = 95.0
	env["gain"] = 1.5
	env["bias"] = 0.0
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := prog.EvalFloat(ctx, env); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSCADA_TimeHour(b *testing.B) {
	eng := NewSecureCalc(WithTimeout(0))
	prog := eng.MustCompile(`timeHour(now)`)
	env := map[string]any{
		"now": time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := prog.EvalFloat(ctx, env); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSCADA_QualifiedMap(b *testing.B) {
	eng := NewSecureCalc(WithTimeout(0))
	prog := eng.MustCompile(`isGood(temp) ? v(temp) : v(backup)`)
	env := map[string]any{
		"temp": map[string]any{
			"v": 72.5,
			"q": float64(0), // Good
		},
		"backup": map[string]any{
			"v": 70.0,
			"q": float64(QualityClassUncertain),
		},
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, err := prog.EvalValue(ctx, env)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := v.TryFloat(); err != nil {
			b.Fatal(err)
		}
	}
}
