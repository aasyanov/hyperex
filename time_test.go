package hyperex

import (
	"context"
	"errors"
	"math"
	"sort"
	"testing"
	"time"
)

func TestSecure_EnvTimeHour(t *testing.T) {
	eng := New(WithSecureSandbox())
	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	result, err := eng.Eval(context.Background(), `timeHour(now)`, map[string]any{"now": ts})
	if err != nil {
		t.Fatalf("Secure + time.Time env: %v", err)
	}
	if NewValue(result).Int() != 14 {
		t.Fatalf("timeHour: got %v, want 14", result)
	}
}

func TestSecure_StillRejectsArbitraryStruct(t *testing.T) {
	type Env struct {
		Secret string
	}
	eng := New(WithSecureSandbox())
	_, err := eng.Eval(context.Background(), `x`, map[string]any{"x": &Env{Secret: "nope"}})
	if err == nil {
		t.Fatal("expected Secure to reject arbitrary struct pointer")
	}
	var sandboxErr *SandboxError
	if !errors.As(err, &sandboxErr) || sandboxErr.Type != "access_denied" {
		t.Fatalf("expected access_denied SandboxError, got %v", err)
	}
}

func TestTimeDuration_Arithmetic(t *testing.T) {
	eng := New()
	ctx := context.Background()
	env := map[string]any{
		"t": time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		"d": 2 * time.Hour,
	}

	got, err := eng.Eval(ctx, `t + d`, env)
	if err != nil {
		t.Fatalf("t+d: %v", err)
	}
	want := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
	if !NewValue(got).Equal(Time(want)) {
		t.Fatalf("t+d: got %v, want %v", got, want)
	}

	got, err = eng.Eval(ctx, `d + t`, env)
	if err != nil {
		t.Fatalf("d+t: %v", err)
	}
	if !NewValue(got).Equal(Time(want)) {
		t.Fatalf("d+t: got %v, want %v", got, want)
	}

	got, err = eng.Eval(ctx, `t - d`, env)
	if err != nil {
		t.Fatalf("t-d: %v", err)
	}
	wantSub := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if !NewValue(got).Equal(Time(wantSub)) {
		t.Fatalf("t-d: got %v, want %v", got, wantSub)
	}

	got, err = eng.Eval(ctx, `t - t`, env)
	if err != nil {
		t.Fatalf("t-t: %v", err)
	}
	if !NewValue(got).Equal(Duration(0)) {
		t.Fatalf("t-t: got %v, want 0 duration", got)
	}

	got, err = eng.Eval(ctx, `(t + d) - t`, env)
	if err != nil {
		t.Fatalf("(t+d)-t: %v", err)
	}
	if !NewValue(got).Equal(Duration(2 * time.Hour)) {
		t.Fatalf("(t+d)-t: got %v, want 2h", got)
	}

	_, err = eng.Eval(ctx, `t + t`, env)
	if err == nil {
		t.Fatal("expected t+t error")
	}

	_, err = eng.Eval(ctx, `"hi" + t`, env)
	if err == nil {
		t.Fatal("expected string+time error")
	}
}

func TestDuration_ScaleOps(t *testing.T) {
	eng := New()
	ctx := context.Background()
	env := map[string]any{"d": time.Second}

	got, err := eng.Eval(ctx, `d * 3`, env)
	if err != nil {
		t.Fatalf("d*3: %v", err)
	}
	if !NewValue(got).Equal(Duration(3 * time.Second)) {
		t.Fatalf("d*3: got %v", got)
	}

	got, err = eng.Eval(ctx, `d / 2`, env)
	if err != nil {
		t.Fatalf("d/2: %v", err)
	}
	if !NewValue(got).Equal(Duration(time.Second / 2)) {
		t.Fatalf("d/2: got %v", got)
	}

	got, err = eng.Eval(ctx, `d + d`, env)
	if err != nil {
		t.Fatalf("d+d: %v", err)
	}
	if !NewValue(got).Equal(Duration(2 * time.Second)) {
		t.Fatalf("d+d: got %v", got)
	}
}

func TestBuiltinNames_IsBuiltin(t *testing.T) {
	names := BuiltinNames()
	if len(names) == 0 {
		t.Fatal("BuiltinNames empty")
	}
	if !sort.StringsAreSorted(names) {
		t.Fatal("BuiltinNames not sorted")
	}
	if !IsBuiltin("len") {
		t.Fatal(`IsBuiltin("len") = false`)
	}
	if !IsBuiltin(nameTimeHour) {
		t.Fatal("IsBuiltin(timeHour) = false")
	}
	if IsBuiltin("not_a_real_builtin_xyz") {
		t.Fatal("unexpected IsBuiltin hit")
	}
	// Copy independence
	names[0] = "mutated"
	if BuiltinNames()[0] == "mutated" {
		t.Fatal("BuiltinNames must return a copy")
	}
}

func TestPresetSecureAuto_DeniesPrint(t *testing.T) {
	eng := New(PresetSecureAuto()...)
	_, err := eng.Eval(context.Background(), `print("x")`)
	if err == nil {
		t.Fatal("expected print denied")
	}
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Fatalf("expected ErrFunctionNotAllowed, got %v", err)
	}
}

func TestPresetSecureCalc_AllowsTimeHour(t *testing.T) {
	eng := NewSecureCalc()
	ts := time.Date(2024, 6, 15, 9, 0, 0, 0, time.UTC)
	got, err := eng.Eval(context.Background(), `timeHour(now)`, map[string]any{"now": ts})
	if err != nil {
		t.Fatalf("NewSecureCalc timeHour: %v", err)
	}
	if NewValue(got).Int() != 9 {
		t.Fatalf("got %v, want 9", got)
	}
}

func TestNewValue_TimeDuration(t *testing.T) {
	ts := time.Date(2020, 1, 2, 3, 4, 5, 6, time.FixedZone("X", 3600))
	v := NewValue(ts)
	if !v.IsTime() {
		t.Fatalf("kind: %v", v.Type())
	}
	iface, ok := v.Interface().(time.Time)
	if !ok {
		t.Fatalf("Interface type %T", v.Interface())
	}
	if iface.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", iface.Location())
	}
	if iface.UnixNano() != ts.UTC().UnixNano() {
		t.Fatalf("instant mismatch")
	}

	d := NewValue(90 * time.Millisecond)
	if !d.IsDuration() {
		t.Fatalf("kind: %v", d.Type())
	}
	if d.Interface().(time.Duration) != 90*time.Millisecond {
		t.Fatalf("duration mismatch")
	}
}

func TestTimeBuiltins_ConstructExtract(t *testing.T) {
	eng := New()
	ctx := context.Background()

	got, err := eng.Eval(ctx, `timeHour(timeUTC(2024, 6, 15, 14, 30, 0))`)
	if err != nil {
		t.Fatalf("timeUTC/timeHour: %v", err)
	}
	if NewValue(got).Int() != 14 {
		t.Fatalf("got %v", got)
	}

	got, err = eng.Eval(ctx, `durSeconds(durationSeconds(2.5))`)
	if err != nil {
		t.Fatalf("durationSeconds: %v", err)
	}
	if NewValue(got).Float() != 2.5 {
		t.Fatalf("got %v", got)
	}

	got, err = eng.Eval(ctx, `timeWeekday(timeUnix(0))`)
	if err != nil {
		t.Fatalf("timeWeekday: %v", err)
	}
	// 1970-01-01 was Thursday -> Go weekday 4
	if NewValue(got).Int() != 4 {
		t.Fatalf("weekday got %v, want 4", got)
	}
}

func TestConcurrency_SharedProgramEval_TimeEnv(t *testing.T) {
	eng := NewSecureCalc()
	prog, err := eng.Compile(`timeHour(now) + timeMinute(now)`)
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Date(2024, 1, 1, 10, 20, 0, 0, time.UTC)
	env := map[string]any{"now": ts}

	const n = 32
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			v, err := prog.Eval(context.Background(), env)
			if err != nil {
				errs <- err
				return
			}
			if NewValue(v).Int() != 30 {
				errs <- errors.New("unexpected result")
				return
			}
			errs <- nil
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> time_test.go
// ---------------------------------------------------------------------------

func TestModeB_TimeBuiltins_FullCatalog(t *testing.T) {
	eng := New()
	ctx := context.Background()
	ts := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	env := map[string]any{"t": ts, "d": 1500 * time.Millisecond}

	cases := []struct {
		expr  string
		want  int64
		fwant float64
		useF  bool
	}{
		{`timeYear(t)`, 2024, 0, false},
		{`timeMonth(t)`, 6, 0, false},
		{`timeDay(t)`, 15, 0, false},
		{`timeHour(t)`, 14, 0, false},
		{`timeMinute(t)`, 30, 0, false},
		{`timeSecond(t)`, 45, 0, false},
		{`timeWeekday(t)`, 6, 0, false}, // Saturday
		{`unix(t)`, ts.Unix(), 0, false},
		{`unixNano(t)`, ts.UnixNano(), 0, false},
		{`timeHour(timeUnixNano(unixNano(t)))`, 14, 0, false},
		{`timeHour(timeUnix(unix(t)))`, 14, 0, false},
		{`durMs(d)`, 1500, 0, false},
		{`durNs(d)`, int64(1500 * time.Millisecond), 0, false},
		{`durSeconds(d)`, 0, 1.5, true},
		{`durNs(durationNs(42))`, 42, 0, false},
		{`durMs(durationMs(3))`, 3, 0, false},
		{`timeHour(timeAdd(t, durationSeconds(3600)))`, 15, 0, false},
	}
	for _, tt := range cases {
		got, err := eng.Eval(ctx, tt.expr, env)
		if err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		v := NewValue(got)
		if tt.useF {
			if math.Abs(v.Float()-tt.fwant) > 1e-9 {
				t.Fatalf("%s: got %v want %v", tt.expr, got, tt.fwant)
			}
			continue
		}
		if v.Int() != tt.want {
			t.Fatalf("%s: got %v want %d", tt.expr, got, tt.want)
		}
	}

	// Error paths: wrong arity / wrong types / non-integer floats.
	errCases := []string{
		`timeUTC(2024)`,
		`timeUTC("y", 1, 1, 0, 0, 0)`,
		`timeUnix()`,
		`timeUnix(1.5)`,
		`timeUnixNano()`,
		`timeUnixNano(1.5)`,
		`unix(1)`,
		`unixNano("t")`,
		`timeYear(1)`,
		`timeMonth()`,
		`timeDay(d)`,
		`timeSecond()`,
		`timeAdd(t)`,
		`timeAdd(t, t)`,
		`timeAdd(d, d)`,
		`durationSeconds()`,
		`durationSeconds("x")`,
		`durationMs()`,
		`durationNs()`,
		`durationNs(1.5)`,
		`durSeconds(1)`,
		`durMs()`,
		`durNs(t)`,
	}
	for _, expr := range errCases {
		if _, err := eng.Eval(ctx, expr, env); err == nil {
			t.Fatalf("%s: expected error", expr)
		}
	}
}

func TestModeB_TimeDuration_OverflowPaths(t *testing.T) {
	eng := New(WithOptimize(false))
	ctx := context.Background()
	env := map[string]any{
		"dmax": time.Duration(math.MaxInt64),
		"dmin": time.Duration(math.MinInt64),
		"tmax": time.Unix(0, math.MaxInt64).UTC(),
		"tmin": time.Unix(0, math.MinInt64).UTC(),
		"d1":   time.Duration(1),
	}

	overflow := []string{
		`dmax + dmax`,
		`dmin + dmin`,
		`dmax - dmin`,
		`dmin - dmax`,
		`dmax * 2`,
		`2 * dmax`,
		`dmin * -1`,
		`dmax * 1.5`,
		`dmax / 0.5`,
		`-dmin`,
		`tmax + dmax`,
		`tmax + d1`,
		`tmin - dmax`,
		`tmax - tmin`,
	}
	for _, expr := range overflow {
		_, err := eng.Eval(ctx, expr, env)
		if err == nil {
			t.Fatalf("%s: expected overflow/error", expr)
		}
	}

	_, err := eng.Eval(ctx, `dmax / 0`, env)
	if !errors.Is(err, ErrDivisionByZero) {
		t.Fatalf("dmax/0: want ErrDivisionByZero, got %v", err)
	}
	_, err = eng.Eval(ctx, `-tmax`, env)
	if err == nil {
		t.Fatal("expected cannot negate time")
	}

	// scaleToDuration: NaN/Inf / out-of-range product
	nanEnv := map[string]any{"nan": math.NaN(), "inf": math.Inf(1), "huge": 1e300}
	for _, expr := range []string{
		`durationSeconds(nan)`,
		`durationSeconds(inf)`,
		`durationSeconds(huge)`,
		`durationMs(huge)`,
		`durationMs(nan)`,
	} {
		if _, e := eng.Eval(ctx, expr, nanEnv); e == nil {
			t.Fatalf("%s: expected error", expr)
		}
	}

	// Happy path still works near edges that fit.
	got, err := eng.Eval(ctx, `durNs(durationNs(0))`)
	if err != nil || NewValue(got).Int() != 0 {
		t.Fatalf("durationNs(0): %v %v", got, err)
	}
}
