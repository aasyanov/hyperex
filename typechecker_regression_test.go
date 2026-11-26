package hyperex

import "testing"

func TestTypeCheckerAcceptsLanguageTruthiness(t *testing.T) {
	eng := New(WithTypeCheck(struct{}{}))
	for _, expr := range []string{`0 || 1`, `1 && 2`, `!1`, `1 ? 2 : 3`} {
		if _, err := eng.Compile(expr); err != nil {
			t.Fatalf("%s rejected: %v", expr, err)
		}
	}
}

func TestTypeCheckerMatchesRuntimeEnvTags(t *testing.T) {
	type hiddenEnv struct {
		Hidden int `expr:"-"`
	}
	tc := NewTypeChecker().WithStructEnv(hiddenEnv{Hidden: 1})
	if _, ok := tc.env["Hidden"]; ok {
		t.Fatal("expr:- field registered")
	}
	type jsonEnv struct {
		Name string `json:"name"`
	}
	tc = NewTypeChecker().WithStructEnv(jsonEnv{Name: "x"})
	if _, ok := tc.env["name"]; ok {
		t.Fatal("json tag unexpectedly became an environment alias")
	}
	if _, ok := tc.env["Name"]; !ok {
		t.Fatal("Go field name not registered")
	}
}

func TestTypeCheckerUsesEngineNumericMode(t *testing.T) {
	eng := New(WithTypeCheck(struct{}{}), WithNumericMode(NumericFloat64))
	node, err := newParser(`1 / 2`).parse()
	if err != nil {
		t.Fatal(err)
	}
	if got := eng.typeChecker.Infer(node); got != TypeFloat {
		t.Fatalf("got %s, want float", got)
	}
}

func TestTypeCheckerValidatesBothOperandsAndStringRepeat(t *testing.T) {
	eng := New(WithTypeCheck(struct{}{}))
	if _, err := eng.Compile(`1 + {}`); err == nil {
		t.Fatal("invalid right operand accepted")
	}
	if _, err := eng.Compile(`"a" * 2`); err != nil {
		t.Fatalf("valid string repetition rejected: %v", err)
	}
}

func TestTypeCheckerChecksVariadicArgumentsAndNumericWidening(t *testing.T) {
	tc := NewTypeChecker().WithFunc("f", FuncSig{
		Params: []Type{TypeArray, TypeFloat}, Variadic: true,
	})
	bad, err := newParser(`f([], 1, "x")`).parse()
	if err != nil {
		t.Fatal(err)
	}
	if errs := tc.Check(bad); errs == nil || errs.Len() == 0 {
		t.Fatal("variadic argument type mismatch accepted")
	}
	good, err := newParser(`f([], 1, 2)`).parse()
	if err != nil {
		t.Fatal(err)
	}
	if errs := tc.Check(good); errs != nil && errs.Len() > 0 {
		t.Fatalf("int to float widening rejected: %v", errs)
	}
}

func TestTypeCheckerSupportsTypedNilSchemaAndRestoresDestructureScope(t *testing.T) {
	type env struct {
		A int `expr:"a"`
		B int `expr:"b"`
	}
	var schema *env
	tc := NewTypeChecker().WithStructEnv(schema)
	if tc.env["a"] != TypeInt || tc.env["b"] != TypeInt {
		t.Fatalf("typed nil schema not registered: %#v", tc.env)
	}
	node, err := newParser(`let [a, c] = [1, 2]; a + c`).parse()
	if err != nil {
		t.Fatal(err)
	}
	if errs := tc.Check(node); errs != nil {
		t.Fatal(errs)
	}
	if tc.env["a"] != TypeInt || tc.env["b"] != TypeInt {
		t.Fatalf("outer environment not restored: %#v", tc.env)
	}
	if _, ok := tc.env["c"]; ok {
		t.Fatal("destructured variable leaked from scope")
	}
}
