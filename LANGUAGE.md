# HyperEx Language Guide and Specification

> **Normative English guide for `github.com/aasyanov/hyperex` v1.5.1 (2026-11).**
> API overview: [README.md](README.md) · [doc.go](doc.go) · History: [CHANGELOG.md](CHANGELOG.md).

This document teaches the expression language step by step and also serves as its reference. It covers syntax, evaluation rules, collections, functions, numeric and temporal behavior, host-side policies, Studio tools, and common failure modes. It intentionally includes both introductory explanations and exact edge cases.

When this document and an older example disagree, the **current implementation and this specification win**. If the implementation and this document disagree, report a documentation or implementation defect rather than silently choosing one.

## How to read this guide

Start with §1–§5 if you are new to HyperEx. Those sections explain the small core: one expression, values, variables, operators, and collections. Continue with §6–§9 for lambdas, pipes, and built-ins. Read §10–§12 before using untrusted expressions: numeric behavior, sandboxing, and host options change observable behavior and are just as important as grammar.

Examples use `hyperex` for an expression and `text` for its result. Results show the logical value rather than Go formatting details such as `int64(3)` versus `float64(3.0)`. Where the distinction matters, it is stated explicitly.

A typical Go call is:

```go
program, err := hyperex.Compile(`price * quantity`)
if err != nil {
    // Handle a parse, type-check, or compile error.
}
result, err := program.Eval(ctx, map[string]any{
    "price":    99.99,
    "quantity": 3,
})
```

The rest of this guide is about the expression inside `Compile` and the environment in which it executes.

---

## 1. The program model: one expression, evaluated once

### 1.1. A program ends at EOF

A HyperEx source unit contains exactly one expression followed by end-of-input:

```text
program ::= expression EOF
```

It is not a script, statement list, template, module, or general-purpose programming language. There are no statement blocks, imports, packages, `for` or `while` loops, `return`, `break`, or `continue`.

These are valid programs:

```hyperex
2 + 2 * 3
```

```hyperex
let subtotal = price * quantity; subtotal * 1.2
```

```hyperex
users | filter(u => u.active) | pluck("name") | sort
```

This is not a program:

```hyperex
let x = 1
x + 2
```

Use `let` and a semicolon inside the single expression instead:

```hyperex
let x = 1; x + 2
```

### 1.2. Compilation and evaluation are separate

The normal lifecycle is:

```text
source text → lexer → Pratt parser → AST → optimizer → bytecode → VM → Value
```

`Compile` performs parsing, optional type checking and optimization, and bytecode generation. `Program.Eval` executes that immutable compiled program against a particular environment and context. Compile once at configuration load and reuse the `Program` for many evaluations; do not recompile on every event unless the expression changed.

### 1.3. The environment supplies names and behavior

An expression can refer to values provided by the host:

```hyperex
price * quantity + fee
```

```json
{"price": 10, "quantity": 3, "fee": 2}
```

```text
32
```

The preferred hot-path environment is `map[string]any`. A struct or pointer to struct is also accepted. Exported fields are mapped by their Go name or by an exact `expr` tag:

```go
type Env struct {
    Price    float64 `expr:"price"`
    Quantity int     `expr:"quantity"`
    Internal string  `expr:"-"`
    Secret   string
}
```

`Internal` is excluded. `Secret` remains available as `Secret`. A `json` tag does **not** create an alias. Unexported fields are ignored. The TypeChecker uses the same rules as runtime environment conversion.

### 1.4. The result is a value, not necessarily a bool

HyperEx is dynamically typed. Arithmetic, collection operations, function calls, and operators can produce different kinds at runtime. Logical operators often preserve an operand rather than manufacturing a boolean:

```hyperex
"hello" && "world"
```

```text
"world"
```

```hyperex
0 || 42
```

```text
42
```

Use `EvalBool` only when that conversion is what you want. Do not assume every predicate is internally represented as a Go `bool` before conversion.

---

## 2. Values, truthiness, equality, and ordering

### 2.1. Runtime kinds

| Kind | `type(x)` | Runtime representation and notes |
|---|---|---|
| `nil` | `"nil"` | Absence of a value |
| `bool` | `"bool"` | Go `bool` |
| `int` | `"int"` | Signed 64-bit integer |
| `float` | `"float"` | IEEE-754 64-bit floating-point value |
| `string` | `"string"` | UTF-8 text; indexing and length are rune-based |
| `array` | `"array"` | `[]any` |
| `map` | `"map"` | `map[string]any` |
| `function` | `"function"` | Lambda closure, built-in, or host function |
| `time` | `"time"` | UTC instant stored as Unix nanoseconds |
| `duration` | `"duration"` | Signed 64-bit nanoseconds |
| `any` | host type name or `"nil"` | External Go value; restricted in Secure mode |

`time.Time` and `time.Duration` have dedicated kinds. A `time.Time` is normalized to UTC and loses its monotonic component. The representable instant range is the signed `int64` Unix-nanosecond range; do not pass host times outside it and expect lossless round trips.

### 2.2. Truthiness is used in more than `if`

Conditions in `&&`, `||`, `!`/`not`, and ternaries use truthiness rather than requiring a boolean:

| Kind | Falsy when | Examples of falsy values |
|---|---|---|
| `nil` | always | `nil` |
| `bool` | false | `false` |
| `int` | zero | `0` |
| `float` | zero | `0.0`, `-0.0` |
| `string` | empty | `""` |
| `array` / `map` | empty | `[]`, `{}` |
| `time` / `duration` | stored bits are zero | Unix epoch, zero duration |
| `function` / `any` | underlying pointer is nil | typed nil where preserved |

All other values are truthy. In particular, `NaN` is truthy, `0.0` is falsy, and an empty array is falsy while `[nil]` is truthy.

Examples:

```hyperex
!0
```

```text
true
```

```hyperex
!1
```

```text
false
```

```hyperex
"" ? "empty" : "not empty"
```

```text
"empty"
```

```hyperex
[] ? 1 : 2
```

```text
2
```

This is deliberate language behavior, not a type-checker bug.

### 2.3. `&&` and `||` return operands

Both operators short-circuit. Their result is the last evaluated operand, not a coerced boolean:

| Expression | Result | Reason |
|---|---|---|
| `"hi" && "bye"` | `"bye"` | Left is truthy; evaluate right |
| `0 && missing` | `0` | Right is never evaluated |
| `5 || 0` | `5` | Left is truthy |
| `false || "fallback"` | `"fallback"` | Left is falsy |
| `nil || "x"` | `"x"` | Nil is falsy |
| `true && 1` | `1` | Result is an int, not `true` |

`and` and `or` are exact aliases for `&&` and `||`. In contrast, `!` and `not` always produce a boolean.

Do not write mathematical comparison chains as `1 < 2 < 3`. Comparisons are left-associative, so the first comparison produces a boolean and the second compares that boolean with `3`. Write:

```hyperex
1 < 2 && 2 < 3
```

### 2.4. Equality: kind, bits, structure, identity

`==` and `!=` use the following rules:

1. Different kinds are unequal, except two numeric values, which are compared as floats.
2. `nil` equals `nil`.
3. `bool`, `int`, `float`, `time`, and `duration` compare their stored bits.
4. Strings compare contents.
5. Arrays and maps compare deeply and structurally, with cycle detection and a recursion-depth guard.
6. Functions and external values use identity-oriented rules rather than structural comparison.

Consequences worth remembering:

- `1 == 1.0` is true, but conversion of very large integers to float can lose precision.
- `0.0 == -0.0` is **false**: their bits differ.
- Two NaNs with the same bit pattern are equal; NaN does not follow the usual IEEE `NaN != NaN` rule here.
- `[1, {a: 2}] == [1, {a: 2}]` is true.
- A cyclic map compared with itself is true and does not recurse forever.

If you need ordering rather than equality, read the next section; it has different NaN behavior.

### 2.5. Ordering and incomparable values

`Compare` is used by `<`, `<=`, `>`, `>=`, `between`, `min`, `max`, `sort`, and `sortBy`:

- Two numbers are compared as floats.
- Two times compare their Unix nanoseconds.
- Two durations compare nanoseconds.
- Two strings compare lexicographically.
- Otherwise, equal values return `0`; unequal or incomparable values return `-1` as an arbitrary fallback.

NaN is neither less nor greater than another numeric value, so `Compare(NaN, x)` returns `0`. Therefore `NaN <= x` and `NaN >= x` are true, while `NaN < x` and `NaN > x` are false. This is unusual enough that non-finite inputs should normally be rejected at the host boundary when ordering matters.

The incomparable fallback is not a total ordering. Sorting arrays with arbitrary mixed kinds can therefore be surprising. Prefer homogeneous arrays or a `sortBy` key with a consistent kind.

---

## 3. Lexical structure: how source text becomes tokens

### 3.1. Whitespace and comments

Spaces, tabs, carriage returns, newlines, and other Unicode whitespace are ignored between tokens.

```hyperex
// A line comment through the end of the line
1 + 2 /* a block comment */ * 3
```

Block comments do not nest. An unterminated block comment is an error. Comments are omitted during parsing but retained by the public `Tokenize` API for editors and highlighters.

### 3.2. Identifiers

An identifier starts with `_`, `$`, `#`, or a Unicode letter. Subsequent characters may also include Unicode digits.

Valid names include `x`, `_tmp`, `$0`, `#`, `price2`, `пользователь`, and `世界`.

Reserved words are:

```text
true false nil in not and or let if else
```

`if` and `else` are reserved but have no production in the language. They should not be used as variable names.

### 3.3. Integer literals

| Form | Example | Meaning |
|---|---|---|
| Decimal | `42` | 42 |
| Leading-zero octal | `012` | **10**, not decimal 12 |
| Explicit octal | `0o777`, `0O10` | 511, 8 |
| Hexadecimal | `0xFF`, `0Xff` | 255 |
| Binary | `0b1010`, `0B10` | 10, 2 |

Integers are parsed as signed 64-bit values. `0x` alone is rejected by the parser. To express `math.MinInt64`, use arithmetic rather than an oversized positive literal:

```hyperex
-9223372036854775807 - 1
```

### 3.4. Floating-point literals

```hyperex
3.14
.5
1e10
1E-5
```

A dot followed by a digit starts a float, so `.5` is `0.5`. A range is different: `1..2` contains the `..` operator rather than a decimal point.

`1e` is not a valid complete literal. NaN and infinity are not source keywords; they can be produced by arithmetic, host values, or built-ins.

### 3.5. Strings and escapes

| Delimiter | Escapes | Literal newline |
|---|---|---|
| `"double"` | supported | rejected as unterminated |
| `'single'` | supported | rejected as unterminated |
| `` `raw` `` | no escapes; backslash is literal | allowed |

Recognized escapes are `\n`, `\r`, `\t`, `\\`, `\"`, `\'`, `\0`, and `\uXXXX` with exactly four hexadecimal digits. An unknown escape keeps the escaped character and discards the backslash. A trailing incomplete escape is an error.

```hyperex
"line1\nline2"
```

```text
line1
line2
```

```hyperex
`C:\raw\path`
```

```text
C:\raw\path
```

String length, indexing, slicing, `substr`, and `indexOf` operate on Unicode code points (runes), not bytes:

```hyperex
len("🎉🎊🎁")
```

```text
3
```

```hyperex
"Привет"[0]
```

```text
"П"
```

---

## 4. Operators and precedence

### 4.1. Complete precedence table

Higher numbers bind more tightly. Parentheses are the clearest way to communicate intent when the order is not obvious.

| Precedence | Operators | Associativity |
|---|---|---|
| 1 | `or`, `||` | left |
| 2 | `and`, `&&` | left |
| 3 | `|` (pipe or bitwise OR) | left |
| 4 | `^` | left |
| 5 | `&` | left |
| 6 | `==`, `!=` | left |
| 7 | `<`, `<=`, `>`, `>=`, `in`, `not in`, `~=` | left |
| 8 | `<<`, `>>`, `>>>` | left |
| 9 | `+`, `-` | left |
| 10 | `*`, `/`, `%` | left |
| 11 | `**` | right |
| 12 | `??` | right |
| 13 | `..` | left |

Ternary `? :`, unary `-`/`!`/`not`/`~`, and postfix member/index/call operations have dedicated parsing rules. `not in` has exactly the same precedence as `in`.

```hyperex
1 + 2 not in [3]
```

```text
false // (1 + 2) not in [3]
```

`??` binds more tightly than arithmetic operators:

```hyperex
2 + 3 ?? 4
```

```text
5 // 2 + (3 ?? 4)
```

Use parentheses to request the other meaning:

```hyperex
(2 + 3) ?? 4
```

### 4.2. Arithmetic `+`, `-`, `*`, `/`, `%`, `**`

`+` concatenates strings whenever either operand is a string, unless temporal rules take precedence. It also concatenates two arrays and performs numeric addition. Other combinations are errors.

```hyperex
"answer: " + 42
```

```text
"answer: 42"
```

```hyperex
[1, 2] + [3, 4]
```

```text
[1, 2, 3, 4]
```

Integer `+`, `-`, and `*` promote to float on signed 64-bit overflow rather than wrapping. Integer division truncates toward zero. One special Go integer edge is retained: `MinInt64 / -1` yields `MinInt64` rather than a larger positive integer.

```hyperex
10 / 3
```

```text
3
```

```hyperex
10.0 / 3
```

```text
3.3333333333333335
```

Division or modulo by zero returns `ErrDivisionByZero`.

`*` also repeats a string when the left operand is a string and the right is an integer. Non-positive counts produce an empty string. Results are capped at 10 MiB to avoid unbounded allocation.

```hyperex
"ab" * 3
```

```text
"ababab"
```

`**` uses integer exponentiation when it is exact and the exponent is non-negative. Otherwise it uses floating-point `math.Pow`.

### 4.3. Unary operators

- `-x` performs numeric or duration negation. Negating `MinInt64` promotes to float. Negating time is an error.
- `!x` and `not x` return boolean negation by truthiness.
- `~x` performs bitwise NOT after integer conversion.

### 4.4. Bitwise operators and shifts

`&`, `|`, `^`, `~`, `<<`, `>>`, and `>>>` use integer conversion: integers stay exact, floats truncate, booleans become `0` or `1`, and numeric strings are parsed. `>>>` treats the left operand as unsigned.

Negative shift counts are errors. For `<<` and `>>>`, counts of 64 or more produce zero. For signed `>>`, counts of 64 or more produce zero for non-negative values and `-1` for negative values.

### 4.5. Membership `in` and `not in`

- In an array, compare each element using `Equal`.
- In a map, test the stringified key.
- In a string, test substring containment; the empty substring is contained in every string.
- External Go slices, arrays, and maps are handled through reflection unless Secure mode forbids them.

```hyperex
3 in [1, 2, 3]
```

```text
true
```

```hyperex
"ell" in "hello"
```

```text
true
```

```hyperex
"email" not in user
```

```text
true // when user has no email key
```

### 4.6. Regular-expression matching `~=`

The right operand is compiled as a Go RE2 regular expression, cached with a bounded LRU. The left operand is matched as a string. Invalid patterns return `ErrInvalidRegex`.

```hyperex
email ~= "^[a-z]+@[a-z]+\\.[a-z]+$"
```

There are no regex literals.

---

## 5. Collections: arrays, maps, indexing, slicing, and spread

### 5.1. Array literals and concatenation

Array elements are expressions evaluated from left to right:

```hyperex
[1, "two", true, nil]
```

A trailing comma is allowed:

```hyperex
[1, 2, 3,]
```

Use `+` to concatenate two arrays. Use `concat` when you want variadic flattening by one level; non-array arguments become individual elements.

### 5.2. Map literals and computed keys

A map literal uses `{key: value}`. Identifier and string keys become literal string keys. Computed keys require brackets:

```hyperex
{name: "Alice", "age": 30}
```

```hyperex
{[prefix + "name"]: "Alice"}
```

Do not confuse `{[key]: value}` with `{key: value}`: the first evaluates `key`, the second uses the literal string `"key"`.

### 5.3. Indexing and optional indexing

Positive and negative indices are supported for arrays and strings. Negative indices count from the end. Strings index by rune.

```hyperex
arr[-1]
```

```text
last element
```

```hyperex
"🎉🎊🎁"[1]
```

```text
"🎊"
```

A missing map key returns `nil`. An out-of-bounds array or string index returns `ErrIndexOutOfBounds`.

`arr?.[i]` returns `nil` if the object is nil or if indexing fails for any reason. Use it when absence is acceptable and you do not need to distinguish a missing key from an invalid object.

### 5.4. Member access and optional member access

`obj.field` reads a map key or an exported struct field. Missing properties are errors. `obj?.field` returns `nil` for a nil object or missing property.

```hyperex
user?.address?.city ?? "unknown"
```

Optional chaining is useful for sparse data, but it also hides the reason for absence. Validate required fields explicitly when an error is preferable.

### 5.5. Slices

The general form is:

```text
object[low:high:step]
```

Low is inclusive, high is exclusive, and step defaults to `1`. Open bounds are allowed. Negative indices count from the end. A negative step walks backward and changes the default bounds.

```hyperex
arr[1:3]
```

```text
[second, third]
```

```hyperex
arr[::2]
```

```text
[first, third, fifth, ...]
```

```hyperex
arr[::-1]
```

```text
reversed array
```

A zero step returns `ErrInvalidSliceIndex`. Slicing a string produces another string, still measured in runes.

### 5.6. Spread in array literals

Spread is valid only inside an array literal:

```hyperex
[0, ...middle, 100]
```

It concatenates arrays at that position. Spread is not supported in map literals or function calls.

---

## 6. Bindings, lexical scope, lambdas, and pipes

### 6.1. `let` is an expression, not a statement

The canonical form is:

```hyperex
let name = value; body
```

The binding is visible in `body`, and the result is the result of `body`:

```hyperex
let price = 100; let quantity = 3; price * quantity
```

```text
300
```

The body may be omitted. In that case the binding value is evaluated **once** and becomes the result:

```hyperex
let x = 5
```

```text
5
```

A binding is evaluated even if its name is never used. This matters for host calls and errors.

### 6.2. Nested bindings have lexical scope

An inner `let` shadows the outer name only inside its body. After the body finishes, the previous binding is restored:

```hyperex
let x = 1; (let x = 2; x) + x
```

```text
3 // inner x is 2; outer x is restored to 1
```

Destructuring follows the same rule. This also means Studio transformations must respect shadowing; simply replacing every same-named identifier is incorrect.

### 6.3. Object and array destructuring

Object destructuring supports aliases and defaults:

```hyperex
let {name, role = "guest", age: years} = person; years
```

Array destructuring binds by position:

```hyperex
let [first, second, third = 0] = arr; third
```

A missing element or property is `nil`, and a default is used only when that value is nil. `_` is an ordinary identifier, not a special skip marker.

Without a semicolon, the source value is returned once, just as with simple `let`.

### 6.4. Lambdas and closures

Lambda forms are:

```hyperex
x => x * 2
```

```hyperex
(acc, item) => acc + item
```

```hyperex
() => 42
```

Closures capture a copy of the current environment. Nested lambdas work naturally:

```hyperex
let multiply = x => y => x * y;
let double = multiply(2);
double(5)
```

```text
10
```

### 6.5. `#` and `$n` are names inside a lambda body

`#` means the first argument. `$0`, `$1`, and so on mean arguments by position. They are available only during lambda execution.

Crucially, **`# + 1` by itself is not a lambda**. This is invalid as a higher-order callback:

```hyperex
map([1, 2], # + 1) // wrong: # is looked up as an ordinary name
```

Write an explicit lambda instead:

```hyperex
map([1, 2], x => x + 1)
```

Or deliberately use the special names as lambda parameters:

```hyperex
map([1, 2], # => # + 1)
```

```hyperex
map([1, 2], $0 => $0 + 1)
```

Higher-order built-ins pass both the element and its index, so `(x, i) => ...` and `$0/$1` can be used where appropriate.

### 6.6. Pipe inserts the left value into a call

For a named function, `value | name` becomes `name(value)`:

```hyperex
[1, 2, 3] | sum
```

```text
6
```

For a call, the left value is prepended to the argument list:

```hyperex
[1, 2, 3, 4] | take(2)
```

```text
[1, 2]
```

Pipes compose left to right:

```hyperex
users | filter(u => u.active) | pluck("name") | sort
```

### 6.7. Why `|` needs disambiguation

The same token also means bitwise OR. The parser chooses using AST shape:

1. An integer or float literal on either side means bitwise OR, never pipe.
2. Array literals, calls, members, indexes, slices, pipes, ranges, or strings on the left, followed by an identifier or call, select pipe.
3. An identifier on the left and a call on the right select pipe.
4. An identifier on the left and a default built-in name on the right select pipe.
5. Otherwise, two identifiers select bitwise OR.

| Expression | Meaning |
|---|---|
| `[1,2,3] \| sum` | pipe |
| `arr \| sum` | pipe because `sum` is a built-in |
| `data \| filter(x => x > 0)` | pipe |
| `x \| y` | bitwise OR |
| `flags \| MASK` | bitwise OR |
| `1 \| 2` | bitwise OR |
| `a.b \| f` | pipe |

To force pipe between arbitrary identifiers, use an explicit call: `x | transform()`. To force bitwise OR with built-in-looking names, introduce neutral intermediate names:

```hyperex
let a = xs; let b = mask; a | b
```

`Analyze` reports `WarnAmbiguousBitwiseOr` for ambiguous identifier/identifier bitwise usage and `WarnBuiltinNamePipe` when the built-in-name rule selected pipe. These warnings are part of Studio support, not evaluation errors.

---

## 7. Built-in functions: a guided tour

The engine exposes 100 built-in names. `BuiltinNames()` returns a sorted copy and `IsBuiltin(name)` checks one name. There is no public `DefaultBuiltins` variable.

### 7.1. Math

| Name | Example | Result and edge behavior |
|---|---|---|
| `abs` | `abs(-42)` | `42`; `MinInt64` becomes float |
| `min` | `min(5, 2, 8)` or `min([5, 2, 8])` | `2`; empty array gives nil |
| `max` | `max(5, 2, 8)` or `max([5, 2, 8])` | `8`; empty array gives nil |
| `ceil` | `ceil(3.2)` | `4`; NaN/Inf/out of int64 stays float |
| `floor` | `floor(3.8)` | `3`; same non-finite rule |
| `round` | `round(3.5)` | `4`; same non-finite rule |
| `sqrt` | `sqrt(16)` | `4.0`; negative input gives NaN by default |
| `pow` | `pow(2, 8)` | `256.0`; always float |
| `log` | `log(2.718281828)` | approximately `1.0` |
| `exp` | `exp(1)` | approximately `2.718` |
| `sin` | `sin(0)` | `0.0` |
| `cos` | `cos(0)` | `1.0` |

`min` and `max` require at least one argument. Their comparisons inherit the NaN and incomparable rules from §2.5.

### 7.2. Strings and length

| Name | Example | Result and edge behavior |
|---|---|---|
| `len` | `len("🎉🎊🎁")` | `3` runes; arrays/maps use element count; other kinds give `0` |
| `upper` | `upper("Hello")` | `"HELLO"`; input is stringified |
| `lower` | `lower("Hello")` | `"hello"`; input is stringified |
| `trim` | `trim("  hi  ")` | `"hi"` |
| `trimLeft` | `trimLeft("xxhi", "x")` | `"hi"`; optional cutset, default whitespace |
| `trimRight` | `trimRight("hixx", "x")` | `"hi"`; same defaults |
| `split` | `split("a,b,c", ",")` | `["a", "b", "c"]` |
| `join` | `join(["a", "b"], "-")` | `"a-b"`; array required |
| `contains` | `contains("hello", "ell")` | `true`; string containment via `strings.Contains` |
| `startsWith` | `startsWith("hello", "he")` | `true` |
| `endsWith` | `endsWith("hello", "lo")` | `true` |
| `replace` | `replace("hello", "l", "L")` | `"heLLo"` |
| `substr` | `substr("hello", 1, 3)` | `"ell"`; rune-based, negative start from end |
| `indexOf` | `indexOf("hello", "l")` | `2`; first rune index or `-1` |

`contains` is not the `in` operator: it is a built-in string helper with stringification. `in` also handles arrays and maps.

### 7.3. Arrays and structure

| Name | Example | Result and edge behavior |
|---|---|---|
| `first` | `first([1, 2])` | `1`; empty gives nil |
| `last` | `last([1, 2])` | `2`; empty gives nil |
| `take` | `take([1, 2, 3], 2)` | `[1, 2]`; count clamped to `[0, len]` |
| `skip` | `skip([1, 2, 3], 1)` | `[2, 3]`; count clamped similarly |
| `reverse` | `reverse([1, 2, 3])` | `[3, 2, 1]`; new array |
| `sort` | `sort([3, 1, 2])` | `[1, 2, 3]`; uses `Compare` |
| `unique` | `unique([1, 2, 1, 3])` | `[1, 2, 3]`; typed, cycle-safe structural keys |
| `concat` | `concat([1], [2, 3], 4)` | `[1, 2, 3, 4]`; one-level flattening |
| `flatten` | `flatten([[1], [2, [3]]])` | `[1, 2, 3]`; recursive, maximum depth 100 |

`unique` distinguishes typed representations and does not concatenate ambiguous textual keys. It should not be confused with `distinct` or set ordering; it preserves first occurrence order.

### 7.4. Aggregation

`sum` keeps integer accumulation until overflow, then continues as float. `avg` returns float and gives nil for an empty array. `count` uses the same length rules as `len`.

`all`, `any`, `none`, and `one` have two forms:

```hyperex
all([true, 1, "ok"])       // truthiness of elements
all([2, 4, 6], x => x > 0) // predicate form
```

| Function | Empty input | Meaning |
|---|---|---|
| `all` | `true` | Every element/predicate is truthy |
| `any` | `false` | At least one is truthy |
| `none` | `true` | None is truthy |
| `one` | `false` | Exactly one is truthy |

Predicates receive `(element, index)` and must be function values. Wrong predicate types return `ErrInvalidType`.

### 7.5. Higher-order functions

| Name | Callback | Behavior |
|---|---|---|
| `map` | `(elem, index)` | Transform every element |
| `filter` | `(elem, index)` | Keep truthy matches |
| `reduce` | `(acc, elem, index)` | Fold left; empty without initial value gives nil |
| `find` | `(elem, index)` | First match or nil |
| `findIndex` | `(elem, index)` | First matching index or `-1` |

Examples:

```hyperex
map([1, 2, 3], x => x * x)
```

```text
[1, 4, 9]
```

```hyperex
reduce([1, 2, 3, 4], (acc, x) => acc + x, 0)
```

```text
10
```

Lambda execution shares the parent context, operation budget, and call-depth policy. Cancellation therefore also applies inside callbacks.

### 7.6. Type conversion

| Name | Behavior |
|---|---|
| `int` | `TryInt`: numeric strings, floats, and bools convert; invalid input errors |
| `float` | `TryFloat`: similar conversion to float |
| `string` | Human-readable string form with bounded composite formatting and cycle markers |
| `bool` | Truthiness conversion |
| `type` | Runtime kind name |
| `isNil` | True only for the nil kind |

A float-to-int conversion truncates. Non-finite or out-of-range conversions can be rejected by stricter numeric paths; do not rely on host-specific float-to-int behavior.

---

## 8. Collection utilities, JSON, and deterministic output

### 8.1. Repetition and generation

- `repeat(value, n)` returns an **array** containing `value` `n` times. Negative counts become zero; the maximum is one million.
- `range(start, end[, step])` returns an inclusive numeric range. Positive steps ascend, negative steps descend, and zero step is an error. The maximum length is one million.
- `zip(a, b, ...)` returns tuples truncated to the shortest input.
- `chunk(array, size)` returns chunks of at most `size` elements. Size must be positive; allocation arithmetic is checked for extremes.

```hyperex
repeat("ab", 3)
```

```text
["ab", "ab", "ab"]
```

```hyperex
range(5, 1, -2)
```

```text
[5, 3, 1]
```

```hyperex
chunk([1, 2, 3, 4, 5], 2)
```

```text
[[1, 2], [3, 4], [5]]
```

### 8.2. Filtering and extracting

- `compact(array)` removes only nil values.
- `truthy(array)` removes all falsy values.
- `pluck(array, key)` extracts that key from map elements and skips elements where it is absent.
- `keys(map)` returns sorted string keys.
- `values(map)` returns values in the same sorted key order, making `zip(keys(m), values(m))` deterministic.

### 8.3. Grouping and sorting

`groupBy(array, keyFn)` calls `keyFn(element)`, converts the result to a string key, and returns groups in first-seen order:

```hyperex
groupBy(users, u => u.role)
```

Each result has the form `{key: "role", items: [...]}`.

`sortBy(array, keyFn[, desc])` computes one key per element, then sorts with `Compare`. Descending order still returns false for equal keys in both comparison directions. It does not promise a stable sort.

### 8.4. Scalar utilities

- `clamp(v, min, max)` keeps an all-int path or uses float.
- `between(v, min, max)` is inclusive on both ends.
- `hasKey(map, key)` tests key presence.
- `empty(x)` is true only for nil, empty string, empty array, or empty map. Zero and `false` are not “empty”.
- `notEmpty(x)` negates `empty`.
- `default(v, fallback)` returns fallback only when `v` is nil.
- `coalesce(...)` returns the first non-nil argument or nil.
- `print(...)` prints and returns nil. It is normally denied in secure presets.
- `sprintf(format, ...)` uses Go formatting.

### 8.5. JSON

`toJSON(value)` uses `encoding/json` marshaling. Unsupported values and non-finite floats return errors. Map key output follows Go JSON ordering.

`fromJSON(text)` parses JSON and then converts whole numeric values to `int64` only when `|n| <= 2^53-1`, the exact-integer range of float64. Larger values stay float to avoid inventing precise integers that were already lost during parsing.

```hyperex
fromJSON("{\"x\":42}")
```

```text
{"x": 42}
```

---

## 9. Time, duration, and process quality

### 9.1. Instants and durations

There is no `now()` built-in. The host must provide wall clock explicitly, which keeps expressions deterministic and testable.

| Name | Purpose |
|---|---|
| `timeUTC(y, m, d, h, min, s)` | Construct a UTC instant |
| `timeUnix(sec)` | Construct from Unix seconds |
| `timeUnixNano(ns)` | Construct from Unix nanoseconds |
| `unix(t)` / `unixNano(t)` | Extract integer seconds/nanoseconds |
| `timeYear`, `timeMonth`, `timeDay` | UTC calendar fields |
| `timeHour`, `timeMinute`, `timeSecond` | UTC clock fields |
| `timeWeekday(t)` | Sunday `0` through Saturday `6` |
| `timeAdd(t, d)` | Add a duration |
| `durationSeconds`, `durationMs`, `durationNs` | Construct a duration |
| `durSeconds(d)` | Floating-point seconds |
| `durMs(d)` / `durNs(d)` | Integer milliseconds/nanoseconds |

All calendar fields are UTC. Time zones are not preserved. `timeUTC` follows Go `time.Date` normalization rather than performing a separate calendar-validation layer.

Temporal arithmetic:

- `time + duration` and `duration + time` produce time.
- `time - duration` produces time.
- `time - time` produces duration.
- Durations support `+`, `-`, `* number`, `/ number`, and unary `-`.
- Adding two times or negating time is an error.
- Duration overflow is checked and returns an error instead of wrapping.

### 9.2. Quality maps `{v, q}`

Process values can be represented as a map with value and quality code:

```go
env["temp"] = map[string]any{"v": 12.5, "q": float64(0)}
```

The quality class uses the OPC UA–compatible top two bits:

- Good: `(code & 0xC0000000) == 0`
- Uncertain: `== 0x40000000`
- Bad: `== 0x80000000`

| Name | Behavior |
|---|---|
| `v(x)` | Extract `x["v"]`; primitive values pass through; nil gives nil |
| `q(x)` | Extract numeric `x["q"]`; a bare number is itself a qcode |
| `isGood(x)` | Test Good class on a map or bare qcode |
| `isBad(x)` | Test Bad class |
| `isUncertain(x)` | Test Uncertain class |
| `withQ(value, qcode)` | Build `{v: value, q: qcode}` |

Quality maps are not numbers and are not automatically unwrapped:

```hyperex
temp + 1 // wrong when temp is {v, q}
```

Use:

```hyperex
isGood(temp) ? v(temp) + 1 : v(backup)
```

Quality codes must be finite and fit `uint32`. Negative and out-of-range codes are rejected.

---

## 10. Numeric semantics in depth

### 10.1. Default numeric mode

`NumericDefault` preserves integers where possible:

- `int + int`, `int - int`, and `int * int` return int unless signed 64-bit overflow occurs, in which case they return float.
- Integer `/` truncates toward zero; mixed numeric arithmetic returns float.
- `%` supports integer remainder and `math.Mod` for other numeric forms.
- `**` tries exact non-negative integer exponentiation before using float.
- `sum` similarly starts with integers and switches to float on overflow or a non-integer input.

```hyperex
1 / 2
```

```text
0 // int
```

### 10.2. Float-first numeric mode

`WithNumericMode(NumericFloat64)` makes numeric `+`, `-`, `*`, `/`, `%`, `**`, and unary `-` return float. String concatenation and temporal arithmetic are unchanged.

```hyperex
1 / 2
```

```text
0.5
```

The TypeChecker uses the engine's numeric mode when inferring types. Optimizer transformations must preserve the same type and error behavior as the unoptimized VM.

### 10.3. Non-finite results

`NonFiniteAllow` permits NaN and infinity. `NonFiniteError` rejects non-finite float results and non-finite values produced by arithmetic or built-ins before successful return. `sqrt(-1)`, overflowing float arithmetic, and final typed float conversion are covered.

The check is **not a recursive scan** of every value nested inside arrays and maps. If nested non-finite data matters, validate it at the host boundary or in domain code.

Constant folding must not hide this policy: an intermediate infinity cannot be folded into a final `true` or `false` to bypass `NonFiniteError`.

### 10.4. Strict conversion mode

`WithStrictMode(true)` makes invalid conversions return `ErrInvalidConversion` instead of silently using zero values. `TryBool` and `TryString` accept only their exact kinds. `TryInt` and `TryFloat` allow documented numeric conversions.

Strict mode is not a complete static type system and does not forbid string concatenation. For example, string `+` remains string concatenation when either operand is a string.

### 10.5. Optimizer equivalence

Optimization is enabled by default and includes constant folding, strength reduction, algebraic simplification, and dead-code elimination. `Unoptimized()` disables optimization for one compilation. `WithConstantFolding(false)` disables only literal folding while leaving other optimization passes enabled.

These options are useful for debugging and differential testing, not as a promise that optimized and unoptimized operation counts or timing will match. Semantic results, error classes, and numeric types are expected to agree.

---

## 11. Host options, security, and resource limits

### 11.1. The trust boundary

Secure mode is a best-effort execution policy, **not operating-system isolation**. Do not run arbitrary native code inside a HyperEx sandbox and expect process-level containment.

`WithFunction` registers trusted Go code. It can perform I/O, allocate, panic recovery notwithstanding, and consume time outside the VM's operation budget. Prefer `WithSafeFunction` or `WithSafeFunctionMeta` when host work should carry an explicit operation tax.

SafeFunction metadata does not measure wall-clock work. `MaxOwnOps` is a declared tax, and `Pure` is documentation only.

### 11.2. Sandbox limits

| Setting | Meaning and caveat |
|---|---|
| `MaxOps` | Aborts after an approximate operation budget; loop accounting can be batched |
| `Timeout` | Wall-clock execution deadline; independent of context deadline |
| `MaxMemory` | Approximate, periodically estimated memory; not a hard heap-allocation cap |
| `MaxStackDepth` | Bounds VM operand stack growth |
| `MaxCallDepth` | Bounds nested function/lambda calls; sandbox value takes precedence over engine default |
| `AllowedFuncs` | Name allowlist; if configured, other names are denied |
| `DeniedFuncs` | Name denylist and takes precedence over allowlist |
| `ReadOnly` | Rejects `let` and destructuring with `ErrAssignmentNotAllowed` |
| `DisableBuiltins` | Starts with an empty default built-in set; custom functions can still be registered |

Both `..` and `range(...)` participate in operation and cancellation checks. Higher-order callbacks inherit the same cumulative budget rather than receiving a fresh one per element.

### 11.3. Secure environment conversion

Secure accepts JSON-like values plus `time.Time` and `time.Duration`. It recursively validates arrays and maps, including values already wrapped with `hyperex.Array` or `hyperex.Map`.

It rejects arbitrary structs, pointers, functions, channels, and other external Go objects. It also forbids reflection-based member access, indexing, slicing, membership testing, and calls to unnamed Go functions.

Environment conversion itself can be expensive and is only periodically budgeted. Do not treat Secure as a guarantee that every allocation happens under `MaxMemory`.

### 11.4. Context and timeouts

Every evaluation accepts a context. Nil context means `context.Background()`, although an explicit context is preferable. `ErrOperationCancelled` is used for context cancellation; `ErrTimeout` is used for the sandbox deadline. Whichever is observed first wins.

Checks happen on VM entry, during bytecode execution, inside loops, and after host callbacks. A trusted host function that never returns still cannot be forcibly killed by context; use process-level isolation when that is required.

### 11.5. Compilation and evaluation options

- `WithMaxSourceBytes(n)` rejects oversized source with `ErrSourceTooLarge`; `n <= 0` means unlimited.
- `WithMaxCallDepth(n)` changes the engine default, with a sandbox override when explicitly configured.
- `WithOperator(op, fnName)` compiles a binary operator as a host-function call. Short-circuit `&&` and `||` are not overloaded this way.
- `WithTypeCheck(env)` enables best-effort compile-time checks. Unknown or `any` types remain permissive for dynamic environments; it is not a proof of runtime success.
- `AsBool`, `AsInt`, `AsFloat`, and `AsString` are compile hints converted on every evaluation. `ResultHint()` exposes the selected hint.
- `Program.Profile` evaluates with the same Secure policy and result conversion as normal evaluation, while recording wall time, operation count, and maximum stack depth.

### 11.6. Presets

`NewSecureCalc` and `NewSecureAuto` assemble common policies: Secure, operation and timeout limits, restricted call depth, read-only evaluation, denied `print`, `NumericFloat64`, and `NonFiniteError`. `NewSecureAuto` uses tighter budgets and also denies JSON conversion functions. Extra options are applied afterward and can deliberately replace parts of a preset, so review the final option order.

---

## 12. Studio and source-level tools

Studio APIs help editors and linters; they are not additional language syntax.

### 12.1. `Tokenize` and `Parse`

`Tokenize` returns stable public token kinds: `Ident`, `Number`, `String`, `Operator`, `Keyword`, `Punct`, `Comment`, `EOF`, and `Illegal`. Tokens include byte offset, line, and column.

`Parse` returns an opaque `Node`. Concrete AST node types remain private. Use `Walk`, `Transform`, and `ASTToString` rather than depending on internal node layouts.

### 12.2. `Analyze`

`Analyze` reports identifiers, called functions, operators, AST depth, estimated complexity, assignments, optional chaining, coalescing, and warnings. Complexity and estimated cost are heuristics, not runtime proof.

`IsPure` only models known side-effecting built-ins. It does not prove that an arbitrary custom function is pure. Identifier lists can include function names and binding names; use `ExtractMemberRoots` when you specifically want `Alarm` from `Alarm.Fire`.

`ProfileExpr` and `Program.Profile` measure actual wall time and VM operations for one evaluation.

### 12.3. `Simplify`

`Simplify` parses, applies literal and algebraic rewrites, and prints an expression. It does not call host functions. It avoids emitting non-finite source literals that cannot be parsed back; where necessary, the original arithmetic form is retained.

`ASTToString` handles computed map keys, postfix precedence, lambdas, nested unary expressions, integer extremes, float type preservation, and non-finite expressions. Round-trip tests should compare semantics, not only whether parsing succeeds.

### 12.4. `Inline`

`Inline` substitutes only closed literal bindings: scalar literals and arrays/maps containing only closed literals. Other bindings remain in place. This deliberately avoids:

- duplicated host calls;
- capture by lambda parameters;
- removal of mandatory errors;
- changed behavior from free identifiers;
- incorrect shadowing by nested `let` or destructuring.

It is a conservative source transformation, not a general evaluator.

### 12.5. `Expand`

`Expand` substitutes named macros with parsed AST definitions. It respects lexical bindings and does not replace non-computed property names after `.`. Macro replacement is a one-pass expansion rather than a recursive interpreter, and replacements are copied to avoid sharing mutable AST nodes.

### 12.6. Type checking and validation

`Validate` checks syntax. `ValidateWithEnv` adds best-effort environment types. `CheckExpressionType` compares inferred and expected types. Function signatures include arity, parameter types, variadic arguments, and numeric widening from int to float. Pipe calls include the inserted left argument during checking.

The checker accepts truthiness and dynamic `any`/`unknown` values. It does not prove that an unknown variable exists at runtime or that a dynamic callback is side-effect free.

---

## 13. Errors and failure handling

### 13.1. Error families

Use `errors.Is` for sentinels and `errors.As` for structured error types:

- `LexerError` and `ParseError`: source location and formatted context.
- `CompileError`: AST-to-bytecode failure.
- `TypeError` and `TypeCheckErrors`: static type diagnostics; multiple errors expose the first through `Unwrap`.
- `RuntimeError`: evaluation failure, optional source, expected/got types, and an underlying cause.
- `UndefinedError`: missing variable or function with name and kind.
- `SandboxError`: limit or access violation with `Type`, `Limit`, `Actual`, and a cause.

`RuntimeError.Offset`, `Line`, and `Column` are zero when unknown. Current AST nodes do not carry full runtime source maps, so runtime locations are often unknown.

### 13.2. Common sentinels

| Sentinel | Typical cause |
|---|---|
| `ErrDivisionByZero` | `/` or `%` by zero |
| `ErrIndexOutOfBounds` | Invalid array/string index |
| `ErrNotCallable` | Calling a non-function |
| `ErrArgumentCount` | Wrong arity, including pipe-adjusted calls |
| `ErrInvalidType` | Wrong built-in argument kind |
| `ErrNotIndexable` | Indexing an unsupported kind |
| `ErrInvalidSliceIndex` | Zero step or invalid bounds |
| `ErrInvalidRegex` | Invalid RE2 pattern |
| `ErrMaxCallDepth` | Nested calls exceed policy |
| `ErrOperationCancelled` | Context cancellation/deadline |
| `ErrInvalidConversion` | Strict conversion failure |
| `ErrNonFinite` | NaN/Inf under `NonFiniteError` |
| `ErrSourceTooLarge` | Source exceeds configured byte limit |
| `ErrOpsLimitExceeded` | Operation budget exhausted |
| `ErrMemoryLimitExceeded` | Estimated memory exceeds policy |
| `ErrTimeout` | Sandbox deadline elapsed |
| `ErrFunctionNotAllowed` | Allow/deny or Secure function restriction |
| `ErrAssignmentNotAllowed` | `let` under read-only policy |
| `ErrStackOverflow` | VM operand stack exceeds policy |

Do not parse error strings when a sentinel or structured type is available. Panics from trusted host callbacks become `*RuntimeError`; `MustCompile` and `MustEval` panic by contract and should be used only where failure is a programmer error.

---

## 14. Practical recipes

### 14.1. Validation

```hyperex
let name = form.name ?? "";
let email = form.email ?? "";
let age = form.age ?? 0;
let errors = [];
let errors = len(name) < 2 ? [...errors, "Name too short"] : errors;
let errors = !(email ~= "@") ? [...errors, "Invalid email"] : errors;
let errors = age < 18 ? [...errors, "Must be 18+"] : errors;
{"valid": len(errors) == 0, "errors": errors}
```

Each nested binding can shadow the previous `errors` while its initializer still sees the preceding value.

### 14.2. Aggregation by key

```hyperex
let regions = unique(sales | pluck("region"));
regions | map(r => {
  "region": r,
  "total": sum(sales | filter(s => s.region == r) | pluck("amount")),
  "count": len(sales | filter(s => s.region == r))
})
```

For complex groups, `groupBy` is often clearer:

```hyperex
groupBy(sales, s => s.region) | map(g => {
  "region": g.key,
  "total": sum(g.items | pluck("amount")),
  "count": len(g.items)
})
```

### 14.3. SCADA quality selection

Two supported host shapes are a separate qcode channel and a map-shaped process value:

```text
temp = 12.5
temp_q = 0
```

```text
temp = {"v": 12.5, "q": 0}
```

Formulas:

```hyperex
isGood(temp_q) && temp > 10
```

```hyperex
isGood(temp) ? v(temp) : v(backup)
```

Use Secure presets for operator-authored formulas and pass time explicitly.

### 14.4. Hot-path evaluation

1. Compile at configuration load.
2. Reuse one `Program` concurrently.
3. Reuse a pre-sized `map[string]any` and update values in place.
4. Prefer `EvalValue` plus `TryFloat`, or `EvalFloat`, when the result is numeric.
5. Avoid unnecessary higher-order callbacks and reflection-heavy struct environments per sample.
6. Size operation and timeout budgets with headroom for real data.

Runnable example: [examples/scada_hotpath/](examples/scada_hotpath/). Related: [examples/scada_secure/](examples/scada_secure/) and [examples/scada_quality/](examples/scada_quality/).

### 14.5. Debugging with Studio

Use `Tokenize` for highlighting, `Analyze` for names and pipe warnings, `Simplify` for conservative literal rewrites, and `Profile` for operation counts and stack depth. If optimized behavior needs investigation, compare the same expression with `Unoptimized()` rather than relying on timing alone.

---

## 15. Deliberate non-goals and frequent mistakes

| Missing feature | What to use instead |
|---|---|
| Statement blocks | One expression with nested `let` |
| `if`, `while`, `for`, `return` statements | Ternary, `??`, `&&`, `||`, higher-order functions |
| Loop statements | `map`, `filter`, `reduce`, `range`, `..` |
| General assignment `=` | Lexically scoped `let` and destructuring |
| Unary `+`, `++`, `--`, `->` | Arithmetic negation only where defined; lambdas use `=>` |
| Optional call `fn?.()` | Check the value before calling |
| Spread in maps or calls | Spread only in array literals |
| Regex literals | `~=` with a string pattern |
| Modules, classes, imports | Host application structure |
| Multiple expressions in one unit | One expression through EOF |

Common mistakes and their precise remedies:

1. **`# + 1` is not a lambda.** Write `x => x + 1`, `# => # + 1`, or `$0 => $0 + 1`.
2. **`&&`/`||` do not force bool.** Use `!` or an explicit comparison when a boolean is required.
3. **`??` binds tightly.** `a + b ?? c` is `a + (b ?? c)`.
4. **`012` is octal.** It means `10`, not decimal `12`.
5. **Zero is not nil.** `0 ?? 42` returns `0`; `empty(0)` is false.
6. **Float equality is bitwise.** `0.0 != -0.0`; NaN equality and ordering have the special rules in §2.
7. **Comparison is not a mathematical chain.** Use `a < b && b < c`.
8. **Secure is not process isolation.** Keep native callbacks outside the untrusted boundary.
9. **SafeFunction tax is declared, not measured.** It does not bound callback wall time.
10. **A quality map is not a number.** Use `v(temp)` for arithmetic and `q(temp)` for the code.
11. **Map output order depends on the operation.** `keys`/`values` are sorted; other representations should not be relied upon for display ordering.
12. **MinInt64 needs an expression.** Use `-9223372036854775807 - 1` or a host value.
13. **Non-finite checks are not recursive.** Validate nested composite values separately if needed.
14. **Type checking is best effort.** Unknown dynamic values are intentionally permissive.
15. **Studio transforms are conservative.** `Inline` will retain many bindings to preserve semantics.

---

## 16. Compact reference

### 16.1. Core forms

```text
one program = one expression

binding:      let x = value; body
lambda:       x => body  |  (a, b) => body  |  () => body
special names: # = first argument; $0, $1, ... = positional arguments
conditional:  condition ? then : else
coalescing:   a ?? b
pipeline:     value | function | function(args...)
```

### 16.2. Precedence, weak to strong

```text
or/||  →  and/&&  →  |  →  ^  →  &  →  == !=
→  < <= > >= in not-in ~=  →  shifts  →  + -
→  * / %  →  **  →  ??  →  ..
```

### 16.3. Value logic

```hyperex
"hi" && "bye"  // "bye"
0 && 5         // 0
5 || 0         // 5
nil ?? 1       // 1
0 ?? 1         // 0
2 + 3 ?? 4     // 5
```

### 16.4. Built-ins by group

```text
math:       abs min max ceil floor round sqrt pow log exp sin cos
string:     len upper lower trim trimLeft trimRight split join
            contains startsWith endsWith replace substr indexOf
array:      first last take skip reverse sort unique concat flatten
aggregation: sum avg count all any none one
higher-order: map filter reduce find findIndex
conversion: int float string bool type isNil
utilities:  print sprintf keys values clamp between hasKey empty notEmpty
            default coalesce repeat range zip chunk compact truthy pluck
            groupBy sortBy toJSON fromJSON
time:       timeUTC timeUnix timeUnixNano unix unixNano timeYear timeMonth
            timeDay timeHour timeMinute timeSecond timeWeekday timeAdd
            durationSeconds durationMs durationNs durSeconds durMs durNs
quality:    v q isGood isBad isUncertain withQ
```

### 16.5. Where to go next

- API and Go integration: [README.md](README.md) and [doc.go](doc.go).
- Runnable examples: [examples/](examples/).
- Interactive catalog: [examples/playground/](examples/playground/).
- Release history and compatibility notes: [CHANGELOG.md](CHANGELOG.md).
