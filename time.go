package hyperex

import (
	"math"
	"time"
)

const (
	nsPerMillisecond = int64(time.Millisecond)
	nsPerSecond      = int64(time.Second)
)

// register time/duration builtins into the default set (called from defaultBuiltins).
func timeDurationBuiltins() map[string]Value {
	return map[string]Value{
		nameTimeUTC:         Func(&builtinFunc{name: nameTimeUTC, fn: wrapBuiltin(builtinTimeUTC)}),
		nameTimeUnix:        Func(&builtinFunc{name: nameTimeUnix, fn: wrapBuiltin(builtinTimeUnix)}),
		nameTimeUnixNano:    Func(&builtinFunc{name: nameTimeUnixNano, fn: wrapBuiltin(builtinTimeUnixNano)}),
		nameUnix:            Func(&builtinFunc{name: nameUnix, fn: wrapBuiltin(builtinUnix)}),
		nameUnixNano:        Func(&builtinFunc{name: nameUnixNano, fn: wrapBuiltin(builtinUnixNano)}),
		nameTimeYear:        Func(&builtinFunc{name: nameTimeYear, fn: wrapBuiltin(builtinTimeYear)}),
		nameTimeMonth:       Func(&builtinFunc{name: nameTimeMonth, fn: wrapBuiltin(builtinTimeMonth)}),
		nameTimeDay:         Func(&builtinFunc{name: nameTimeDay, fn: wrapBuiltin(builtinTimeDay)}),
		nameTimeHour:        Func(&builtinFunc{name: nameTimeHour, fn: wrapBuiltin(builtinTimeHour)}),
		nameTimeMinute:      Func(&builtinFunc{name: nameTimeMinute, fn: wrapBuiltin(builtinTimeMinute)}),
		nameTimeSecond:      Func(&builtinFunc{name: nameTimeSecond, fn: wrapBuiltin(builtinTimeSecond)}),
		nameTimeWeekday:     Func(&builtinFunc{name: nameTimeWeekday, fn: wrapBuiltin(builtinTimeWeekday)}),
		nameTimeAdd:         Func(&builtinFunc{name: nameTimeAdd, fn: wrapBuiltin(builtinTimeAdd)}),
		nameDurationSeconds: Func(&builtinFunc{name: nameDurationSeconds, fn: wrapBuiltin(builtinDurationSeconds)}),
		nameDurationMs:      Func(&builtinFunc{name: nameDurationMs, fn: wrapBuiltin(builtinDurationMs)}),
		nameDurationNs:      Func(&builtinFunc{name: nameDurationNs, fn: wrapBuiltin(builtinDurationNs)}),
		nameDurSeconds:      Func(&builtinFunc{name: nameDurSeconds, fn: wrapBuiltin(builtinDurSeconds)}),
		nameDurMs:           Func(&builtinFunc{name: nameDurMs, fn: wrapBuiltin(builtinDurMs)}),
		nameDurNs:           Func(&builtinFunc{name: nameDurNs, fn: wrapBuiltin(builtinDurNs)}),
	}
}

func requireTimeArg(args []Value, name string) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsTime() {
		return Nil, &RuntimeError{Message: name + ": expected time"}
	}
	return args[0], nil
}

func requireDurationArg(args []Value, name string) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsDuration() {
		return Nil, &RuntimeError{Message: name + ": expected duration"}
	}
	return args[0], nil
}

func numericAsInt64(v Value) (int64, error) {
	if v.IsInt() {
		return v.Int(), nil
	}
	if v.IsFloat() {
		n, ok := float64ToInt64Exact(v.Float())
		if !ok {
			return 0, &RuntimeError{Message: "expected integer number"}
		}
		return n, nil
	}
	return 0, &RuntimeError{Message: "expected number"}
}

func numericAsFloat64(v Value) (float64, error) {
	if v.IsNumber() {
		return v.Float(), nil
	}
	return 0, &RuntimeError{Message: "expected number"}
}

func scaleToDuration(n float64, unitNs int64, name string) (Value, error) {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return Nil, &RuntimeError{Message: name + ": invalid number"}
	}
	product := n * float64(unitNs)
	const maxInt64Plus1 = 9223372036854775808.0
	if product < float64(math.MinInt64) || product >= maxInt64Plus1 {
		return Nil, &RuntimeError{Message: name + ": duration out of range"}
	}
	return Duration(time.Duration(int64(product))), nil
}

func builtinTimeUTC(args []Value) (Value, error) {
	if len(args) != 6 {
		return Nil, ErrArgumentCount
	}
	y, err := numericAsInt64(args[0])
	if err != nil {
		return Nil, err
	}
	month, err := numericAsInt64(args[1])
	if err != nil {
		return Nil, err
	}
	day, err := numericAsInt64(args[2])
	if err != nil {
		return Nil, err
	}
	hour, err := numericAsInt64(args[3])
	if err != nil {
		return Nil, err
	}
	min, err := numericAsInt64(args[4])
	if err != nil {
		return Nil, err
	}
	sec, err := numericAsInt64(args[5])
	if err != nil {
		return Nil, err
	}
	t := time.Date(int(y), time.Month(month), int(day), int(hour), int(min), int(sec), 0, time.UTC)
	return Time(t), nil
}

func builtinTimeUnix(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	sec, err := numericAsInt64(args[0])
	if err != nil {
		return Nil, err
	}
	return Time(time.Unix(sec, 0).UTC()), nil
}

func builtinTimeUnixNano(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	ns, err := numericAsInt64(args[0])
	if err != nil {
		return Nil, err
	}
	return Time(time.Unix(0, ns).UTC()), nil
}

func builtinUnix(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameUnix)
	if err != nil {
		return Nil, err
	}
	return Int(timeFromBits(t.bits).Unix()), nil
}

func builtinUnixNano(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameUnixNano)
	if err != nil {
		return Nil, err
	}
	return Int(int64(t.bits)), nil
}

func builtinTimeYear(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameTimeYear)
	if err != nil {
		return Nil, err
	}
	return Int(int64(timeFromBits(t.bits).Year())), nil
}

func builtinTimeMonth(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameTimeMonth)
	if err != nil {
		return Nil, err
	}
	return Int(int64(timeFromBits(t.bits).Month())), nil
}

func builtinTimeDay(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameTimeDay)
	if err != nil {
		return Nil, err
	}
	return Int(int64(timeFromBits(t.bits).Day())), nil
}

func builtinTimeHour(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameTimeHour)
	if err != nil {
		return Nil, err
	}
	return Int(int64(timeFromBits(t.bits).Hour())), nil
}

func builtinTimeMinute(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameTimeMinute)
	if err != nil {
		return Nil, err
	}
	return Int(int64(timeFromBits(t.bits).Minute())), nil
}

func builtinTimeSecond(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameTimeSecond)
	if err != nil {
		return Nil, err
	}
	return Int(int64(timeFromBits(t.bits).Second())), nil
}

func builtinTimeWeekday(args []Value) (Value, error) {
	t, err := requireTimeArg(args, nameTimeWeekday)
	if err != nil {
		return Nil, err
	}
	// Go convention: Sunday = 0 … Saturday = 6.
	return Int(int64(timeFromBits(t.bits).Weekday())), nil
}

func builtinTimeAdd(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsTime() || !args[1].IsDuration() {
		return Nil, &RuntimeError{Message: nameTimeAdd + ": expected (time, duration)"}
	}
	return addTimeDuration(args[0], args[1])
}

func builtinDurationSeconds(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	n, err := numericAsFloat64(args[0])
	if err != nil {
		return Nil, err
	}
	return scaleToDuration(n, nsPerSecond, nameDurationSeconds)
}

func builtinDurationMs(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	n, err := numericAsFloat64(args[0])
	if err != nil {
		return Nil, err
	}
	return scaleToDuration(n, nsPerMillisecond, nameDurationMs)
}

func builtinDurationNs(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	ns, err := numericAsInt64(args[0])
	if err != nil {
		return Nil, err
	}
	return Duration(time.Duration(ns)), nil
}

func builtinDurSeconds(args []Value) (Value, error) {
	d, err := requireDurationArg(args, nameDurSeconds)
	if err != nil {
		return Nil, err
	}
	return Float(durationFromBits(d.bits).Seconds()), nil
}

func builtinDurMs(args []Value) (Value, error) {
	d, err := requireDurationArg(args, nameDurMs)
	if err != nil {
		return Nil, err
	}
	return Int(durationFromBits(d.bits).Milliseconds()), nil
}

func builtinDurNs(args []Value) (Value, error) {
	d, err := requireDurationArg(args, nameDurNs)
	if err != nil {
		return Nil, err
	}
	return Int(int64(d.bits)), nil
}
