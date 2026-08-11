package main

// Config holds application configuration.
type Config struct {
	Port    int
	Verbose bool
}

// BuiltinFunction describes a built-in function.
type BuiltinFunction struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Signature   string `json:"signature"`
	Example     string `json:"example"`
}

// GetBuiltinFunctions returns all built-in functions documentation.
func GetBuiltinFunctions() []BuiltinFunction {
	return []BuiltinFunction{
		// Math
		{Name: "abs", Category: catMath, Description: "Absolute value", Signature: "abs(x) → number", Example: "abs(-5)"},
		{Name: "min", Category: catMath, Description: "Minimum of values", Signature: "min(a, b, ...) → number", Example: "min(3, 1, 4)"},
		{Name: "max", Category: catMath, Description: "Maximum of values", Signature: "max(a, b, ...) → number", Example: "max(3, 1, 4)"},
		{Name: "ceil", Category: catMath, Description: "Round up", Signature: "ceil(x) → int", Example: "ceil(3.2)"},
		{Name: "floor", Category: catMath, Description: "Round down", Signature: "floor(x) → int", Example: "floor(3.8)"},
		{Name: "round", Category: catMath, Description: "Round to nearest", Signature: "round(x) → int", Example: "round(3.5)"},
		{Name: "sqrt", Category: catMath, Description: "Square root", Signature: "sqrt(x) → float", Example: "sqrt(16)"},
		{Name: "pow", Category: catMath, Description: descPower, Signature: "pow(base, exp) → number", Example: "pow(2, 10)"},
		{Name: exLog, Category: catMath, Description: "Natural logarithm", Signature: "log(x) → float", Example: "log(2.718)"},
		{Name: "exp", Category: catMath, Description: "Exponential (e^x)", Signature: "exp(x) → float", Example: "exp(1)"},
		{Name: "sin", Category: catMath, Description: "Sine (radians)", Signature: "sin(x) → float", Example: "sin(0)"},
		{Name: "cos", Category: catMath, Description: "Cosine (radians)", Signature: "cos(x) → float", Example: "cos(0)"},
		{Name: "clamp", Category: catMath, Description: "Clamp to range", Signature: "clamp(x, min, max) → number", Example: "clamp(15, 0, 10)"},
		{Name: "between", Category: catMath, Description: "Check if in range", Signature: "between(x, min, max) → bool", Example: "between(5, 0, 10)"},

		// String
		{Name: "len", Category: catString, Description: "Length (runes)", Signature: "len(s) → int", Example: `len("hello")`},
		{Name: "upper", Category: catString, Description: "To uppercase", Signature: "upper(s) → string", Example: `upper("hello")`},
		{Name: "lower", Category: catString, Description: "To lowercase", Signature: "lower(s) → string", Example: `lower("HELLO")`},
		{Name: "trim", Category: catString, Description: "Trim whitespace", Signature: "trim(s) → string", Example: `trim("  hi  ")`},
		{Name: "trimLeft", Category: catString, Description: "Trim left", Signature: "trimLeft(s, chars?) → string", Example: `trimLeft("xxhello", "x")`},
		{Name: "trimRight", Category: catString, Description: "Trim right", Signature: "trimRight(s, chars?) → string", Example: `trimRight("helloxx", "x")`},
		{Name: "split", Category: catString, Description: "Split string", Signature: "split(s, sep) → array", Example: `split("a,b,c", ",")`},
		{Name: "join", Category: catString, Description: "Join array", Signature: "join(arr, sep) → string", Example: `join(["a","b"], "-")`},
		{Name: "contains", Category: catString, Description: "Contains substring", Signature: "contains(s, sub) → bool", Example: `contains("hello", "ell")`},
		{Name: "startsWith", Category: catString, Description: "Starts with", Signature: "startsWith(s, prefix) → bool", Example: `startsWith("hello", "he")`},
		{Name: "endsWith", Category: catString, Description: "Ends with", Signature: "endsWith(s, suffix) → bool", Example: `endsWith("hello", "lo")`},
		{Name: "replace", Category: catString, Description: "Replace all", Signature: "replace(s, old, new) → string", Example: `replace("hello", "l", "L")`},
		{Name: "substr", Category: catString, Description: "Substring", Signature: "substr(s, start, len?) → string", Example: `substr("hello", 1, 3)`},
		{Name: "indexOf", Category: catString, Description: "Index of substring", Signature: "indexOf(s, sub) → int", Example: `indexOf("hello", "l")`},
		{Name: "repeat", Category: catString, Description: "Repeat string", Signature: "repeat(s, n) → string", Example: `repeat("ab", 3)`},

		// Array
		{Name: "first", Category: catArray, Description: "First element", Signature: "first(arr) → any", Example: "first([1,2,3])"},
		{Name: "last", Category: catArray, Description: descLastElement, Signature: "last(arr) → any", Example: "last([1,2,3])"},
		{Name: "take", Category: catArray, Description: "First n elements", Signature: "take(arr, n) → array", Example: "take([1,2,3,4], 2)"},
		{Name: "skip", Category: catArray, Description: "Skip n elements", Signature: "skip(arr, n) → array", Example: "skip([1,2,3,4], 2)"},
		{Name: "reverse", Category: catArray, Description: "Reverse array", Signature: "reverse(arr) → array", Example: "reverse([1,2,3])"},
		{Name: "sort", Category: catArray, Description: "Sort array", Signature: "sort(arr) → array", Example: "sort([3,1,2])"},
		{Name: "unique", Category: catArray, Description: "Unique elements", Signature: "unique(arr) → array", Example: "unique([1,2,2,3])"},
		{Name: "concat", Category: catArray, Description: "Concatenate arrays", Signature: "concat(a, b, ...) → array", Example: "concat([1], [2], [3])"},
		{Name: "flatten", Category: catArray, Description: "Flatten nested", Signature: "flatten(arr) → array", Example: "flatten([[1,2], [3]])"},
		{Name: "chunk", Category: catArray, Description: "Split into chunks", Signature: "chunk(arr, size) → array", Example: "chunk([1,2,3,4], 2)"},
		{Name: "zip", Category: catArray, Description: "Zip arrays", Signature: "zip(a, b, ...) → array", Example: `zip([1,2], ["a","b"])`},
		{Name: "compact", Category: catArray, Description: "Remove nil values", Signature: "compact(arr) → array", Example: `compact([1,nil,2])`},
		{Name: "range", Category: catArray, Description: "Generate range", Signature: "range(start, end, step?) → array", Example: "range(0, 10, 2)"},
		{Name: "pluck", Category: catArray, Description: "Extract field", Signature: "pluck(arr, key) → array", Example: `pluck(users, "name")`},
		{Name: "groupBy", Category: catArray, Description: "Group by key", Signature: "groupBy(arr, fn) → [{key, items}, ...]", Example: `groupBy(users, u => u.role)`},
		{Name: "sortBy", Category: catArray, Description: "Sort by key", Signature: "sortBy(arr, fn, desc?) → array", Example: `sortBy(users, u => u.age)`},

		// Aggregate
		{Name: "sum", Category: catAggregate, Description: "Sum of array", Signature: "sum(arr) → number", Example: "sum([1,2,3,4])"},
		{Name: "avg", Category: catAggregate, Description: "Average", Signature: "avg(arr) → float", Example: "avg([1,2,3,4])"},
		{Name: "count", Category: catAggregate, Description: "Count elements", Signature: "count(arr) → int", Example: "count([1,2,3])"},
		{Name: "all", Category: catAggregate, Description: "All truthy?", Signature: "all(arr) → bool", Example: `all([true, 1, "x"])`},
		{Name: "any", Category: catAggregate, Description: "Any truthy?", Signature: "any(arr) → bool", Example: "any([false, 0, 1])"},
		{Name: "none", Category: catAggregate, Description: "All falsy?", Signature: "none(arr) → bool", Example: `none([false, 0, ""])`},
		{Name: "one", Category: catAggregate, Description: "Exactly one truthy?", Signature: "one(arr) → bool", Example: "one([false, true, false])"},

		// Higher-Order
		{Name: exMap, Category: catHigherOrder, Description: "Transform each", Signature: "map(arr, fn) → array", Example: "map([1,2,3], x => x * 2)"},
		{Name: "filter", Category: catHigherOrder, Description: "Filter by condition", Signature: "filter(arr, fn) → array", Example: "filter([1,2,3,4], x => x > 2)"},
		{Name: "reduce", Category: catHigherOrder, Description: "Reduce to single value", Signature: "reduce(arr, fn, init?) → any", Example: "reduce([1,2,3], (a,x) => a+x, 0)"},
		{Name: "find", Category: catHigherOrder, Description: "Find first match", Signature: "find(arr, fn) → any", Example: "find([1,2,3], x => x > 1)"},
		{Name: "findIndex", Category: catHigherOrder, Description: "Index of first match", Signature: "findIndex(arr, fn) → int", Example: "findIndex([1,2,3], x => x > 1)"},

		// Type
		{Name: "int", Category: catType, Description: "Convert to int", Signature: "int(x) → int", Example: `int("42")`},
		{Name: "float", Category: catType, Description: "Convert to float", Signature: "float(x) → float", Example: `float("3.14")`},
		{Name: "string", Category: catType, Description: "Convert to string", Signature: "string(x) → string", Example: "string(42)"},
		{Name: "bool", Category: catType, Description: "Convert to bool", Signature: "bool(x) → bool", Example: "bool(1)"},
		{Name: exType, Category: catType, Description: "Get type name", Signature: "type(x) → string", Example: "type([1,2])"},
		{Name: "isNil", Category: catType, Description: "Check if nil", Signature: "isNil(x) → bool", Example: "isNil(null)"},

		// Utility
		{Name: "keys", Category: catUtility, Description: "Map keys", Signature: "keys(map) → array", Example: `keys({"a":1,"b":2})`},
		{Name: "values", Category: catUtility, Description: "Map values", Signature: "values(map) → array", Example: `values({"a":1,"b":2})`},
		{Name: "hasKey", Category: catUtility, Description: "Has key?", Signature: "hasKey(map, key) → bool", Example: `hasKey(obj, "name")`},
		{Name: "empty", Category: catUtility, Description: "Is empty?", Signature: "empty(x) → bool", Example: "empty([])"},
		{Name: "notEmpty", Category: catUtility, Description: "Is not empty?", Signature: "notEmpty(x) → bool", Example: `notEmpty("hi")`},
		{Name: "default", Category: catUtility, Description: "Default if nil", Signature: "default(x, fallback) → any", Example: "default(nil, 42)"},
		{Name: "coalesce", Category: catUtility, Description: "First non-nil", Signature: "coalesce(a, b, ...) → any", Example: "coalesce(nil, nil, 5)"},
		{Name: "truthy", Category: catUtility, Description: "Remove falsy values", Signature: "truthy(arr) → array", Example: "truthy([0,1,nil])"},
		{Name: "print", Category: catUtility, Description: "Debug print", Signature: "print(...) → nil", Example: `print("x =", x)`},
		{Name: "sprintf", Category: catUtility, Description: "Format string", Signature: "sprintf(fmt, ...) → string", Example: `sprintf("%d + %d = %d", 1, 2, 3)`},
		{Name: "toJSON", Category: catUtility, Description: "To JSON string", Signature: "toJSON(x) → string", Example: `toJSON({"a":1})`},
		{Name: "fromJSON", Category: catUtility, Description: "Parse JSON", Signature: "fromJSON(s) → any", Example: `fromJSON('{"a":1}')`},
	}
}
