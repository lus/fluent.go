package fluent

import (
	"fmt"
	"strconv"
)

// TODO: Implement DateTimes

// Function represents a function that builds a Value based on parameters
type Function func(positional []Value, named map[string]Value) Value

// A Value is the result of a resolving operation performed by the Resolver.
// It represents either a string, a number or a date time.
type Value interface {
	fmt.Stringer
	_value()
}

// StringValue wraps a string in order to comply with the Value API
type StringValue struct {
	Value string
}

func (*StringValue) _value() {}

// String returns the wrapped value of a StringValue
func (value *StringValue) String() string {
	return value.Value
}

// String returns a new StringValue with the given value; used for variables
//
// DEPRECATED: Use direct `StringValue{"some string"}` for constructing object
func String(val string) *StringValue {
	return &StringValue{
		Value: val,
	}
}

func strUnescape(v string) Value {
	if unescaped, err := strconv.Unquote("\"" + v + "\""); err != nil {
		return &NoValue{"%!(UNESCAPE " + err.Error() + ")"}
	} else {
		return &StringValue{unescaped}
	}
}

// NumberValue wraps a number (float32 at the moment) in order to comply with the Value API
type NumberValue struct {
	Value float32
}

func (*NumberValue) _value() {}

// String formats a NumberValue into a string
func (value *NumberValue) String() string {
	// TODO: Simulate JavaScript number formatting
	return strconv.FormatFloat(float64(value.Value), 'f', -1, 32)
}

// Number returns a new NumberValue with the given value; used for variables
func Number(val float32) *NumberValue {
	return &NumberValue{
		Value: val,
	}
}

// NoValue is used whenever no "real" value could be built
type NoValue struct {
	value string
}

func (*NoValue) _value() {}

// String returns the NoValue's string representation
func (value *NoValue) String() string {
	return "{" + value.value + "}"
}
