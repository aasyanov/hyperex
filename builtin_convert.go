package hyperex

// Type conversion builtins
func builtinInt(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	// Use TryInt for strict conversion - returns error on invalid strings
	n, err := args[0].TryInt()
	if err != nil {
		return Nil, &RuntimeError{Message: err.Error()}
	}
	return Int(n), nil
}

func builtinFloat(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	// Use TryFloat for strict conversion - returns error on invalid strings
	f, err := args[0].TryFloat()
	if err != nil {
		return Nil, &RuntimeError{Message: err.Error()}
	}
	return Float(f), nil
}

func builtinString(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(args[0].String()), nil
}

func builtinBool(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Bool(args[0].Bool()), nil
}

func builtinType(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(args[0].Type()), nil
}

func builtinIsNil(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Bool(args[0].IsNil()), nil
}
