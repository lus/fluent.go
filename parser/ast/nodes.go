package ast

// Node represents an interface that every AST node type implements to act as a super type
type Node interface {
	node()
}

// Base represents the base structure that every AST node embeds
type Base struct {
	Type nodeType
	Span [2]uint
}

func (_ *Base) node() {}

// Resource represents the AST node of the whole FLT source (the parent node of the final AST)
type Resource struct {
	Base
	Body []Node // Message, Term, Comment
}

// Identifier represents the identifier AST node
type Identifier struct {
	Base
	Name string
}

// Comment represents the single-# comment AST node
type Comment struct {
	Base
	Content string
}

// GroupComment represents the double-# comment AST node
type GroupComment struct {
	Base
	Content string
}

// ResourceComment represents the triple-# comment AST node
type ResourceComment struct {
	Base
	Content string
}

// Message represents the message declaration AST node
type Message struct {
	Base
	ID         *Identifier
	Value      *Pattern
	Attributes []*Attribute
	Comment    *Comment
}

// Term represents the term declaration AST node
type Term struct {
	Base
	ID         *Identifier
	Value      *Pattern
	Attributes []*Attribute
	Comment    *Comment
}

// Attribute represents the AST node of an attribute of a message or term
type Attribute struct {
	Base
	ID    *Identifier
	Value *Pattern
}

// Pattern represents the pattern AST node consisting of text and placeables
type Pattern struct {
	Base
	Elements []Node // Pattern elements (Text & Placeable)
}

// Text represents a simple text AST node
type Text struct {
	Base
	Value string
}

// Placeable represents the placeable AST node
type Placeable struct {
	Base
	Expression Node // Expression (References & Select)
}

// StringLiteral represents a literal string AST node
type StringLiteral struct {
	Base
	Value string
}

// NumberLiteral represents a literal number AST node
type NumberLiteral struct {
	Base
	Value string
}

// MessageReference represents the AST node of a reference to a message
type MessageReference struct {
	Base
	ID        *Identifier
	Attribute *Identifier
}

// TermReference represents the AST node of a reference to a term
type TermReference struct {
	Base
	ID        *Identifier
	Attribute *Identifier
	Arguments *CallArguments
}

// VariableReference represents the AST node of a reference to a variable
type VariableReference struct {
	Base
	ID *Identifier
}

// FunctionReference represents the AST node of a reference to a function
type FunctionReference struct {
	Base
	ID        *Identifier
	Arguments *CallArguments
}

// CallArguments represents the AST node of arguments passed to a term or function reference
type CallArguments struct {
	Base
	Positional []Node // Expression (References & Select)
	Named      []*NamedArgument
}

// NamedArgument represents the AST node of a named argument passed to a term or function reference
type NamedArgument struct {
	Base
	Name  *Identifier
	Value Node // Literal
}

// SelectExpression represents the select AST node
type SelectExpression struct {
	Base
	Selector Node // Expression (References & Select)
	Variants []*Variant
}

// Variant represents the AST node of a select expression variant
type Variant struct {
	Base
	Key     Node // Identifier or NumberLiteral
	Value   *Pattern
	Default bool
}

// Junk represents the AST node of unparsed content
type Junk struct {
	Base
	Content     string
	Annotations []string
}
