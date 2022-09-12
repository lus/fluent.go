package fluent

import (
	"fmt"
	"github.com/lus/fluent.go/fluent/parser/ast"
	"golang.org/x/text/language"
	"strings"
)

// Bundle represents a collection of messages and terms collected from one or many resources.
// It provides the main API to format messages.
type Bundle struct {
	locales  []language.Tag
	messages map[string]*ast.Message
	terms    map[string]*ast.Term
}

// NewBundle creates a new empty bundle
func NewBundle(primaryLocale language.Tag, fallbackLocales ...language.Tag) *Bundle {
	locales := make([]language.Tag, 0, len(fallbackLocales)+1)
	locales = append(locales, primaryLocale)
	for _, fallback := range fallbackLocales {
		locales = append(locales, fallback)
	}

	return &Bundle{
		locales:  locales,
		messages: make(map[string]*ast.Message),
		terms:    make(map[string]*ast.Term),
	}
}

// AddResource adds a Resource to the Bundle.
// If a message or term was already defined by another resource, an error is raised and the entry is skipped.
func (bundle *Bundle) AddResource(resource *Resource) (errs []error) {
	for _, message := range resource.messages {
		id := message.ID.Name
		if bundle.messages[id] != nil {
			errs = append(errs, fmt.Errorf("message '%s' is already defined", id))
			continue
		}
		bundle.messages[id] = message
	}
	for _, term := range resource.terms {
		id := term.ID.Name
		if bundle.terms[id] != nil {
			errs = append(errs, fmt.Errorf("term '%s' is already defined", id))
			continue
		}
		bundle.terms[id] = term
	}
	return
}

// AddResourceOverriding adds a Resource to the Bundle.
// If a message or term was already defined by another resource, the already existing one gets overridden.
func (bundle *Bundle) AddResourceOverriding(resource *Resource) {
	for _, message := range resource.messages {
		bundle.messages[message.ID.Name] = message
	}
	for _, term := range resource.terms {
		bundle.terms[term.ID.Name] = term
	}
}

type formatOption func(*resolver)

// WithVariable creates a FormatContext with a single variable
func WithVariable(key string, value interface{}) formatOption {
	return WithVariables(map[string]interface{}{key: value})
}

// WithVariables creates a FormatContext with multiple variables
func WithVariables(variables map[string]interface{}) formatOption {
	return func(r *resolver) {
		if r.variables == nil {
			r.variables = make(map[string]Value, len(variables))
		}

		for name, variable := range variables {
			r.variables[strings.TrimSpace(name)] = resolveValue(variable)
		}
	}
}

func resolveValue(value interface{}) Value {
	if strVal, ok := value.(string); ok {
		return String(strVal)
	}
	if strVal, ok := value.(*StringValue); ok {
		return strVal
	}
	if float32Val, ok := value.(float32); ok {
		return Number(float32Val)
	}
	if float64Val, ok := value.(float64); ok {
		return Number(float32(float64Val))
	}
	if uintVal, ok := value.(uint); ok {
		return Number(float32(uintVal))
	}
	if uint8Val, ok := value.(uint8); ok {
		return Number(float32(uint8Val))
	}
	if uint16Val, ok := value.(uint16); ok {
		return Number(float32(uint16Val))
	}
	if uint32val, ok := value.(uint32); ok {
		return Number(float32(uint32val))
	}
	if uint64val, ok := value.(uint64); ok {
		return Number(float32(uint64val))
	}
	if intVal, ok := value.(int); ok {
		return Number(float32(intVal))
	}
	if int8Val, ok := value.(int8); ok {
		return Number(float32(int8Val))
	}
	if int16Val, ok := value.(int16); ok {
		return Number(float32(int16Val))
	}
	if int32val, ok := value.(int32); ok {
		return Number(float32(int32val))
	}
	if int64val, ok := value.(int64); ok {
		return Number(float32(int64val))
	}
	if numVal, ok := value.(*NumberValue); ok {
		return numVal
	}
	return nil
}

// WithFunction creates a FormatContext with a single function
func WithFunction(k string, f Function) formatOption {
	return WithFunctions(map[string]Function{k: f})
}

// WithFunctions creates a FormatContext with multiple functions
func WithFunctions(functions map[string]Function) formatOption {
	return func(r *resolver) {
		if r.functions == nil {
			r.functions = make(map[string]Function, len(functions))
		}

		for name, function := range functions {
			r.functions[strings.TrimSpace(name)] = function
		}
	}
}

// FormatMessage formats the message with the given key.
// To pass variables or functions, pass contexts created using WithVariable, WithVariables, WithFunction or WithFunctions.
// Besides the formatted message, this method returns the errors the resolver stumbled upon during resolving specific values
// and an optional error if there is no message with the given key.
// If the resolver returns errors it does not automatically mean that the whole message could not be resolved.
// It may be just incomplete.
func (bundle *Bundle) FormatMessage(key string, options ...formatOption) (string, []error, error) {
	if bundle.messages[key] == nil {
		return "", nil, fmt.Errorf("message '%s' does not exist", key)
	}

	msg := bundle.messages[key]
	res := &resolver{
		bundle:    bundle,
		params:    nil,
		variables: make(map[string]Value),
		functions: make(map[string]Function),
		errors:    []error{},
	}
	for _, opt := range options {
		opt(res)
	}

	result := res.resolvePattern(msg.Value).String()
	return result, res.errors, nil
}
