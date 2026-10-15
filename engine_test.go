package hyperex

import (
	"context"
	"testing"
)

// =============================================================================
// ENGINE & OPTIONS TESTS
// =============================================================================

func TestEngine_CustomFunctions(t *testing.T) {
	engine := New(
		WithFunction("double", func(x int) int { return x * 2 }),
		WithFunction("greet", func(name string) string { return "Hello, " + name }),
		WithFunction("add", func(a, b int) int { return a + b }),
		WithFunction("sum3", func(a, b, c int) int { return a + b + c }),
	)

	tests := []struct {
		expr   string
		result any
	}{
		{"double(21)", int64(42)},
		{`greet("World")`, "Hello, World"},
		{"add(10, 20)", int64(30)},
		{"sum3(1, 2, 3)", int64(6)},
		{"double(double(5))", int64(20)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			got := NewValue(result).Interface()
			expected := NewValue(tt.result).Interface()
			if got != expected {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	}
}

func TestEngine_WithFunctions(t *testing.T) {
	engine := New(
		WithFunctions(map[string]any{
			"add": func(a, b int) int { return a + b },
			"mul": func(a, b int) int { return a * b },
		}),
	)

	result, err := engine.Eval(context.Background(), "add(2, 3) * mul(2, 3)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 30 { // (2+3) * (2*3) = 5 * 6 = 30
		t.Errorf("got %v, want 30", result)
	}
}

func TestEngine_WithOptimize(t *testing.T) {
	// Disabled optimization
	engine := New(WithOptimize(false))
	result, err := engine.Eval(context.Background(), "2 + 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(5) {
		t.Errorf("got %v, want 5", result)
	}
}

func TestEngine_WithConstantFolding(t *testing.T) {
	engine := New(WithConstantFolding(false))
	result, err := engine.Eval(context.Background(), "2 + 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(5) {
		t.Errorf("got %v, want 5", result)
	}
}

func TestEngine_WithStrengthReduction(t *testing.T) {
	engine := New(WithStrengthReduction(false))
	result, err := engine.Eval(context.Background(), "5 * 2", map[string]any{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(10) {
		t.Errorf("got %v, want 10", result)
	}
}

func TestEngine_WithMaxCallDepth(t *testing.T) {
	engine := New(WithMaxCallDepth(5))
	// Simple call should work
	result, err := engine.Eval(context.Background(), "len([1,2,3])")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(3) {
		t.Errorf("got %v, want 3", result)
	}
}

func TestEngine_AddFunction(t *testing.T) {
	engine := New()
	engine.AddFunction("triple", func(x int) int { return x * 3 })

	result, err := engine.Eval(context.Background(), "triple(7)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 21 {
		t.Errorf("got %v, want 21", result)
	}
}

func TestEngine_StructEnv(t *testing.T) {
	type Env struct {
		X    int     `expr:"x"`
		Y    float64 `expr:"y"`
		Name string
		skip int // unexported, should be skipped
	}

	engine := New()
	env := Env{X: 10, Y: 2.0, Name: "Bob", skip: 999}

	result, err := engine.Eval(context.Background(), "x + 5", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 15 {
		t.Errorf("got %v, want 15", result)
	}

	result, err = engine.Eval(context.Background(), `"Hi, " + Name`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "Hi, Bob" {
		t.Errorf("got %v, want Hi, Bob", result)
	}
}

func TestEngine_MapStringEnv(t *testing.T) {
	engine := New()
	env := map[string]int{"x": 10, "y": 20}

	result, err := engine.Eval(context.Background(), "x + y", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 30 {
		t.Errorf("got %v, want 30", result)
	}
}

func TestProgram_TypedEval(t *testing.T) {
	program := MustCompile("x + y")
	env := map[string]any{"x": int64(10), "y": int64(20)}

	// EvalInt
	i, err := program.EvalInt(context.Background(), env)
	if err != nil {
		t.Fatalf("EvalInt error: %v", err)
	}
	if i != 30 {
		t.Errorf("EvalInt: got %v, want 30", i)
	}

	// EvalFloat
	f, err := program.EvalFloat(context.Background(), env)
	if err != nil {
		t.Fatalf("EvalFloat error: %v", err)
	}
	if f != 30.0 {
		t.Errorf("EvalFloat: got %v, want 30.0", f)
	}

	// EvalString
	program2 := MustCompile(`"Hello"`)
	s, err := program2.EvalString(context.Background(), nil)
	if err != nil {
		t.Fatalf("EvalString error: %v", err)
	}
	if s != "Hello" {
		t.Errorf("EvalString: got %v, want Hello", s)
	}
}

func TestProgram_Source(t *testing.T) {
	program := MustCompile("x + y")
	if program.Source() != "x + y" {
		t.Errorf("got %v, want x + y", program.Source())
	}
}

func TestCompileOptions(t *testing.T) {
	tests := []struct {
		name string
		opt  CompileOption
	}{
		{"AsBool", AsBool()},
		{"AsInt", AsInt()},
		{"AsFloat", AsFloat()},
		{"AsString", AsString()},
		{"Unoptimized", Unoptimized()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile("1 + 1", tt.opt)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMustEval(t *testing.T) {
	result := MustEval(context.Background(), "2 + 2")
	if result != int64(4) {
		t.Errorf("got %v, want 4", result)
	}
}

func TestMustCompile_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	MustCompile("1 +")
}

func TestEngine_MustEval(t *testing.T) {
	eng := New()
	got := eng.MustEval(context.Background(), "3 * 7")
	if got != int64(21) {
		t.Fatalf("MustEval success: got %v want 21", got)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Engine.MustEval should panic on error")
		}
	}()
	_ = eng.MustEval(context.Background(), "1 / 0")
}

func TestEngine_MustCompile_Panic(t *testing.T) {
	eng := New()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Engine.MustCompile should panic on invalid expr")
		}
	}()
	_ = eng.MustCompile("1 +")
}

// =============================================================================
// ENGINE TYPED EVAL METHODS TESTS
// =============================================================================

func TestEngine_EvalBool(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected bool
	}{
		{"true", nil, true},
		{"false", nil, false},
		{"5 > 3", nil, true},
		{"5 < 3", nil, false},
		{"x > 10", map[string]any{"x": 15}, true},
		{"x > 10", map[string]any{"x": 5}, false},
		{`name == "test"`, map[string]any{"name": "test"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalBool(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_EvalInt(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected int64
	}{
		{"42", nil, 42},
		{"2 + 2 * 3", nil, 8},
		{"x + y", map[string]any{"x": 10, "y": 20}, 30},
		{"len([1, 2, 3])", nil, 3},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalInt(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_EvalFloat(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected float64
	}{
		{"3.14", nil, 3.14},
		{"10.0 / 4", nil, 2.5},
		{"price * discount", map[string]any{"price": 100.0, "discount": 0.9}, 90.0},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalFloat(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_EvalString(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected string
	}{
		{`"hello"`, nil, "hello"},
		{`"Hello, " + name`, map[string]any{"name": "World"}, "Hello, World"},
		{`upper("test")`, nil, "TEST"},
		{`lower("HELLO")`, nil, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalString(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// =============================================================================
// PACKAGE-LEVEL TYPED EVAL FUNCTIONS TESTS
// =============================================================================

func TestPackage_EvalBool(t *testing.T) {
	result, err := EvalBool(context.Background(), "5 > 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestPackage_EvalInt(t *testing.T) {
	result, err := EvalInt(context.Background(), "2 + 2 * 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 8 {
		t.Errorf("got %v, want 8", result)
	}
}

func TestPackage_EvalFloat(t *testing.T) {
	result, err := EvalFloat(context.Background(), "3.14 * 2")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 6.28 {
		t.Errorf("got %v, want 6.28", result)
	}
}

func TestPackage_EvalString(t *testing.T) {
	result, err := EvalString(context.Background(), `"Hello, " + "World"`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "Hello, World" {
		t.Errorf("got %q, want %q", result, "Hello, World")
	}
}
