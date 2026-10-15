package hyperex

import (
	"strings"
	"unicode/utf8"
)

// String builtins
func builtinLen(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Int(int64(args[0].Len())), nil
}

func builtinUpper(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(strings.ToUpper(args[0].String())), nil
}

func builtinLower(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(strings.ToLower(args[0].String())), nil
}

func builtinTrim(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(strings.TrimSpace(args[0].String())), nil
}

func builtinTrimLeft(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	if len(args) == 2 {
		return String(strings.TrimLeft(s, args[1].String())), nil
	}
	return String(strings.TrimLeft(s, " \t\n\r")), nil
}

func builtinTrimRight(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	if len(args) == 2 {
		return String(strings.TrimRight(s, args[1].String())), nil
	}
	return String(strings.TrimRight(s, " \t\n\r")), nil
}

func builtinSplit(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	parts := strings.Split(args[0].String(), args[1].String())
	arr := make([]any, len(parts))
	for i, p := range parts {
		arr[i] = p
	}
	return Array(arr), nil
}

func builtinJoin(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = NewValue(v).String()
	}
	return String(strings.Join(parts, args[1].String())), nil
}

func builtinContains(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	return Bool(strings.Contains(args[0].String(), args[1].String())), nil
}

func builtinStartsWith(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	return Bool(strings.HasPrefix(args[0].String(), args[1].String())), nil
}

func builtinEndsWith(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	return Bool(strings.HasSuffix(args[0].String(), args[1].String())), nil
}

func builtinReplace(args []Value) (Value, error) {
	if len(args) != 3 {
		return Nil, ErrArgumentCount
	}
	return String(strings.ReplaceAll(args[0].String(), args[1].String(), args[2].String())), nil
}

func builtinSubstr(args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	runes := []rune(s)
	runeLen := len(runes)

	start64 := args[1].Int()
	// Clamp extremes without relying on int width (important on 32-bit platforms).
	if start64 > int64(runeLen) {
		return String(""), nil
	}
	if start64 < -int64(runeLen) {
		start64 = 0
	}
	start := int(start64)
	if start < 0 {
		start = runeLen + start
	}
	if start < 0 {
		start = 0
	}
	if start > runeLen {
		return String(""), nil
	}
	if len(args) == 3 {
		length64 := args[2].Int()
		// Guard against negative length
		if length64 < 0 {
			length64 = 0
		}
		// Compute end in int64 then clamp into [start, runeLen] to avoid int overflow.
		end64 := int64(start) + length64
		if end64 < int64(start) || end64 > int64(runeLen) {
			end64 = int64(runeLen)
		}
		return String(string(runes[start:int(end64)])), nil
	}
	return String(string(runes[start:])), nil
}

func builtinIndexOf(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	substr := args[1].String()
	// Get byte index
	byteIdx := strings.Index(s, substr)
	if byteIdx == -1 {
		return Int(-1), nil
	}
	// Convert byte index to rune index
	runeIdx := utf8.RuneCountInString(s[:byteIdx])
	return Int(int64(runeIdx)), nil
}
