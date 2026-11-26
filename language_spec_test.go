package hyperex

import (
	"context"
	"math"
	"testing"
)

func TestLanguageGuideExamples(t *testing.T) {
	for _, tc := range []struct {
		name string
		expr string
		env  map[string]any
		want any
	}{
		{name: "precedence", expr: `2 + 2 * 3`, want: int64(8)},
		{name: "leading zero octal", expr: `012`, want: int64(10)},
		{name: "min int expression", expr: `-9223372036854775807 - 1`, want: int64(math.MinInt64)},
		{name: "float bits", expr: `0.0 == -0.0`, want: false},
		{name: "nested let", expr: `let x = 1; (let x = 2; x) + x`, want: int64(3)},
		{name: "implicit lambda", expr: `map([1, 2], x => x + 1)`, want: []any{int64(2), int64(3)}},
		{name: "hash lambda", expr: `map([1, 2], # => # + 1)`, want: []any{int64(2), int64(3)}},
		{name: "positional lambda", expr: `map([1, 2], $0 => $0 + 1)`, want: []any{int64(2), int64(3)}},
		{name: "pipe inserted argument", expr: `[1, 2, 3, 4] | take(2)`, want: []any{int64(1), int64(2)}},
		{name: "computed key", expr: `{[prefix + "name"]: "Alice"}`, env: map[string]any{"prefix": "user"}, want: map[string]any{"username": "Alice"}},
		{name: "reverse slice", expr: `[1, 2, 3][::-1]`, want: []any{int64(3), int64(2), int64(1)}},
		{name: "negative range", expr: `range(5, 1, -2)`, want: []any{int64(5), int64(3), int64(1)}},
		{name: "repeat", expr: `repeat("ab", 3)`, want: []any{"ab", "ab", "ab"}},
		{name: "flatten", expr: `flatten([[1], [2, [3]]])`, want: []any{int64(1), int64(2), int64(3)}},
		{name: "not in precedence", expr: `1 + 2 not in [3]`, want: false},
		{name: "coalesce precedence", expr: `2 + 3 ?? 4`, want: int64(5)},
		{name: "comparison chain alternative", expr: `1 < 2 && 2 < 3`, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Eval(context.Background(), tc.expr, tc.env)
			if err != nil {
				t.Fatal(err)
			}
			if !NewValue(got).Equal(NewValue(tc.want)) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestLanguageGuideNonLambdaHashIsRejected(t *testing.T) {
	if _, err := Eval(context.Background(), `map([1, 2], # + 1)`); err == nil {
		t.Fatal("bare # unexpectedly became a lambda")
	}
}
