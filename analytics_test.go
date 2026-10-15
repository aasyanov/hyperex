package hyperex

import (
	"context"
	"testing"
)

// =============================================================================
// ANALYTICS TESTS
// =============================================================================

func TestAnalyzeAST(t *testing.T) {
	expr := "x + y * len(arr)"
	fromStr, err := Analyze(expr)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	node, err := Parse(expr)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	fromAST := AnalyzeAST(node)
	if fromAST == nil {
		t.Fatal("AnalyzeAST returned nil")
	}
	if len(fromAST.Variables) != len(fromStr.Variables) {
		t.Fatalf("Variables len: AST=%d str=%d", len(fromAST.Variables), len(fromStr.Variables))
	}
	for i := range fromStr.Variables {
		if fromAST.Variables[i] != fromStr.Variables[i] {
			t.Fatalf("Variables[%d]: AST=%q str=%q", i, fromAST.Variables[i], fromStr.Variables[i])
		}
	}
	if len(fromAST.Functions) != len(fromStr.Functions) {
		t.Fatalf("Functions len: AST=%d str=%d", len(fromAST.Functions), len(fromStr.Functions))
	}
	for i := range fromStr.Functions {
		if fromAST.Functions[i] != fromStr.Functions[i] {
			t.Fatalf("Functions[%d]: AST=%q str=%q", i, fromAST.Functions[i], fromStr.Functions[i])
		}
	}
	if fromAST.Complexity != fromStr.Complexity {
		t.Fatalf("Complexity: AST=%v str=%v", fromAST.Complexity, fromStr.Complexity)
	}
	if fromAST.IsPure != fromStr.IsPure {
		t.Fatalf("IsPure: AST=%v str=%v", fromAST.IsPure, fromStr.IsPure)
	}
}

func TestAnalyze_Variables(t *testing.T) {
	analysis, err := Analyze("x + y * z")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(analysis.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d: %v", len(analysis.Variables), analysis.Variables)
	}

	// Variables should be sorted
	expected := []string{"x", "y", "z"}
	for i, v := range expected {
		if analysis.Variables[i] != v {
			t.Errorf("expected variable %s at position %d, got %s", v, i, analysis.Variables[i])
		}
	}
}

func TestAnalyze_Functions(t *testing.T) {
	analysis, err := Analyze("sum([1, 2, 3]) + len(arr)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(analysis.Functions) != 2 {
		t.Errorf("expected 2 functions, got %d: %v", len(analysis.Functions), analysis.Functions)
	}

	// Functions should be sorted
	expected := []string{"len", "sum"}
	for i, f := range expected {
		if analysis.Functions[i] != f {
			t.Errorf("expected function %s at position %d, got %s", f, i, analysis.Functions[i])
		}
	}
}

func TestAnalyze_Operators(t *testing.T) {
	analysis, err := Analyze("x + y * z - w / v")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(analysis.Operators) != 4 {
		t.Errorf("expected 4 operators, got %d: %v", len(analysis.Operators), analysis.Operators)
	}
}

func TestAnalyze_Complexity(t *testing.T) {
	tests := []struct {
		expr       string
		complexity Complexity
	}{
		{"1 + 2", ComplexityConstant},
		{"x + y", ComplexityConstant},
		{"sum([1, 2, 3])", ComplexityLinear},
		{"sort([3, 1, 2])", ComplexityLinearithmic}, // O(n log n) is correct for sorting
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			analysis, err := Analyze(tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if analysis.Complexity != tt.complexity {
				t.Errorf("expected complexity %s, got %s", tt.complexity, analysis.Complexity)
			}
		})
	}
}

func TestAnalyze_IsPure(t *testing.T) {
	tests := []struct {
		expr   string
		isPure bool
	}{
		{"1 + 2", true},
		{"x + y", true},
		{"sum([1, 2, 3])", true},
		{`print("hello")`, false}, // Side effect
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			analysis, err := Analyze(tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if analysis.IsPure != tt.isPure {
				t.Errorf("expected IsPure=%v, got %v", tt.isPure, analysis.IsPure)
			}
		})
	}
}

func TestAnalyze_HasAssignment(t *testing.T) {
	tests := []struct {
		expr          string
		hasAssignment bool
	}{
		{"1 + 2", false},
		{"let x = 5; x + 1", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			analysis, err := Analyze(tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if analysis.HasAssignment != tt.hasAssignment {
				t.Errorf("expected HasAssignment=%v, got %v", tt.hasAssignment, analysis.HasAssignment)
			}
		})
	}
}

func TestAnalyze_OptionalChaining(t *testing.T) {
	analysis, err := Analyze("obj?.field")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !analysis.UsesOptionalChaining {
		t.Error("expected UsesOptionalChaining=true")
	}
}

func TestAnalyze_NilCoalescing(t *testing.T) {
	analysis, err := Analyze("x ?? 0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !analysis.UsesNilCoalescing {
		t.Error("expected UsesNilCoalescing=true")
	}
}

func TestAnalyze_Depth(t *testing.T) {
	// Simple expression
	a1, _ := Analyze("1 + 2")
	// Nested expression
	a2, _ := Analyze("(1 + (2 + (3 + 4)))")

	if a1.Depth >= a2.Depth {
		t.Errorf("expected deeper nesting in second expr: %d vs %d", a1.Depth, a2.Depth)
	}
}

func TestAnalyze_NodeCount(t *testing.T) {
	a1, _ := Analyze("1")
	a2, _ := Analyze("1 + 2 + 3")

	if a1.NodeCount >= a2.NodeCount {
		t.Errorf("expected more nodes in second expr: %d vs %d", a1.NodeCount, a2.NodeCount)
	}
}

func TestProfileExpr(t *testing.T) {
	profile, result, err := ProfileExpr(context.Background(), "1 + 2 + 3", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if NewValue(result).Int() != 6 {
		t.Errorf("expected 6, got %v", result)
	}

	// Times can be 0 for very fast operations on fast machines
	// Just check they're non-negative
	if profile.TotalTime < 0 {
		t.Error("expected non-negative total time")
	}

	if profile.CompileTime < 0 {
		t.Error("expected non-negative compile time")
	}

	if profile.ExecuteTime < 0 {
		t.Error("expected non-negative exec time")
	}

	// Ops should always be positive
	if profile.OpsCount <= 0 {
		t.Error("expected positive ops count")
	}
}

func TestProfileExpr_MaxStackDepthTracksPeak(t *testing.T) {
	// Use env variables to avoid constant folding eliminating intermediate stack usage.
	profile, _, err := ProfileExpr(context.Background(), "a + b + c", map[string]any{
		"a": int64(1),
		"b": int64(2),
		"c": int64(3),
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// During evaluation there must be at least one moment where two operands are on stack.
	if profile.MaxStackDepth < 2 {
		t.Fatalf("expected MaxStackDepth >= 2, got %d", profile.MaxStackDepth)
	}
}

func TestProgram_Analyze(t *testing.T) {
	program := MustCompile("x + y * 2")
	analysis := program.Analyze()

	if len(analysis.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(analysis.Variables))
	}
}

func TestProgram_Profile(t *testing.T) {
	program := MustCompile("x + y")
	env := map[string]any{"x": int64(1), "y": int64(2)}

	profile, result, err := program.Profile(context.Background(), env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	if profile.OpsCount <= 0 {
		t.Error("expected positive ops count")
	}
}
