package hyperex

import (
	"context"
	"testing"
)

func TestUniqueUsesUnambiguousStructuralKeys(t *testing.T) {
	got, err := Eval(context.Background(), `unique([["a,s:b"], ["a","b"]])`)
	if err != nil {
		t.Fatal(err)
	}
	if len(NewValue(got).Array()) != 2 {
		t.Fatalf("distinct arrays collapsed: %#v", got)
	}
}

func TestUniqueDistinguishesMapKeyDelimiters(t *testing.T) {
	got, err := Eval(context.Background(), `unique([{"a:1": 2}, {"a": 1}])`)
	if err != nil {
		t.Fatal(err)
	}
	if len(NewValue(got).Array()) != 2 {
		t.Fatalf("distinct maps collapsed: %#v", got)
	}
}
