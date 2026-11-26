package main

// Example represents a single expression example.
type Example struct {
	Name        string         `json:"name"`
	Expression  string         `json:"expression"`
	Description string         `json:"description"`
	Expected    string         `json:"expected"`
	Env         map[string]any `json:"env,omitempty"`
}

// Category represents a group of examples.
type Category struct {
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Description string    `json:"description"`
	Examples    []Example `json:"examples"`
}

// GetAllExamples returns all example categories.
func GetAllExamples() []Category {
	return []Category{
		// ============================================================
		// LITERALS
		// ============================================================
		{
			Name:        "Literals",
			Icon:        "📦",
			Description: "Basic value literals: numbers, strings, booleans, nil",
			Examples: []Example{
				{Name: "Integer", Expression: "42", Description: "Integer literal", Expected: "42"},
				{Name: "Negative", Expression: "-17", Description: "Negative integer", Expected: "-17"},
				{Name: "Float", Expression: litPi, Description: "Float literal", Expected: litPi},
				{Name: "Float shorthand", Expression: ".5", Description: "Float without leading zero", Expected: "0.5"},
				{Name: "Scientific", Expression: "1e10", Description: "Scientific notation", Expected: "1e+10"},
				{Name: "Scientific negative", Expression: "1.5e-3", Description: "Negative exponent", Expected: "0.0015"},
				{Name: "Hexadecimal", Expression: "0xFF", Description: "Hex literal", Expected: "255"},
				{Name: "Binary", Expression: "0b1010", Description: "Binary literal", Expected: "10"},
				{Name: "Octal", Expression: "0o777", Description: "Octal literal", Expected: "511"},
				{Name: "Large int", Expression: "1000000", Description: "Million", Expected: "1000000"},
				{Name: "String double", Expression: `"hello world"`, Description: "Double-quoted string", Expected: `"hello world"`},
				{Name: "String single", Expression: `'single quotes'`, Description: "Single-quoted string", Expected: `"single quotes"`},
				{Name: "Raw string", Expression: "`raw\\nstring`", Description: "Raw string (no escapes)", Expected: `"raw\\nstring"`},
				{Name: "Escape sequences", Expression: `"line1\nline2"`, Description: "String with newline", Expected: `"line1\nline2"`},
				{Name: "Boolean true", Expression: litTrue, Description: "Boolean true", Expected: litTrue},
				{Name: "Boolean false", Expression: litFalse, Description: "Boolean false", Expected: litFalse},
				{Name: "Nil", Expression: litNil, Description: "Nil value", Expected: litNil},
			},
		},

		// ============================================================
		// ARITHMETIC
		// ============================================================
		{
			Name:        "Arithmetic",
			Icon:        "➕",
			Description: "Arithmetic operators: +, -, *, /, %, **",
			Examples: []Example{
				{Name: "Addition", Expression: "2 + 3", Description: "Integer addition", Expected: "5"},
				{Name: "Subtraction", Expression: "10 - 4", Description: "Integer subtraction", Expected: "6"},
				{Name: "Multiplication", Expression: "6 * 7", Description: "Integer multiplication", Expected: "42"},
				{Name: "Division int", Expression: "10 / 3", Description: "Integer division (truncates)", Expected: "3"},
				{Name: "Division float", Expression: "10.0 / 3", Description: "Float division", Expected: "3.333..."},
				{Name: "Modulo", Expression: "17 % 5", Description: "Remainder", Expected: "2"},
				{Name: descPower, Expression: "2 ** 10", Description: "Exponentiation", Expected: "1024"},
				{Name: "Negative power", Expression: "2 ** -1", Description: "Negative exponent", Expected: "0.5"},
				{Name: "Unary minus", Expression: "-(5 + 3)", Description: "Unary negation", Expected: "-8"},
				{Name: "Precedence", Expression: "2 + 3 * 4", Description: "Multiplication first", Expected: "14"},
				{Name: "Parentheses", Expression: "(2 + 3) * 4", Description: "Override precedence", Expected: "20"},
				{Name: descComplex, Expression: "(10 + 5) * 2 - 8 / 4", Description: "Complex expression", Expected: "28"},
				{Name: "Float precision", Expression: "0.1 + 0.2", Description: "Float arithmetic", Expected: "0.3"},
				{Name: "Mixed types", Expression: "5 + 2.5", Description: "Int + float = float", Expected: "7.5"},
				{Name: "String concat", Expression: `"Hello, " + "World!"`, Description: "String concatenation", Expected: `"Hello, World!"`},
				{Name: "Array concat", Expression: "[1, 2] + [3, 4]", Description: "Array concatenation", Expected: litArr1234},
			},
		},

		// ============================================================
		// COMPARISON
		// ============================================================
		{
			Name:        "Comparison",
			Icon:        "⚖️",
			Description: "Comparison operators: ==, !=, <, <=, >, >=",
			Examples: []Example{
				{Name: "Equal int", Expression: "5 == 5", Description: "Integer equality", Expected: litTrue},
				{Name: "Equal mixed", Expression: "1 == 1.0", Description: "Int equals float", Expected: litTrue},
				{Name: "Not equal", Expression: "5 != 3", Description: "Not equal", Expected: litTrue},
				{Name: "Less than", Expression: "3 < 5", Description: "Less than", Expected: litTrue},
				{Name: "Less or equal", Expression: "5 <= 5", Description: "Less or equal", Expected: litTrue},
				{Name: "Greater than", Expression: "7 > 3", Description: "Greater than", Expected: litTrue},
				{Name: "Greater or equal", Expression: "5 >= 5", Description: "Greater or equal", Expected: litTrue},
				{Name: "String compare", Expression: `"apple" < "banana"`, Description: "Lexicographic comparison", Expected: litTrue},
				{Name: "Array equality", Expression: "[1, 2] == [1, 2]", Description: "Deep array equality", Expected: litTrue},
				{Name: "Array not equal", Expression: "[1, 2] != [1, 3]", Description: "Arrays differ", Expected: litTrue},
				{Name: "Map equality", Expression: `{"a": 1} == {"a": 1}`, Description: "Deep map equality", Expected: litTrue},
				{Name: "Nil comparison", Expression: "nil == nil", Description: "nil self-equality", Expected: litTrue},
				{Name: "Chain comparison", Expression: "1 < 2 && 2 < 3", Description: "Chained comparisons", Expected: litTrue},
			},
		},

		// ============================================================
		// LOGICAL
		// ============================================================
		{
			Name:        "Logical",
			Icon:        "🔀",
			Description: "Logical operators: &&, ||, !, and, or, not",
			Examples: []Example{
				{Name: "AND true", Expression: "true && true", Description: "Logical AND (both true)", Expected: litTrue},
				{Name: "AND false", Expression: "true && false", Description: "Logical AND (one false)", Expected: litFalse},
				{Name: "OR true", Expression: "false || true", Description: "Logical OR", Expected: litTrue},
				{Name: "OR false", Expression: "false || false", Description: "Logical OR (both false)", Expected: litFalse},
				{Name: "NOT", Expression: "!false", Description: "Logical NOT", Expected: litTrue},
				{Name: "Double NOT", Expression: "!!1", Description: "Double negation (to bool)", Expected: litTrue},
				{Name: "Keyword and", Expression: "true and false", Description: "Using 'and' keyword", Expected: litFalse},
				{Name: "Keyword or", Expression: "false or true", Description: "Using 'or' keyword", Expected: litTrue},
				{Name: "Keyword not", Expression: "not true", Description: "Using 'not' keyword", Expected: litFalse},
				{Name: descComplex, Expression: "(a > 0 && b > 0) || c", Description: "Complex logical", Expected: litTrue, Env: map[string]any{"a": 5, "b": 3, "c": false}},
				{Name: "Short-circuit AND", Expression: "false && error()", Description: "AND short-circuits on false", Expected: litFalse},
				{Name: "Short-circuit OR", Expression: "true || error()", Description: "OR short-circuits on true", Expected: litTrue},
			},
		},

		// ============================================================
		// BITWISE
		// ============================================================
		{
			Name:        "Bitwise",
			Icon:        "🔢",
			Description: "Bitwise operators: &, |, ^, ~, <<, >>, >>>",
			Examples: []Example{
				{Name: "AND", Expression: "5 & 3", Description: "Bitwise AND (0101 & 0011)", Expected: "1"},
				{Name: "OR", Expression: "5 | 3", Description: "Bitwise OR (0101 | 0011)", Expected: "7"},
				{Name: "XOR", Expression: "5 ^ 3", Description: "Bitwise XOR (0101 ^ 0011)", Expected: "6"},
				{Name: "NOT", Expression: "~5", Description: "Bitwise NOT", Expected: "-6"},
				{Name: "Left shift", Expression: "1 << 4", Description: "Left shift", Expected: "16"},
				{Name: "Right shift", Expression: "16 >> 2", Description: "Right shift (signed)", Expected: "4"},
				{Name: "Unsigned right", Expression: "-1 >>> 63", Description: "Unsigned right shift", Expected: "1"},
				{Name: "Flags check", Expression: "(flags & 0x04) != 0", Description: "Check flag bit", Expected: litTrue, Env: map[string]any{exFlags: 7}},
				{Name: "Set bit", Expression: "flags | 0x08", Description: "Set a bit", Expected: "15", Env: map[string]any{exFlags: 7}},
				{Name: "Clear bit", Expression: "flags & ~0x02", Description: "Clear a bit", Expected: "5", Env: map[string]any{exFlags: 7}},
				{Name: "Toggle bit", Expression: "flags ^ 0x01", Description: "Toggle a bit", Expected: "6", Env: map[string]any{exFlags: 7}},
				{Name: "Power of 2", Expression: "1 << 10", Description: "2^10 via shift", Expected: "1024"},
			},
		},

		// ============================================================
		// TERNARY & COALESCING
		// ============================================================
		{
			Name:        "Ternary & Coalescing",
			Icon:        "❓",
			Description: "Conditional operators: ?:, ??",
			Examples: []Example{
				{Name: "Ternary true", Expression: "true ? 1 : 2", Description: "Ternary (condition true)", Expected: "1"},
				{Name: "Ternary false", Expression: "false ? 1 : 2", Description: "Ternary (condition false)", Expected: "2"},
				{Name: "Ternary expr", Expression: "x > 10 ? \"big\" : \"small\"", Description: "Ternary with expression", Expected: `"big"`, Env: map[string]any{"x": 15}},
				{Name: "Nested ternary", Expression: "x > 90 ? \"A\" : x > 80 ? \"B\" : \"C\"", Description: "Nested ternary (grade)", Expected: `"B"`, Env: map[string]any{"x": 85}},
				{Name: "Coalesce nil", Expression: "nil ?? 42", Description: "Nil-coalescing (nil)", Expected: "42"},
				{Name: "Coalesce value", Expression: "5 ?? 42", Description: "Nil-coalescing (has value)", Expected: "5"},
				{Name: "Coalesce zero", Expression: "0 ?? 42", Description: "0 is not nil!", Expected: "0"},
				{Name: "Coalesce empty", Expression: `"" ?? "default"`, Description: "Empty string is not nil!", Expected: `""`},
				{Name: "Coalesce chain", Expression: "a ?? b ?? c ?? 100", Description: "Chained coalescing", Expected: "100", Env: map[string]any{"a": nil, "b": nil, "c": nil}},
				{Name: "OR truthiness", Expression: "0 || 42", Description: "|| uses truthiness (0 is falsy)", Expected: "42"},
			},
		},

		// ============================================================
		// OPTIONAL CHAINING
		// ============================================================
		{
			Name:        "Optional Chaining",
			Icon:        "🔗",
			Description: "Safe navigation: ?., ?.[]",
			Examples: []Example{
				{Name: "Property exists", Expression: "user?.name", Description: "Access existing property", Expected: litQuotedAlice, Env: map[string]any{exUser: map[string]any{exName: litAlice}}},
				{Name: "Property nil", Expression: "user?.name", Description: "Access on nil (returns nil)", Expected: litNil, Env: map[string]any{exUser: nil}},
				{Name: "Nested chain", Expression: "user?.address?.city", Description: "Deep optional chain", Expected: `"NYC"`, Env: map[string]any{exUser: map[string]any{exAddress: map[string]any{"city": "NYC"}}}},
				{Name: "Nested nil", Expression: "user?.address?.city", Description: "Chain stops at nil", Expected: litNil, Env: map[string]any{exUser: map[string]any{exAddress: nil}}},
				{Name: "Array access", Expression: "arr?.[0]", Description: "Optional array access", Expected: "1", Env: map[string]any{exArr: []any{1, 2, 3}}},
				{Name: "Array nil", Expression: "arr?.[0]", Description: "Array access on nil", Expected: litNil, Env: map[string]any{exArr: nil}},
				{Name: "Mixed chain", Expression: "data?.items?.[0]?.name", Description: "Complex chain", Expected: `"Item1"`, Env: map[string]any{exData: map[string]any{exItems: []any{map[string]any{exName: "Item1"}}}}},
				{Name: "With default", Expression: "user?.name ?? \"Anonymous\"", Description: "Optional with default", Expected: `"Anonymous"`, Env: map[string]any{exUser: nil}},
			},
		},

		// ============================================================
		// IN OPERATOR
		// ============================================================
		{
			Name:        "In Operator",
			Icon:        "🔍",
			Description: "Membership testing: in, not in",
			Examples: []Example{
				{Name: "In array", Expression: "5 in [1, 3, 5, 7]", Description: "Element in array", Expected: litTrue},
				{Name: "Not in array", Expression: "4 in [1, 3, 5, 7]", Description: "Element not in array", Expected: litFalse},
				{Name: "In string", Expression: `"ell" in "hello"`, Description: "Substring in string", Expected: litTrue},
				{Name: "In map", Expression: `"name" in user`, Description: "Key in map", Expected: litTrue, Env: map[string]any{exUser: map[string]any{exName: litAlice, exAge: 30}}},
				{Name: "Not in keyword", Expression: `"x" not in ["a", "b"]`, Description: "Using 'not in'", Expected: litTrue},
				{Name: "Key not in map", Expression: `"email" not in user`, Description: "Key not in map", Expected: litTrue, Env: map[string]any{exUser: map[string]any{exName: litAlice}}},
				{Name: "Deep equality", Expression: `{"a":1} in [{"a":1}, {"b":2}]`, Description: "Object in array (deep)", Expected: litTrue},
			},
		},

		// ============================================================
		// RANGE
		// ============================================================
		{
			Name:        "Range",
			Icon:        "📏",
			Description: "Range operator and function",
			Examples: []Example{
				{Name: "Simple range", Expression: "1..5", Description: "Range 1 to 5 inclusive", Expected: litArr12345},
				{Name: "Zero start", Expression: "0..3", Description: "Range from 0", Expected: "[0, 1, 2, 3]"},
				{Name: "Range function", Expression: "range(0, 10, 2)", Description: "Range with step", Expected: "[0, 2, 4, 6, 8, 10]"},
				{Name: "Reverse range", Expression: "range(10, 0, -1)", Description: "Descending range", Expected: "[10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0]"},
				{Name: "Sum of range", Expression: "sum(1..10)", Description: "Sum numbers 1-10", Expected: "55"},
			},
		},

		// ============================================================
		// REGEX
		// ============================================================
		{
			Name:        "Regex",
			Icon:        "🔤",
			Description: "Regular expression matching: ~=",
			Examples: []Example{
				{Name: "Simple match", Expression: `"hello" ~= "^h"`, Description: "Starts with 'h'", Expected: litTrue},
				{Name: "Email pattern", Expression: `email ~= "^[a-z]+@[a-z]+\\.[a-z]+$"`, Description: "Basic email validation", Expected: litTrue, Env: map[string]any{exEmail: "test@example.com"}},
				{Name: "Phone pattern", Expression: `phone ~= "^\\+?[0-9]{10,}$"`, Description: "Phone validation", Expected: litTrue, Env: map[string]any{"phone": "+1234567890"}},
				{Name: "No match", Expression: `"hello" ~= "^world"`, Description: "Pattern doesn't match", Expected: litFalse},
				{Name: "Case insensitive", Expression: `"HELLO" ~= "(?i)^hello$"`, Description: "Case-insensitive match", Expected: litTrue},
				{Name: "Digits only", Expression: `"12345" ~= "^[0-9]+$"`, Description: "All digits", Expected: litTrue},
			},
		},

		// ============================================================
		// PIPE
		// ============================================================
		{
			Name:        "Pipe",
			Icon:        "🔀",
			Description: "Pipe operator for chaining: |",
			Examples: []Example{
				{Name: "Simple pipe", Expression: "[3, 1, 2] | sort", Description: "Pipe to sort", Expected: litArr123},
				{Name: "Pipe chain", Expression: "[3, 1, 2] | sort | first", Description: "Chained pipes", Expected: "1"},
				{Name: "Reverse first", Expression: "[1, 2, 3, 4, 5] | reverse | take(3)", Description: "Reverse then take", Expected: "[5, 4, 3]"},
				{Name: "Filter sum", Expression: "[1, 2, 3, 4, 5] | filter(x => x > 2) | sum", Description: "Filter then sum", Expected: "12"},
				{Name: "String pipe", Expression: `"  HELLO  " | trim | lower`, Description: "String transformations", Expected: litQuotedHello},
				{Name: "Complex chain", Expression: "data | filter(x => x.active) | pluck(\"name\") | sort", Description: "Data pipeline", Expected: `["Alice", "Charlie"]`, Env: map[string]any{exData: []any{map[string]any{exName: litCharlie, exActive: true}, map[string]any{exName: litBob, exActive: false}, map[string]any{exName: litAlice, exActive: true}}}},
			},
		},

		// ============================================================
		// ARRAYS
		// ============================================================
		{
			Name:        "Arrays",
			Icon:        "📚",
			Description: "Array literals, access, and slicing",
			Examples: []Example{
				{Name: "Literal", Expression: litArr123, Description: "Array literal", Expected: litArr123},
				{Name: "Mixed types", Expression: `[1, "two", true, nil]`, Description: "Mixed type array", Expected: `[1, "two", true, nil]`},
				{Name: "Index access", Expression: "arr[1]", Description: "Access by index", Expected: "2", Env: map[string]any{exArr: []any{1, 2, 3}}},
				{Name: "Negative index", Expression: "arr[-1]", Description: descLastElement, Expected: "3", Env: map[string]any{exArr: []any{1, 2, 3}}},
				{Name: "Negative second", Expression: "arr[-2]", Description: "Second to last", Expected: "2", Env: map[string]any{exArr: []any{1, 2, 3}}},
				{Name: "Slice", Expression: "arr[1:3]", Description: "Slice [1:3)", Expected: "[2, 3]", Env: map[string]any{exArr: []any{1, 2, 3, 4, 5}}},
				{Name: "Slice from start", Expression: "arr[:3]", Description: "First 3 elements", Expected: litArr123, Env: map[string]any{exArr: []any{1, 2, 3, 4, 5}}},
				{Name: "Slice to end", Expression: "arr[2:]", Description: "From index 2 to end", Expected: litArr345, Env: map[string]any{exArr: []any{1, 2, 3, 4, 5}}},
				{Name: "Slice step", Expression: "arr[::2]", Description: "Every second element", Expected: "[1, 3, 5]", Env: map[string]any{exArr: []any{1, 2, 3, 4, 5}}},
				{Name: "2D array", Expression: "matrix[0][1]", Description: "2D array access", Expected: "2", Env: map[string]any{"matrix": []any{[]any{1, 2}, []any{3, 4}}}},
				{Name: descEmptyArray, Expression: "[]", Description: descEmptyArray, Expected: "[]"},
				{Name: "Nested", Expression: "[[1, 2], [3, 4]]", Description: "Nested arrays", Expected: "[[1, 2], [3, 4]]"},
			},
		},

		// ============================================================
		// MAPS
		// ============================================================
		{
			Name:        "Maps",
			Icon:        "🗺️",
			Description: "Map/object literals and access",
			Examples: []Example{
				{Name: "Literal", Expression: `{"name": "Alice", "age": 30}`, Description: "Map literal", Expected: `{"age": 30, "name": "Alice"}`},
				{Name: "Dot access", Expression: "user.name", Description: "Dot notation", Expected: litQuotedAlice, Env: map[string]any{exUser: map[string]any{exName: litAlice, exAge: 30}}},
				{Name: "Bracket access", Expression: `user["name"]`, Description: "Bracket notation", Expected: litQuotedAlice, Env: map[string]any{exUser: map[string]any{exName: litAlice, exAge: 30}}},
				{Name: "Dynamic key", Expression: "user[key]", Description: "Dynamic key access", Expected: "30", Env: map[string]any{exUser: map[string]any{exName: litAlice, exAge: 30}, "key": exAge}},
				{Name: "Nested access", Expression: "data.user.address.city", Description: "Deep nested access", Expected: `"NYC"`, Env: map[string]any{exData: map[string]any{exUser: map[string]any{exAddress: map[string]any{"city": "NYC"}}}}},
				{Name: "Special keys", Expression: `obj["key-with-dash"]`, Description: "Keys with special chars", Expected: "value", Env: map[string]any{"obj": map[string]any{"key-with-dash": "value"}}},
				{Name: "Empty map", Expression: "{}", Description: "Empty map", Expected: "{}"},
				{Name: "Computed values", Expression: `{"sum": 1 + 2, "product": 3 * 4}`, Description: "Computed values in map", Expected: `{"product": 12, "sum": 3}`},
			},
		},

		// ============================================================
		// LET EXPRESSIONS
		// ============================================================
		{
			Name:        "Let Expressions",
			Icon:        "📝",
			Description: "Local variable bindings",
			Examples: []Example{
				{Name: "Simple let", Expression: "let x = 5; x * 2", Description: "Simple binding", Expected: "10"},
				{Name: "Multiple let", Expression: "let a = 10; let b = 20; a + b", Description: "Multiple bindings", Expected: "30"},
				{Name: "Computed let", Expression: "let total = price * qty; total * 1.2", Description: "Computed binding", Expected: "120", Env: map[string]any{exPrice: 50, exQty: 2}},
				{Name: "Let with expr", Expression: "let doubled = arr | map(x => x * 2); sum(doubled)", Description: "Complex expression", Expected: "12", Env: map[string]any{exArr: []any{1, 2, 3}}},
				{Name: "Chained let", Expression: "let a = 1; let b = a + 1; let c = b + 1; c", Description: "Chained dependencies", Expected: "3"},
				{Name: "Let in condition", Expression: "let isAdult = age >= 18; isAdult ? \"adult\" : \"minor\"", Description: "Let in ternary", Expected: `"adult"`, Env: map[string]any{exAge: 25}},
			},
		},

		// ============================================================
		// DESTRUCTURING
		// ============================================================
		{
			Name:        "Destructuring",
			Icon:        "📦",
			Description: "Object and array destructuring",
			Examples: []Example{
				{Name: "Object basic", Expression: `let {name, age} = user; name + " is " + age`, Description: "Basic object destructuring", Expected: `"Alice is 30"`, Env: map[string]any{exUser: map[string]any{exName: litAlice, exAge: 30}}},
				{Name: "With default", Expression: `let {name, role = "guest"} = user; role`, Description: "Destructuring with default", Expected: `"guest"`, Env: map[string]any{exUser: map[string]any{exName: litAlice}}},
				{Name: "With alias", Expression: `let {name: userName} = user; userName`, Description: "Destructuring with alias", Expected: litQuotedAlice, Env: map[string]any{exUser: map[string]any{exName: litAlice}}},
				{Name: descComplex, Expression: `let {name, role = "user", age: years} = person; sprintf("%s (%s, %v)", name, role, years)`, Description: "Combined features", Expected: `"Bob (user, 25)"`, Env: map[string]any{"person": map[string]any{exName: litBob, exAge: 25}}},
				{Name: "Array basic", Expression: "let [first, second] = arr; first + second", Description: "Array destructuring", Expected: "3", Env: map[string]any{exArr: []any{1, 2, 3}}},
				{Name: "Array default", Expression: "let [a, b, c = 0] = arr; c", Description: "Array with default", Expected: "0", Env: map[string]any{exArr: []any{1, 2}}},
				{Name: "Named underscore", Expression: "let [first, _, third] = arr; first + third", Description: "Named _ binding (not a placeholder)", Expected: "4", Env: map[string]any{exArr: []any{1, 2, 3}}},
			},
		},

		// ============================================================
		// SPREAD
		// ============================================================
		{
			Name:        "Spread",
			Icon:        "✨",
			Description: "Spread operator for arrays",
			Examples: []Example{
				{Name: "Basic spread", Expression: "[1, ...[2, 3], 4]", Description: "Spread in literal", Expected: litArr1234},
				{Name: "Merge arrays", Expression: "let merged = [...arr1, ...arr2]; merged", Description: "Merge two arrays", Expected: litArr1234, Env: map[string]any{"arr1": []any{1, 2}, "arr2": []any{3, 4}}},
				{Name: "With elements", Expression: "[0, ...middle, 100]", Description: "Spread with elements", Expected: "[0, 2, 3, 100]", Env: map[string]any{"middle": []any{2, 3}}},
				{Name: "Nested spread", Expression: "[...a, ...[...b, ...c]]", Description: "Nested spreads", Expected: litArr1234, Env: map[string]any{"a": []any{1}, "b": []any{2, 3}, "c": []any{4}}},
				{Name: "Empty spread", Expression: "[1, ...[], 2]", Description: "Spread empty array", Expected: "[1, 2]"},
				{Name: "Triple merge", Expression: "[...a, ...b, ...c]", Description: "Three array merge", Expected: "[1, 2, 3, 4, 5, 6]", Env: map[string]any{"a": []any{1, 2}, "b": []any{3, 4}, "c": []any{5, 6}}},
			},
		},

		// ============================================================
		// LAMBDAS
		// ============================================================
		{
			Name:        "Lambdas",
			Icon:        "λ",
			Description: "Lambda/arrow functions",
			Examples: []Example{
				{Name: "Single param", Expression: "map([1, 2, 3], x => x * 2)", Description: "Single parameter lambda", Expected: "[2, 4, 6]"},
				{Name: "Multi param", Expression: "reduce([1, 2, 3, 4], (acc, x) => acc + x, 0)", Description: "Two parameter lambda", Expected: "10"},
				{Name: "With expression", Expression: "filter(items, item => item.price < 100)", Description: "Lambda with property access", Expected: `[{"price": 50}]`, Env: map[string]any{exItems: []any{map[string]any{exPrice: 50}, map[string]any{exPrice: 150}}}},
				{Name: "Nested lambda", Expression: "map([1, 2, 3], x => map([1, 2], y => x * y))", Description: "Nested lambdas", Expected: "[[1, 2], [2, 4], [3, 6]]"},
				{Name: "Boolean lambda", Expression: "filter([1, 2, 3, 4, 5], x => x % 2 == 0)", Description: "Lambda returning bool", Expected: "[2, 4]"},
				{Name: "String lambda", Expression: `map(["hello", "world"], s => upper(s))`, Description: "Lambda with function call", Expected: `["HELLO", "WORLD"]`},
			},
		},

		// ============================================================
		// MATH FUNCTIONS
		// ============================================================
		{
			Name:        "Math Functions",
			Icon:        "🔢",
			Description: "Mathematical functions",
			Examples: []Example{
				{Name: "abs", Expression: "abs(-42)", Description: "Absolute value", Expected: "42"},
				{Name: "min", Expression: "min(5, 2, 8, 1, 9)", Description: "Minimum value", Expected: "1"},
				{Name: "max", Expression: "max(5, 2, 8, 1, 9)", Description: "Maximum value", Expected: "9"},
				{Name: "ceil", Expression: "ceil(3.2)", Description: "Ceiling", Expected: "4"},
				{Name: "floor", Expression: "floor(3.8)", Description: "Floor", Expected: "3"},
				{Name: "round", Expression: "round(3.5)", Description: "Round", Expected: "4"},
				{Name: "sqrt", Expression: "sqrt(16)", Description: "Square root", Expected: "4"},
				{Name: "pow", Expression: "pow(2, 8)", Description: descPower, Expected: "256"},
				{Name: exLog, Expression: "round(log(2.718281828) * 1000) / 1000", Description: "Natural log of e ≈ 1", Expected: "1"},
				{Name: "exp", Expression: "round(exp(1) * 100) / 100.0", Description: "e^1 ≈ 2.72", Expected: "2.72"},
				{Name: "sin", Expression: "sin(0)", Description: "Sine of 0", Expected: "0"},
				{Name: "cos", Expression: "cos(0)", Description: "Cosine of 0", Expected: "1"},
				{Name: "clamp", Expression: "clamp(15, 0, 10)", Description: "Clamp to range", Expected: "10"},
				{Name: "between", Expression: "between(5, 0, 10)", Description: "Check range", Expected: litTrue},
			},
		},

		// ============================================================
		// STRING FUNCTIONS
		// ============================================================
		{
			Name:        "String Functions",
			Icon:        "📝",
			Description: "String manipulation functions",
			Examples: []Example{
				{Name: "len", Expression: `len("hello")`, Description: "String length", Expected: "5"},
				{Name: "len unicode", Expression: `len("Привет")`, Description: "Unicode length (runes)", Expected: "6"},
				{Name: "upper", Expression: `upper("hello")`, Description: "To uppercase", Expected: `"HELLO"`},
				{Name: "lower", Expression: `lower("WORLD")`, Description: "To lowercase", Expected: `"world"`},
				{Name: "trim", Expression: `trim("  hello  ")`, Description: "Trim whitespace", Expected: litQuotedHello},
				{Name: "trimLeft", Expression: `trimLeft("xxhello", "x")`, Description: "Trim left chars", Expected: litQuotedHello},
				{Name: "trimRight", Expression: `trimRight("helloxx", "x")`, Description: "Trim right chars", Expected: litQuotedHello},
				{Name: "split", Expression: `split("a,b,c", ",")`, Description: "Split string", Expected: `["a", "b", "c"]`},
				{Name: "join", Expression: `join(["a", "b", "c"], "-")`, Description: "Join array", Expected: `"a-b-c"`},
				{Name: "contains", Expression: `contains("hello world", "world")`, Description: "Contains substring", Expected: litTrue},
				{Name: "startsWith", Expression: `startsWith("hello", "he")`, Description: "Starts with", Expected: litTrue},
				{Name: "endsWith", Expression: `endsWith("hello", "lo")`, Description: "Ends with", Expected: litTrue},
				{Name: "replace", Expression: `replace("hello", "l", "L")`, Description: "Replace all", Expected: `"heLLo"`},
				{Name: "substr", Expression: `substr("hello", 1, 3)`, Description: "Substring", Expected: `"ell"`},
				{Name: "indexOf", Expression: `indexOf("hello", "l")`, Description: "Index of", Expected: "2"},
				{Name: "repeat", Expression: exRepeatAB, Description: "Repeat value into an array", Expected: `["ab", "ab", "ab"]`},
			},
		},

		// ============================================================
		// ARRAY FUNCTIONS
		// ============================================================
		{
			Name:        "Array Functions",
			Icon:        "📚",
			Description: "Array manipulation functions",
			Examples: []Example{
				{Name: "first", Expression: "first([1, 2, 3])", Description: "First element", Expected: "1"},
				{Name: "last", Expression: "last([1, 2, 3])", Description: descLastElement, Expected: "3"},
				{Name: "take", Expression: "take([1, 2, 3, 4, 5], 3)", Description: "First n elements", Expected: litArr123},
				{Name: "skip", Expression: "skip([1, 2, 3, 4, 5], 2)", Description: "Skip n elements", Expected: litArr345},
				{Name: "reverse", Expression: "reverse([1, 2, 3])", Description: "Reverse array", Expected: "[3, 2, 1]"},
				{Name: "sort", Expression: "sort([3, 1, 4, 1, 5])", Description: "Sort ascending", Expected: "[1, 1, 3, 4, 5]"},
				{Name: "unique", Expression: "unique([1, 2, 2, 3, 3, 3])", Description: "Unique elements", Expected: litArr123},
				{Name: "concat", Expression: "concat([1, 2], [3, 4], [5])", Description: "Concatenate", Expected: litArr12345},
				{Name: "flatten", Expression: "flatten([[1, 2], [3, [4, 5]]])", Description: "Flatten recursively", Expected: "[1, 2, 3, 4, 5]"},
				{Name: "chunk", Expression: "chunk([1, 2, 3, 4, 5], 2)", Description: "Split into chunks", Expected: "[[1, 2], [3, 4], [5]]"},
				{Name: "zip", Expression: `zip([1, 2, 3], ["a", "b", "c"])`, Description: "Zip arrays", Expected: `[[1, "a"], [2, "b"], [3, "c"]]`},
				{Name: "compact", Expression: `compact([1, nil, 2, nil, 3])`, Description: "Remove nil values", Expected: litArr123},
				{Name: "range", Expression: "range(1, 5)", Description: "Generate range", Expected: litArr12345},
				{Name: "pluck", Expression: `pluck([{"name": "Alice"}, {"name": "Bob"}], "name")`, Description: "Extract field", Expected: `["Alice", "Bob"]`},
				{Name: "groupBy", Expression: `groupBy([{"t": "a", "v": 1}, {"t": "b", "v": 2}, {"t": "a", "v": 3}], x => x.t)`, Description: "Group by key", Expected: `[{"items": [{"t": "a", "v": 1}, {"t": "a", "v": 3}], "key": "a"}, {"items": [{"t": "b", "v": 2}], "key": "b"}]`},
				{Name: "sortBy asc", Expression: `sortBy([{"n": "B"}, {"n": "A"}, {"n": "C"}], x => x.n)`, Description: "Sort by field", Expected: `[{"n": "A"}, {"n": "B"}, {"n": "C"}]`},
				{Name: "sortBy desc", Expression: `sortBy([{"v": 1}, {"v": 3}, {"v": 2}], x => x.v, true)`, Description: "Sort desc", Expected: `[{"v": 3}, {"v": 2}, {"v": 1}]`},
			},
		},

		// ============================================================
		// AGGREGATE FUNCTIONS
		// ============================================================
		{
			Name:        "Aggregate Functions",
			Icon:        "📊",
			Description: "Aggregation and reduction functions",
			Examples: []Example{
				{Name: "sum", Expression: "sum([1, 2, 3, 4, 5])", Description: "Sum of array", Expected: "15"},
				{Name: "avg", Expression: "avg([2, 4, 6, 8, 10])", Description: "Average", Expected: "6"},
				{Name: "count", Expression: "count([1, 2, 3, 4])", Description: "Count elements", Expected: "4"},
				{Name: "all true", Expression: `all([true, 1, "x"])`, Description: "All truthy", Expected: litTrue},
				{Name: "all false", Expression: `all([true, 0, "x"])`, Description: "Not all truthy", Expected: litFalse},
				{Name: "any true", Expression: "any([false, 0, 1])", Description: "At least one truthy", Expected: litTrue},
				{Name: "any false", Expression: `any([false, 0, ""])`, Description: "None truthy", Expected: litFalse},
				{Name: "none", Expression: `none([false, 0, ""])`, Description: "All falsy", Expected: litTrue},
				{Name: "one", Expression: "one([false, true, false])", Description: "Exactly one truthy", Expected: litTrue},
			},
		},

		// ============================================================
		// HIGHER-ORDER FUNCTIONS
		// ============================================================
		{
			Name:        "Higher-Order Functions",
			Icon:        "🔄",
			Description: "map, filter, reduce, find, findIndex",
			Examples: []Example{
				{Name: exMap, Expression: "map([1, 2, 3], x => x * x)", Description: "Square each", Expected: "[1, 4, 9]"},
				{Name: "map objects", Expression: `map(users, u => u.name)`, Description: "Extract names", Expected: `["Alice", "Bob"]`, Env: map[string]any{exUsers: []any{map[string]any{exName: litAlice}, map[string]any{exName: litBob}}}},
				{Name: "filter", Expression: "filter([1, 2, 3, 4, 5], x => x > 2)", Description: "Filter by condition", Expected: litArr345},
				{Name: "filter objects", Expression: "filter(items, i => i.active)", Description: "Filter objects", Expected: `[{"active": true}]`, Env: map[string]any{exItems: []any{map[string]any{exActive: true}, map[string]any{exActive: false}}}},
				{Name: "reduce sum", Expression: "reduce([1, 2, 3, 4], (acc, x) => acc + x, 0)", Description: "Sum via reduce", Expected: "10"},
				{Name: "reduce max", Expression: "reduce([3, 1, 4, 1, 5], (m, x) => x > m ? x : m, 0)", Description: "Max via reduce", Expected: "5"},
				{Name: "find", Expression: "find([1, 2, 3, 4, 5], x => x > 3)", Description: "Find first match", Expected: "4"},
				{Name: "find object", Expression: "find(users, u => u.id == 2)", Description: "Find object", Expected: `{"id": 2, ...}`, Env: map[string]any{exUsers: []any{map[string]any{"id": 1}, map[string]any{"id": 2}}}},
				{Name: "findIndex", Expression: "findIndex([10, 20, 30, 40], x => x > 25)", Description: "Find index", Expected: "2"},
				{Name: "chain", Expression: "filter(map([1,2,3,4,5], x => x*2), x => x > 5)", Description: "Chained HOF", Expected: "[6, 8, 10]"},
			},
		},

		// ============================================================
		// TYPE FUNCTIONS
		// ============================================================
		{
			Name:        "Type Functions",
			Icon:        "🏷️",
			Description: "Type conversion and checking",
			Examples: []Example{
				{Name: "int from string", Expression: `int("42")`, Description: "String to int", Expected: "42"},
				{Name: "int from float", Expression: "int(3.9)", Description: "Float to int (truncate)", Expected: "3"},
				{Name: "float from string", Expression: `float("3.14")`, Description: "String to float", Expected: litPi},
				{Name: "float from int", Expression: "float(42)", Description: "Int to float", Expected: "42"},
				{Name: "string from int", Expression: "string(42)", Description: "Int to string", Expected: `"42"`},
				{Name: "string from bool", Expression: "string(true)", Description: "Bool to string", Expected: `"true"`},
				{Name: "bool truthy", Expression: "bool(1)", Description: "Truthy to bool", Expected: litTrue},
				{Name: "bool falsy", Expression: "bool(0)", Description: "Falsy to bool", Expected: litFalse},
				{Name: "type int", Expression: "type(42)", Description: "Type of int", Expected: `"int"`},
				{Name: "type array", Expression: "type([1, 2, 3])", Description: "Type of array", Expected: `"array"`},
				{Name: "type map", Expression: `type({"a": 1})`, Description: "Type of map", Expected: `"map"`},
				{Name: "isNil true", Expression: "isNil(nil)", Description: "Check nil", Expected: litTrue},
				{Name: "isNil false", Expression: "isNil(0)", Description: "0 is not nil", Expected: litFalse},
			},
		},

		// ============================================================
		// UTILITY FUNCTIONS
		// ============================================================
		{
			Name:        "Utility Functions",
			Icon:        "🛠️",
			Description: "Utility and helper functions",
			Examples: []Example{
				{Name: "keys", Expression: `keys({"a": 1, "b": 2, "c": 3})`, Description: "Map keys", Expected: `["a", "b", "c"]`},
				{Name: "values", Expression: `values({"a": 1, "b": 2, "c": 3})`, Description: "Map values", Expected: litArr123},
				{Name: "hasKey true", Expression: `hasKey({"name": "Alice"}, "name")`, Description: "Has key", Expected: litTrue},
				{Name: "hasKey false", Expression: `hasKey({"name": "Alice"}, "age")`, Description: "Missing key", Expected: litFalse},
				{Name: "empty array", Expression: "empty([])", Description: descEmptyArray, Expected: litTrue},
				{Name: "empty string", Expression: `empty("")`, Description: "Empty string", Expected: litTrue},
				{Name: "notEmpty", Expression: `notEmpty("hello")`, Description: "Not empty", Expected: litTrue},
				{Name: "default nil", Expression: "default(nil, 42)", Description: "Default for nil", Expected: "42"},
				{Name: "default value", Expression: "default(10, 42)", Description: "Has value", Expected: "10"},
				{Name: "coalesce", Expression: "coalesce(nil, nil, 5, 10)", Description: "First non-nil", Expected: "5"},
				{Name: "truthy", Expression: "truthy([0, 1, nil, 2])", Description: "Remove falsy values", Expected: "[1, 2]"},
				{Name: "sprintf", Expression: `sprintf("%s is %d years old", "Alice", 30)`, Description: "Format string", Expected: `"Alice is 30 years old"`},
				{Name: "toJSON", Expression: `toJSON({"name": "Alice", "age": 30})`, Description: "To JSON", Expected: `"{\"age\":30,\"name\":\"Alice\"}"`},
				{Name: "fromJSON", Expression: `fromJSON("{\"x\": 42}")`, Description: "Parse JSON", Expected: `{"x": 42}`},
			},
		},

		// ============================================================
		// UNICODE
		// ============================================================
		{
			Name:        "Unicode",
			Icon:        "🌍",
			Description: "Unicode support in identifiers and strings",
			Examples: []Example{
				{Name: "Cyrillic string", Expression: `len("Привет")`, Description: "Russian string length", Expected: "6"},
				{Name: "Chinese string", Expression: `len("日本語")`, Description: "Japanese string length", Expected: "3"},
				{Name: "Emoji string", Expression: `len("🎉🎊🎁")`, Description: "Emoji string length", Expected: "3"},
				{Name: "Unicode index", Expression: `"Привет"[0]`, Description: "First char (Cyrillic)", Expected: `"П"`},
				{Name: "Unicode slice", Expression: `"Привет"[0:3]`, Description: "Slice Cyrillic", Expected: `"При"`},
				{Name: "Mixed unicode", Expression: `"Hello, 世界!"`, Description: "Mixed unicode string", Expected: `"Hello, 世界!"`},
				{Name: "Escape sequence", Expression: `"\u0041\u0042\u0043"`, Description: "Unicode escapes", Expected: `"ABC"`},
				{Name: "Upper unicode", Expression: `upper("привет")`, Description: "Uppercase Cyrillic", Expected: `"ПРИВЕТ"`},
			},
		},

		// ============================================================
		// REAL SCENARIOS
		// ============================================================
		{
			Name:        "Real Scenarios",
			Icon:        "💼",
			Description: "Real-world use cases",
			Examples: []Example{
				{Name: "Discount rule", Expression: `let discount = tier == "gold" ? 0.2 : 0.1; price * (1 - discount)`, Description: "Tiered discount", Expected: "80", Env: map[string]any{"tier": "gold", exPrice: 100}},
				{Name: "Age validation", Expression: "age >= 18 && age <= 120", Description: "Valid age range", Expected: litTrue, Env: map[string]any{exAge: 25}},
				{Name: "Email check", Expression: `email ~= "^[a-z]+@[a-z]+\\.[a-z]+$"`, Description: "Basic email validation", Expected: litTrue, Env: map[string]any{exEmail: "test@example.com"}},
				{Name: "Filter active", Expression: "filter(users, u => u.active) | pluck(\"name\")", Description: "Active user names", Expected: `["Alice"]`, Env: map[string]any{exUsers: []any{map[string]any{exName: litAlice, exActive: true}, map[string]any{exName: litBob, exActive: false}}}},
				{Name: "Total price", Expression: "sum(map(items, i => i.price * i.qty))", Description: "Cart total", Expected: "150", Env: map[string]any{exItems: []any{map[string]any{exPrice: 10, exQty: 5}, map[string]any{exPrice: 20, exQty: 5}}}},
				{Name: "User greeting", Expression: `let name = user?.name ?? "Guest"; "Hello, " + name + "!"`, Description: "Safe greeting", Expected: `"Hello, Guest!"`, Env: map[string]any{exUser: nil}},
				{Name: "Permission check", Expression: "(role == \"admin\" || role == \"moderator\") && active", Description: "RBAC check", Expected: litTrue, Env: map[string]any{exRole: exAdmin, exActive: true}},
				{Name: "Data transform", Expression: `let {name, age} = user; {"displayName": upper(name), "isAdult": age >= 18}`, Description: "Transform user", Expected: `{"displayName": "ALICE", "isAdult": true}`, Env: map[string]any{exUser: map[string]any{exName: litAlice, exAge: 25}}},
				{Name: "Pagination", Expression: "skip(take(items, (page + 1) * size), page * size)", Description: "Paginate array", Expected: "[3, 4]", Env: map[string]any{exItems: []any{1, 2, 3, 4, 5, 6}, "page": 1, "size": 2}},
				{Name: "Conditional sum", Expression: "sum(filter(transactions, t => t.type == \"credit\") | pluck(\"amount\"))", Description: "Sum credits only", Expected: "300", Env: map[string]any{"transactions": []any{map[string]any{exType: "credit", exAmount: 100}, map[string]any{exType: "debit", exAmount: 50}, map[string]any{exType: "credit", exAmount: 200}}}},

				// E-commerce: Complex order processing
				{Name: "Order total with tax & discount",
					Expression:  `let subtotal = sum(cart | map(i => i.price * i.qty)); let discount = subtotal > 100 ? 0.1 : 0; let taxRate = 0.2; round((subtotal * (1 - discount)) * (1 + taxRate) * 100) / 100.0`,
					Description: "Calculate order: subtotal → discount → tax",
					Expected:    "216",
					Env: map[string]any{"cart": []any{
						map[string]any{exName: "Widget", exPrice: 25, exQty: 2},
						map[string]any{exName: "Gadget", exPrice: 75, exQty: 2},
					}}},

				// Analytics: Team stats
				{Name: "Team sales stats",
					Expression:  `let active = employees | filter(e => e.sales > 0); {"total": sum(active | pluck("sales")), "count": len(active), "avg": round(avg(active | pluck("sales"))), "top": max(active | pluck("sales"))}`,
					Description: "Calculate team sales metrics",
					Expected:    `{"avg": 11667, "count": 3, "top": 15000, "total": 35000}`,
					Env: map[string]any{"employees": []any{
						map[string]any{exName: litAlice, exSalesNum: 15000, exDept: litSales},
						map[string]any{exName: litBob, exSalesNum: 8000, exDept: litSales},
						map[string]any{exName: litCharlie, exSalesNum: 12000, exDept: "Marketing"},
						map[string]any{exName: litDiana, exSalesNum: 0, exDept: "Support"},
					}}},

				// User management: Role-based access
				{Name: "Access control matrix",
					Expression:  `let perms = role == "admin" ? ["read", "write", "delete", "admin"] : role == "editor" ? ["read", "write"] : ["read"]; {"user": username, "role": role, "canDelete": "delete" in perms, "canAdmin": "admin" in perms}`,
					Description: "Dynamic permission assignment",
					Expected:    `{"canAdmin": false, "canDelete": false, "role": "editor", "user": "john"}`,
					Env:         map[string]any{"username": "john", exRole: "editor"}},

				// Data pipeline: Log analysis
				{Name: "Error log analysis",
					Expression:  `let errors = logs | filter(l => l.level == "error"); let sources = unique(errors | pluck("source")); sources | map(s => {"source": s, "count": len(errors | filter(e => e.source == s)), "messages": errors | filter(e => e.source == s) | pluck("message")})`,
					Description: "Analyze errors by source",
					Expected:    `[{"count": 2, "messages": ["Rate limit exceeded", "Connection timeout"], "source": "api"}, {"count": 1, "messages": ["Query failed"], "source": "db"}]`,
					Env: map[string]any{"logs": []any{
						map[string]any{exLevel: exError, exSource: exApi, exMessage: "Rate limit exceeded"},
						map[string]any{exLevel: "info", exSource: exApi, exMessage: "Request OK"},
						map[string]any{exLevel: exError, exSource: "db", exMessage: "Query failed"},
						map[string]any{exLevel: exError, exSource: exApi, exMessage: "Connection timeout"},
					}}},

				// Inventory: Stock management
				{Name: "Low stock alerts",
					Expression:  `let low = products | filter(p => p.stock <= p.reorderLevel); {"count": len(low), "items": low | map(p => p.name + ": " + string(p.stock) + "/" + string(p.reorderLevel)), "totalNeeded": sum(low | map(p => p.reorderLevel * 2 - p.stock))}`,
					Description: "Find products needing reorder",
					Expected:    `{"count": 2, "items": ["Widget A: 5/10", "Widget B: 3/10"], "totalNeeded": 32}`,
					Env: map[string]any{"products": []any{
						map[string]any{exSku: "WA001", exName: "Widget A", exStock: 5, exReorderLevel: 10},
						map[string]any{exSku: "WB002", exName: "Widget B", exStock: 3, exReorderLevel: 10},
						map[string]any{exSku: "WC003", exName: "Widget C", exStock: 50, exReorderLevel: 10},
					}}},

				// Financial: Portfolio analysis
				{Name: "Portfolio summary",
					Expression:  `let totalValue = sum(holdings | map(h => h.shares * h.price)); let gains = holdings | filter(h => h.price > h.cost) | map(h => (h.price - h.cost) * h.shares); {"totalValue": totalValue, "totalGain": sum(gains), "winners": len(gains), "losers": len(holdings) - len(gains)}`,
					Description: "Calculate portfolio metrics",
					Expected:    `{"losers": 1, "totalGain": 460, "totalValue": 3200, "winners": 2}`,
					Env: map[string]any{"holdings": []any{
						map[string]any{exSymbol: "AAPL", exShares: 10, exPrice: 150, exCost: 120},
						map[string]any{exSymbol: "GOOGL", exShares: 5, exPrice: 180, exCost: 200},
						map[string]any{exSymbol: "MSFT", exShares: 8, exPrice: 100, exCost: 80},
					}}},

				// Text processing: Data cleanup
				{Name: "Clean & normalize data",
					Expression:  `records | map(r => {"id": r.id, "email": lower(trim(r.email)), "name": trim(r.name) | split(" ") | map(w => upper(substr(w, 0, 1)) + lower(substr(w, 1))) | join(" ")})`,
					Description: "Normalize emails and title-case names",
					Expected:    `[{"email": "john@test.com", "id": 1, "name": "John Doe"}, {"email": "jane@test.com", "id": 2, "name": "Jane Smith"}]`,
					Env: map[string]any{"records": []any{
						map[string]any{"id": 1, exEmail: "  JOHN@TEST.COM  ", exName: "john doe"},
						map[string]any{"id": 2, exEmail: "Jane@Test.com", exName: "JANE SMITH"},
					}}},

				// Scheduling: Time slot finder
				{Name: "Available time slots",
					Expression:  `let booked = appointments | pluck("hour"); range(9, 17) | filter(h => !(h in booked)) | map(h => string(h) + ":00")`,
					Description: "Find free hours (9-17)",
					Expected:    `["9:00", "11:00", "12:00", "13:00", "15:00", "17:00"]`,
					Env: map[string]any{"appointments": []any{
						map[string]any{exHour: 10, exTitle: "Meeting"},
						map[string]any{exHour: 14, exTitle: "Call"},
						map[string]any{exHour: 16, exTitle: "Review"},
					}}},

				// Reporting: Sales dashboard
				{Name: "Sales by region",
					Expression:  `let regions = unique(sales | pluck("region")); regions | map(r => {"region": r, "total": sum(sales | filter(s => s.region == r) | pluck("amount")), "count": len(sales | filter(s => s.region == r)), "avg": round(avg(sales | filter(s => s.region == r) | pluck("amount")))})`,
					Description: "Aggregate sales metrics by region",
					Expected:    `[{"avg": 250, "count": 2, "region": "East", "total": 500}, {"avg": 350, "count": 2, "region": "West", "total": 700}]`,
					Env: map[string]any{exSalesNum: []any{
						map[string]any{exRegion: "East", exAmount: 200, exRep: litAlice},
						map[string]any{exRegion: "West", exAmount: 350, exRep: litBob},
						map[string]any{exRegion: "East", exAmount: 300, exRep: litCharlie},
						map[string]any{exRegion: "West", exAmount: 350, exRep: litDiana},
					}}},

				// Validation: Form data checker
				{Name: "Form validation",
					Expression:  `let errors = []; let errors = len(form.name ?? "") < 2 ? [...errors, "Name too short"] : errors; let errors = !(form.email ?? "" ~= "@") ? [...errors, "Invalid email"] : errors; let errors = (form.age ?? 0) < 18 ? [...errors, "Must be 18+"] : errors; {"valid": len(errors) == 0, "errors": errors}`,
					Description: "Multi-field validation with error list",
					Expected:    `{"errors": ["Invalid email", "Must be 18+"], "valid": false}`,
					Env:         map[string]any{"form": map[string]any{exName: "Jo", exEmail: "invalid", exAge: 16}}},

				// GroupBy: Basic grouping
				{Name: "Group users by role",
					Expression:  `groupBy(users, u => u.role) | map(g => {"role": g.key, "names": g.items | pluck("name")})`,
					Description: "Group users and extract names per role",
					Expected:    `[{"names": ["Alice", "Charlie"], "role": "admin"}, {"names": ["Bob"], "role": "user"}]`,
					Env: map[string]any{exUsers: []any{
						map[string]any{exName: litAlice, exRole: exAdmin},
						map[string]any{exName: litBob, exRole: exUser},
						map[string]any{exName: litCharlie, exRole: exAdmin},
					}}},

				// GroupBy: With aggregation
				{Name: "Sales by category",
					Expression:  `groupBy(sales, s => s.category) | map(g => {"category": g.key, "total": sum(g.items | pluck("amount")), "count": len(g.items)})`,
					Description: "Aggregate sales by product category",
					Expected:    `[{"category": "Electronics", "count": 2, "total": 1500}, {"category": "Books", "count": 1, "total": 30}]`,
					Env: map[string]any{exSalesNum: []any{
						map[string]any{exProduct: "Laptop", exCategory: "Electronics", exAmount: 1000},
						map[string]any{exProduct: "Book", exCategory: "Books", exAmount: 30},
						map[string]any{exProduct: "Phone", exCategory: "Electronics", exAmount: 500},
					}}},

				// SortBy: Ascending
				{Name: "Sort by age",
					Expression:  `sortBy(people, p => p.age) | pluck("name")`,
					Description: "Sort people by age ascending",
					Expected:    `["Charlie", "Alice", "Bob"]`,
					Env: map[string]any{"people": []any{
						map[string]any{exName: litAlice, exAge: 30},
						map[string]any{exName: litBob, exAge: 45},
						map[string]any{exName: litCharlie, exAge: 25},
					}}},

				// SortBy: Descending
				{Name: "Top scores",
					Expression:  `sortBy(players, p => p.score, true) | take(3) | map(p => p.name + ": " + string(p.score))`,
					Description: "Top 3 players by score",
					Expected:    `["Diana: 950", "Alice: 850", "Charlie: 720"]`,
					Env: map[string]any{"players": []any{
						map[string]any{exName: litAlice, exScore: 850},
						map[string]any{exName: litBob, exScore: 650},
						map[string]any{exName: litCharlie, exScore: 720},
						map[string]any{exName: litDiana, exScore: 950},
					}}},

				// Combined: GroupBy + SortBy
				{Name: "Department leaderboard",
					Expression:  `let byDept = groupBy(employees, e => e.dept); byDept | map(g => {"dept": g.key, "topPerformer": first(sortBy(g.items, e => e.sales, true)).name, "totalSales": sum(g.items | pluck("sales"))})`,
					Description: "Top performer and total sales per department",
					Expected:    `[{"dept": "Sales", "topPerformer": "Alice", "totalSales": 25000}, {"dept": "Marketing", "topPerformer": "Charlie", "totalSales": 12000}]`,
					Env: map[string]any{"employees": []any{
						map[string]any{exName: litAlice, exDept: litSales, exSalesNum: 15000},
						map[string]any{exName: litBob, exDept: litSales, exSalesNum: 10000},
						map[string]any{exName: litCharlie, exDept: "Marketing", exSalesNum: 12000},
					}}},
			},
		},

		// ============================================================
		// ERROR CASES
		// ============================================================
		{
			Name:        "Error Cases",
			Icon:        "❌",
			Description: "Error handling and edge cases",
			Examples: []Example{
				{Name: "Division by zero", Expression: "10 / 0", Description: "Triggers runtime error", Expected: "Error: division by zero"},
				{Name: "Index out of bounds", Expression: "arr[10]", Description: "Array index error", Expected: "Error: index out of bounds", Env: map[string]any{exArr: []any{1, 2, 3}}},
				{Name: "Undefined variable", Expression: "unknownVar + 1", Description: "Undefined variable error", Expected: "Error: undefined variable"},
				{Name: "Type coercion", Expression: `"hello" + 42`, Description: "String + int (auto-converts)", Expected: `"hello42"`},
				{Name: "Invalid regex", Expression: `"test" ~= "[invalid"`, Description: "Bad regex pattern", Expected: "Error: invalid regex"},
				{Name: "Nil access", Expression: "nil.property", Description: "Access property of nil", Expected: "Error: cannot access property of nil"},
				{Name: "Not callable", Expression: "42()", Description: "Call non-function", Expected: "Error: value is not callable"},
				{Name: "Wrong arg count", Expression: "abs()", Description: "Missing argument", Expected: "Error: wrong number of arguments"},
			},
		},
	}
}
