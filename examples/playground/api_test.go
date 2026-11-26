package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aasyanov/hyperex"
)

func TestAPIEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		body    any
		check   func(*testing.T, map[string]any)
	}{
		{
			name: "eval", handler: handleEval,
			body: EvalRequest{Expression: "1 + 2"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["result"] != float64(3) {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "analyze", handler: handleAnalyze,
			body: AnalyzeRequest{Expression: "sum([1,2])"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["complexity"] != "O(n)" {
					t.Fatalf("unexpected response: %#v", response)
				}
				for _, field := range []string{"isPure", "depth", "hasAssignment", "usesOptionalChaining", "usesNilCoalescing", "hasSideEffects"} {
					if _, ok := response[field]; !ok {
						t.Errorf("field %q missing from analysis response: %#v", field, response)
					}
				}
			},
		},
		{
			name: "profile", handler: handleProfile,
			body: ProfileRequest{Expression: "1 + 2"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["result"] != float64(3) {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "simplify", handler: handleSimplify,
			body: TransformRequest{Expression: "1 + 2"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["result"] != "3" {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "expand", handler: handleExpand,
			body: TransformRequest{Expression: "A", Macros: map[string]string{"A": "1 + 2"}},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["result"] != "1 + 2" {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "inline", handler: handleInline,
			body: TransformRequest{Expression: "let x = 1; x + x"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["result"] != "1 + 1" {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "validate", handler: handleValidate,
			body: ValidateRequest{Expression: "1 + 2"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["valid"] != true {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "extractVariables", handler: handleExtractVariables,
			body: ExtractRequest{Expression: "x + y"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || len(response["names"].([]any)) != 2 {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "extractFunctions", handler: handleExtractFunctions,
			body: ExtractRequest{Expression: "sum(x)"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || len(response["names"].([]any)) != 1 {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
		{
			name: "typecheck", handler: handleTypecheck,
			body: TypecheckRequest{Expression: "1 + 2"},
			check: func(t *testing.T, response map[string]any) {
				if response["success"] != true || response["valid"] != true {
					t.Fatalf("unexpected response: %#v", response)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/"+tc.name, bytes.NewReader(payload))
			recorder := httptest.NewRecorder()
			tc.handler(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var response map[string]any
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			tc.check(t, response)
		})
	}
}

func TestCatalogEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "examples", handler: handleExamples},
		{name: "functions", handler: handleFunctions},
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/"+tc.name, nil)
		recorder := httptest.NewRecorder()
		tc.handler(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status=%d", tc.name, recorder.Code)
		}
		var response []any
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if len(response) == 0 {
			t.Fatalf("%s returned an empty catalog", tc.name)
		}
	}
}

func TestBuiltinFunctionDocsComplete(t *testing.T) {
	documented := make(map[string]bool)
	for _, fn := range GetBuiltinFunctions() {
		documented[fn.Name] = true
	}
	for _, name := range hyperex.BuiltinNames() {
		if !documented[name] {
			t.Errorf("builtin %q is missing from playground docs", name)
		}
	}
}
