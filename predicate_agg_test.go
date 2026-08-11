package hyperex

import (
	"context"
	"errors"
	"testing"
)

func TestPredicateAgg_TruthinessFormsPreserved(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{`all([true, true, true])`, true},
		{`all([true, false])`, false},
		{`all([1, 2, 3])`, true},
		{`all([1, 0, 3])`, false},
		{`all([])`, true},

		{`any([false, false, true])`, true},
		{`any([false, 0, ""])`, false},
		{`any([])`, false},

		{`none([false, 0, ""])`, true},
		{`none([false, true])`, false},
		{`none([])`, true},

		{`one([false, true, false])`, true},
		{`one([true, true, false])`, false},
		{`one([false, false, false])`, false},
		{`one([])`, false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalBool(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("EvalBool(%q): %v", tt.expr, err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPredicateAgg_WithPredicate(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{`all([2, 4, 6], x => x % 2 == 0)`, true},
		{`all([2, 3, 6], x => x % 2 == 0)`, false},
		{`all([], x => false)`, true},
		{`all([10, 20], (x, i) => x > i)`, true},
		{`all([0, 1], (x, i) => x > i)`, false},

		{`any([1, 2, 3], x => x > 2)`, true},
		{`any([1, 2, 3], x => x > 10)`, false},
		{`any([], x => true)`, false},
		{`any([5, 6], (x, i) => i == 1)`, true},

		{`none([1, 2, 3], x => x > 10)`, true},
		{`none([1, 2, 3], x => x == 2)`, false},
		{`none([], x => true)`, true},

		{`one([1, 2, 3], x => x == 2)`, true},
		{`one([2, 2, 3], x => x == 2)`, false},
		{`one([1, 3, 5], x => x == 2)`, false},
		{`one([], x => true)`, false},
		{`one([10, 20, 30], (x, i) => i == 1)`, true},
		{`one([10, 20, 30], (x, i) => i >= 0)`, false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalBool(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("EvalBool(%q): %v", tt.expr, err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPredicateAgg_PipeExamples(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{`[1, 2, 3] | all(x => x > 0)`, true},
		{`[1, 0, 3] | all(x => x > 0)`, false},
		{`[1, 2, 3] | any(x => x == 2)`, true},
		{`[1, 2, 3] | none(x => x < 0)`, true},
		{`[1, 2, 3] | one(x => x == 2)`, true},
		{`[2, 4, 6] | all(x => x % 2 == 0)`, true},
		{`users | all(u => u.active)`, true},
		{`users | any(u => u.age >= 30)`, true},
		{`users | none(u => u.age < 18)`, true},
		{`users | one(u => u.role == "admin")`, true},
		// truthiness form still pipes
		{`[true, 1, "x"] | all`, true},
		{`[false, 0, ""] | none`, true},
		{`[false, true, false] | one`, true},
		{`[false, false] | any`, false},
	}
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "a", "active": true, "age": 25, "role": "user"},
			map[string]any{"name": "b", "active": true, "age": 32, "role": "admin"},
			map[string]any{"name": "c", "active": true, "age": 28, "role": "user"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalBool(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("EvalBool(%q): %v", tt.expr, err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPredicateAgg_BuiltinAndCustomPred(t *testing.T) {
	// Wrap 1-arg builtins — HO always passes (elem, index), same as filter/map.
	got, err := EvalBool(context.Background(), `all(["a", "b"], x => notEmpty(x))`)
	if err != nil {
		t.Fatalf("builtin pred: %v", err)
	}
	if !got {
		t.Fatalf("all with notEmpty: want true")
	}

	engine := New(WithFunction("gt5", func(x int) bool { return x > 5 }))
	got, err = engine.EvalBool(context.Background(), `any([1, 3, 9], x => gt5(x))`)
	if err != nil {
		t.Fatalf("custom pred: %v", err)
	}
	if !got {
		t.Fatalf("any with gt5: want true")
	}
	got, err = engine.EvalBool(context.Background(), `none([1, 2, 3], x => gt5(x))`)
	if err != nil {
		t.Fatalf("custom pred none: %v", err)
	}
	if !got {
		t.Fatalf("none with gt5: want true")
	}

	// 2-arg custom pred can consume index directly (like filter).
	engine2 := New(WithFunction("atIndex", func(x, i int) bool { return i == 1 && x > 0 }))
	got, err = engine2.EvalBool(context.Background(), `one([5, 7, 9], atIndex)`)
	if err != nil {
		t.Fatalf("2-arg custom: %v", err)
	}
	if !got {
		t.Fatalf("one with atIndex: want true")
	}
}

func TestPredicateAgg_Errors(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want error
	}{
		{"all_no_args", `all()`, ErrArgumentCount},
		{"all_three_args", `all([1], x => x, 1)`, ErrArgumentCount},
		{"any_three_args", `any([1], x => x, 1)`, ErrArgumentCount},
		{"none_three_args", `none([1], x => x, 1)`, ErrArgumentCount},
		{"one_three_args", `one([1], x => x, 1)`, ErrArgumentCount},

		{"all_not_array", `all(42)`, ErrInvalidType},
		{"any_not_array", `any("x")`, ErrInvalidType},
		{"none_not_array", `none(1)`, ErrInvalidType},
		{"one_not_array", `one(true)`, ErrInvalidType},

		{"all_pred_not_func", `all([1, 2], 42)`, ErrInvalidType},
		{"any_pred_not_func", `any([1], "x")`, ErrInvalidType},
		{"none_pred_not_func", `none([1], true)`, ErrInvalidType},
		{"one_pred_not_func", `one([1], [])`, ErrInvalidType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.expr)
			if err == nil {
				t.Fatalf("expected error %v", tt.want)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("errors.Is(%v, %v) = false; err=%v", err, tt.want, err)
			}
		})
	}
}

func TestPredicateAgg_TypeCheckArity(t *testing.T) {
	engine := New(WithTypeCheck(struct{}{}))
	if _, err := engine.Compile(`all([1, 2], x => x > 0)`); err != nil {
		t.Fatalf("2-arg form should typecheck: %v", err)
	}
	if _, err := engine.Compile(`all([1, 2])`); err != nil {
		t.Fatalf("1-arg form should typecheck: %v", err)
	}
	if _, err := engine.Compile(`all()`); err == nil {
		t.Fatal("all() should fail typecheck")
	}
	if _, err := engine.Compile(`any([1], x => x, 1)`); err == nil {
		t.Fatal("3-arg any should fail typecheck")
	}
}

func TestPredicateAgg_ShortCircuit(t *testing.T) {
	got, err := EvalBool(context.Background(), `all([1, -1, 2], x => x > 0)`)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("all should be false")
	}
	got, err = EvalBool(context.Background(), `any([-1, 2, 3], x => x > 0)`)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("any should be true")
	}
	got, err = EvalBool(context.Background(), `one([1, 2, 3], x => x == 2)`)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("one should be true")
	}
	// one stops after finding a second match
	got, err = EvalBool(context.Background(), `one([2, 2, 3], x => x == 2)`)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("one with two matches should be false")
	}
}

func TestPredicateAgg_PredErrorPropagates(t *testing.T) {
	_, err := Eval(context.Background(), `all([1, 0], x => 1 / x)`)
	if err == nil {
		t.Fatal("expected division error from pred")
	}
}
