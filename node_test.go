package hyperex

import "testing"

// TestNodeInterface tests that all AST types implement the Node interface.
// These tests exist purely for coverage of the empty node() marker methods.
func TestNodeInterface(t *testing.T) {
	tests := []struct {
		name string
		node Node
	}{
		{"intLit", &intLit{Value: 42}},
		{"floatLit", &floatLit{Value: 3.14}},
		{"stringLit", &stringLit{Value: "hello"}},
		{"boolLit", &boolLit{Value: true}},
		{"nilLit", &nilLit{}},
		{"ident", &ident{Name: "x"}},
		{"binaryExpr", &binaryExpr{Op: "+", Left: &intLit{Value: 1}, Right: &intLit{Value: 2}}},
		{"unaryExpr", &unaryExpr{Op: "-", Operand: &intLit{Value: 1}}},
		{"ternaryExpr", &ternaryExpr{Cond: &boolLit{Value: true}, Then: &intLit{Value: 1}, Else: &intLit{Value: 2}}},
		{"callExpr", &callExpr{Func: &ident{Name: "fn"}, Args: nil}},
		{"memberExpr", &memberExpr{Object: &ident{Name: "obj"}, Property: &ident{Name: "field"}}},
		{"indexExpr", &indexExpr{Object: &ident{Name: "arr"}, Index: &intLit{Value: 0}}},
		{"sliceExpr", &sliceExpr{Object: &ident{Name: "arr"}, Low: &intLit{Value: 0}, High: &intLit{Value: 5}}},
		{"arrayLit", &arrayLit{Elements: []Node{&intLit{Value: 1}}}},
		{"mapLit", &mapLit{Keys: []Node{&stringLit{Value: "a"}}, Values: []Node{&intLit{Value: 1}}}},
		{"rangeExpr", &rangeExpr{Start: &intLit{Value: 0}, End: &intLit{Value: 10}}},
		{"pipeExpr", &pipeExpr{Left: &ident{Name: "arr"}, Right: &ident{Name: "filter"}}},
		{"lambdaExpr", &lambdaExpr{Params: []string{"x"}, Body: &ident{Name: "x"}}},
		{"coalesceExpr", &coalesceExpr{Left: &ident{Name: "a"}, Right: &ident{Name: "b"}}},
		{"letExpr", &letExpr{Name: "x", Value: &intLit{Value: 1}, Body: &ident{Name: "x"}}},
		{"spreadExpr", &spreadExpr{Expr: &ident{Name: "arr"}}},
		{"DestructureExpr_object", &destructureExpr{Pattern: nil, IsArray: false, Value: &ident{Name: "obj"}, Body: &ident{Name: "a"}}},
		{"DestructureExpr_array", &destructureExpr{Pattern: nil, IsArray: true, Value: &ident{Name: "arr"}, Body: &ident{Name: "x"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call node() method to get coverage
			tt.node.node()

			// Verify String() returns non-empty
			if s := tt.node.String(); s == "" {
				t.Errorf("%s.String() returned empty string", tt.name)
			}
		})
	}
}

// TestNodeMethodsDirect directly calls node() methods for explicit coverage.
func TestNodeMethodsDirect(t *testing.T) {
	(&intLit{}).node()
	(&floatLit{}).node()
	(&stringLit{}).node()
	(&boolLit{}).node()
	(&nilLit{}).node()
	(&ident{}).node()
	(&binaryExpr{}).node()
	(&unaryExpr{}).node()
	(&ternaryExpr{}).node()
	(&callExpr{}).node()
	(&memberExpr{}).node()
	(&indexExpr{}).node()
	(&sliceExpr{}).node()
	(&arrayLit{}).node()
	(&mapLit{}).node()
	(&rangeExpr{}).node()
	(&pipeExpr{}).node()
	(&lambdaExpr{}).node()
	(&coalesceExpr{}).node()
	(&letExpr{}).node()
	(&spreadExpr{}).node()
	(&destructureExpr{}).node()
}
