package ast

// nodeType is used to declare the different possible types of AST nodes
type nodeType string

const (
	TypeResource          nodeType = "Resource"
	TypeIdentifier        nodeType = "Identifier"
	TypeComment           nodeType = "Comment"
	TypeGroupComment      nodeType = "GroupComment"
	TypeResourceComment   nodeType = "ResourceComment"
	TypeMessage           nodeType = "Message"
	TypeTerm              nodeType = "Term"
	TypeAttribute         nodeType = "Attribute"
	TypePattern           nodeType = "Pattern"
	TypeText              nodeType = "TextElement"
	TypePlaceable         nodeType = "Placeable"
	TypeStringLiteral     nodeType = "StringLiteral"
	TypeNumberLiteral     nodeType = "NumberLiteral"
	TypeMessageReference  nodeType = "MessageReference"
	TypeTermReference     nodeType = "TermReference"
	TypeVariableReference nodeType = "VariableReference"
	TypeFunctionReference nodeType = "FunctionReference"
	TypeCallArguments     nodeType = "CallArguments"
	TypeNamedArgument     nodeType = "NamedArgument"
	TypeSelectExpression  nodeType = "SelectExpression"
	TypeVariant           nodeType = "Variant"
	TypeJunk              nodeType = "Junk"
)

// IsEntry checks if a type represents an entry of a resource
func IsEntry(typ nodeType) bool {
	return IsComment(typ) || anyOf(typ, TypeMessage, TypeTerm)
}

// IsComment checks if a type represents any of the three comment types
func IsComment(typ nodeType) bool {
	return anyOf(typ, TypeComment, TypeGroupComment, TypeResourceComment)
}

// IsPatternElement checks if a type represents an element of a pattern
func IsPatternElement(typ nodeType) bool {
	return anyOf(typ, TypeText, TypePlaceable)
}

// IsExpression checks if a type represents an expression used in placeables
func IsExpression(typ nodeType) bool {
	return IsLiteral(typ) || anyOf(TypeMessageReference, TypeTermReference, TypeVariableReference, TypeFunctionReference, TypeSelectExpression)
}

// IsLiteral checks if a type represents a literal (text or number)
func IsLiteral(typ nodeType) bool {
	return anyOf(typ, TypeStringLiteral, TypeNumberLiteral)
}

// anyOf checks if the given type matches any of the specified other types
func anyOf(typ nodeType, types ...nodeType) bool {
	for _, toCompare := range types {
		if typ == toCompare {
			return true
		}
	}
	return false
}
