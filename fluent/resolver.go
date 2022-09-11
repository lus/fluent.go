package fluent

import (
	"fmt"
	"github.com/lus/fluent.go/fluent/parser/ast"
	"golang.org/x/text/feature/plural"
	"strconv"
	"strings"
)

var pluralStrings = map[plural.Form]string{
	plural.Other: "other",
	plural.Zero:  "zero",
	plural.One:   "one",
	plural.Two:   "two",
	plural.Few:   "few",
	plural.Many:  "many",
}

// The resolver is used to resolve instances of as t.Pattern into instances of Value.
// It uses context-relevant values and the initial Bundle for resolving specific values.
type resolver struct {
	bundle    *Bundle
	params    map[string]Value
	variables map[string]Value
	functions map[string]Function
	errors    []error
}

func (resolver *resolver) resolveExpression(expression ast.Node) Value {
	switch e := expression.(type) {
	case *ast.Placeable:
		return resolver.resolveExpression(e.Expression)

	case *ast.StringLiteral:
		return strUnescape(e.Value)

	case *ast.NumberLiteral:
		parsed, err := strconv.ParseFloat(e.Value, 32)
		if err != nil {
			resolver.errors = append(resolver.errors, err)
			return &NoValue{value: "[" + e.Value + "]"}
		}
		return &NumberValue{Value: float32(parsed)}

	case *ast.MessageReference:
		return resolver.resolveMessageReference(e)

	case *ast.TermReference:
		return resolver.resolveTermReference(e)

	case *ast.VariableReference:
		return resolver.resolveVariableReference(e)

	case *ast.FunctionReference:
		return resolver.resolveFunctionReference(e)

	case *ast.SelectExpression:
		return resolver.resolveSelectExpression(e)

	// omitting ast.Identifier, cause it's not self-sufficient node, and works
	// only with other types.

	default:
		return &NoValue{value: "???"}
	}
}

func (resolver *resolver) resolveMessageReference(ref *ast.MessageReference) Value {
	message := resolver.bundle.messages[ref.ID.Name]
	if message == nil {
		resolver.errors = append(resolver.errors, fmt.Errorf("unknown message '%s'", ref.ID.Name))
		return &NoValue{
			value: ref.ID.Name,
		}
	}

	if ref.Attribute != nil {
		var attribute *ast.Attribute
		for _, attr := range message.Attributes {
			if attr.ID.Name == ref.Attribute.Name {
				attribute = attr
				break
			}
		}
		if attribute == nil {
			resolver.errors = append(resolver.errors, fmt.Errorf("unknown message attribute '%s.%s'", ref.ID.Name, ref.Attribute.Name))
			return &NoValue{
				value: ref.ID.Name + "." + ref.Attribute.Name,
			}
		}
		return resolver.resolvePattern(attribute.Value)
	}

	if message.Value == nil {
		resolver.errors = append(resolver.errors, fmt.Errorf("message '%s' has no value", ref.ID.Name))
		return &NoValue{
			value: ref.ID.Name,
		}
	}

	return resolver.resolvePattern(message.Value)
}

func (resolver *resolver) resolveTermReference(ref *ast.TermReference) Value {
	term := resolver.bundle.terms[ref.ID.Name]
	if term == nil {
		resolver.errors = append(resolver.errors, fmt.Errorf("unknown term '%s'", ref.ID.Name))
		return &NoValue{
			value: ref.ID.Name,
		}
	}

	if ref.Attribute != nil {
		var attribute *ast.Attribute
		for _, attr := range term.Attributes {
			if attr.ID.Name == ref.Attribute.Name {
				attribute = attr
				break
			}
		}
		if attribute == nil {
			resolver.errors = append(resolver.errors, fmt.Errorf("unknown term attribute '%s.%s'", ref.ID.Name, ref.Attribute.Name))
			return &NoValue{
				value: ref.ID.Name + "." + ref.Attribute.Name,
			}
		}
		if ref.Arguments != nil {
			_, params := resolver.assembleArguments(ref.Arguments)
			resolver.params = params
		}
		resolved := resolver.resolvePattern(attribute.Value)
		resolver.params = nil
		return resolved
	}

	if term.Value == nil {
		resolver.errors = append(resolver.errors, fmt.Errorf("term '%s' has no value", ref.ID.Name))
		return &NoValue{
			value: ref.ID.Name,
		}
	}

	if ref.Arguments != nil {
		_, params := resolver.assembleArguments(ref.Arguments)
		resolver.params = params
	}
	resolved := resolver.resolvePattern(term.Value)
	resolver.params = nil
	return resolved
}

func (resolver *resolver) resolveVariableReference(ref *ast.VariableReference) Value {
	if resolver.params != nil {
		if val, set := resolver.params[ref.ID.Name]; set {
			return val
		}
		return &NoValue{"$" + ref.ID.Name}
	} else if resolver.variables != nil {
		if val, set := resolver.variables[ref.ID.Name]; set {
			return val
		}
	}

	resolver.errors = append(resolver.errors, fmt.Errorf("unknown variable '$%s'", ref.ID.Name))
	return &NoValue{"$" + ref.ID.Name}
}

func (resolver *resolver) resolveFunctionReference(ref *ast.FunctionReference) Value {
	function := resolver.functions[ref.ID.Name]
	if function == nil {
		resolver.errors = append(resolver.errors, fmt.Errorf("unknown function '%s'", ref.ID.Name))
		return &NoValue{
			value: ref.ID.Name,
		}
	}

	positional, named := resolver.assembleArguments(ref.Arguments)
	return function(positional, named)
}

func (resolver *resolver) resolveSelectExpression(ref *ast.SelectExpression) Value {
	selector := resolver.resolveExpression(ref.Selector)
	if _, ok := selector.(*NoValue); ok {
		return resolver.resolveDefaultVariant(ref.Variants)
	}

	for _, variant := range ref.Variants {
		if resolver.matchesVariant(selector, resolver.resolveExpression(variant.Key)) {
			return resolver.resolvePattern(variant.Value)
		}
	}

	return resolver.resolveDefaultVariant(ref.Variants)
}

func (resolver *resolver) resolveDefaultVariant(variants []*ast.Variant) Value {
	for _, variant := range variants {
		if variant.Default {
			return resolver.resolvePattern(variant.Value)
		}
	}
	resolver.errors = append(resolver.errors, fmt.Errorf("no default variant specified"))
	return &NoValue{
		value: "???",
	}
}

func (resolver *resolver) matchesVariant(selector, variant Value) bool {
	if selStr, ok := selector.(*StringValue); ok {
		if varStr, ok := variant.(*StringValue); ok {
			return selStr.Value == varStr.Value
		}
	}

	if selNum, ok := selector.(*NumberValue); ok {
		if varNum, ok := variant.(*NumberValue); ok {
			return selNum.Value == varNum.Value
		}
		if varStr, ok := variant.(*StringValue); ok {
			category := pluralStrings[resolver.getPluralCategory(selNum.Value)]
			return varStr.Value == category
		}
	}

	return false
}

func (resolver *resolver) resolvePattern(pattern *ast.Pattern) Value {
	result := ""
	for _, element := range pattern.Elements {
		if text, ok := element.(*ast.Text); ok {
			result += text.Value
			continue
		}
		result += resolver.resolveExpression(element.(*ast.Placeable).Expression).String()
	}
	return &StringValue{
		Value: result,
	}
}

func (resolver *resolver) assembleArguments(args *ast.CallArguments) (positional []Value, named map[string]Value) {
	positional = make([]Value, 0, len(args.Positional))
	for _, arg := range args.Positional {
		positional = append(positional, resolver.resolveExpression(arg))
	}
	named = make(map[string]Value, len(args.Named))
	for _, arg := range args.Named {
		named[arg.Name.Name] = resolver.resolveExpression(arg.Value)
	}
	return
}

func (resolver *resolver) getPluralCategory(value float32) plural.Form {
	format := fmt.Sprintf("%.2f", value)
	parts := strings.Split(strings.TrimRight(format, "0"), ".")

	bytes := make([]byte, len(parts[0])+len(parts[1]))
	for i, digit := range parts[0] {
		bytes[i] = byte(digit - 48)
	}
	for i, digit := range parts[1] {
		bytes[i+len(parts[0])] = byte(digit - 48)
	}

	return plural.Cardinal.MatchDigits(resolver.bundle.locales[0], bytes, len(parts[0]), len(parts[1]))
}
