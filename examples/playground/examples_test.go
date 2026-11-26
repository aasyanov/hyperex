package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestAllExamplesEvaluate(t *testing.T) {
	for _, category := range GetAllExamples() {
		for _, example := range category.Examples {
			t.Run(category.Name+"/"+example.Name, func(t *testing.T) {
				engine := buildEngine(EvalOptions{})
				program, err := engine.Compile(example.Expression)
				if err != nil {
					t.Fatalf("compile: %v", err)
				}
				_, err = program.Eval(context.Background(), example.Env)
				wantError := strings.HasPrefix(example.Expected, "Error:")
				if wantError && err == nil {
					t.Fatal("expected an evaluation error")
				}
				if !wantError && err != nil {
					t.Fatalf("eval: %v", err)
				}
			})
		}
	}
}

func TestRepeatAndFlattenDocumentedResults(t *testing.T) {
	for _, tc := range []struct {
		expression string
		expected   string
		want       any
	}{
		{
			expression: `repeat("ab", 3)`,
			expected:   `["ab", "ab", "ab"]`,
			want:       []any{"ab", "ab", "ab"},
		},
		{
			expression: `flatten([[1, 2], [3, [4, 5]]])`,
			expected:   "[1, 2, 3, 4, 5]",
			want:       []any{int64(1), int64(2), int64(3), int64(4), int64(5)},
		},
	} {
		program, err := buildEngine(EvalOptions{}).Compile(tc.expression)
		if err != nil {
			t.Fatal(err)
		}
		got, err := program.Eval(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %#v, want %#v", tc.expression, got, tc.want)
		}
		examples := GetAllExamples()
		found := false
		for _, category := range examples {
			for _, example := range category.Examples {
				if example.Expression == tc.expression {
					found = true
					if example.Expected != tc.expected {
						t.Fatalf("documented expected=%q, want %q", example.Expected, tc.expected)
					}
				}
			}
		}
		if !found {
			t.Fatalf("example %q not found", tc.expression)
		}
	}
}
