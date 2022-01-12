package ast

// baseNode represents an interface that every AST node type implements to act as a super type
type baseNode interface {
	node()
}

// Node represents the base structure that every AST node embeds
type Node struct {
	Type nodeType
	Span [2]uint
}

func (_ *Node) node() {}

// Resource represents the AST node of the whole FLT source (the parent node of the final AST)
type Resource struct {
	Node
	Entries []baseNode // Message, Term, Comment
}

// Identifier represents the identifier AST node
type Identifier struct {
	Node
	Name string
}

// Comment represents the single-# comment AST node
type Comment struct {
	Node
	Content string
}

// GroupComment represents the double-# comment AST node
type GroupComment struct {
	Node
	Content string
}

// ResourceComment represents the triple-# comment AST node
type ResourceComment struct {
	Node
	Content string
}

// Message represents the message declaration AST node
type Message struct {
	Node
	ID         Identifier
	Value      Pattern
	Attributes []Attribute
	Comment    Comment
}

// Term represents the term declaration AST node
type Term struct {
	Node
	ID         Identifier
	Value      Pattern
	Attributes []Attribute
	Comment    Comment
}

// Attribute represents the AST node of an attribute of a message or term
type Attribute struct {
	Node
	ID    Identifier
	Value baseNode // Pattern (Text, Placeable)
}

// Pattern represents the pattern AST node consisting of text and placeables
type Pattern struct {
	Node
	Elements []baseNode // Pattern elements (Text & Placeable)
}

// Text represents a simple text AST node
type Text struct {
	Node
	Value string
}

// Placeable represents the placeable AST node
type Placeable struct {
	Node
	Expression baseNode // Expression (References & Select)
}

// StringLiteral represents a literal string AST node
type StringLiteral struct {
	Node
	Value string
}

// NumberLiteral represents a literal number AST node
type NumberLiteral struct {
	Node
	Value float32
}

// MessageReference represents the AST node of a reference to a message
type MessageReference struct {
	Node
	ID        Identifier
	Attribute Identifier
}

// TermReference represents the AST node of a reference to a term
type TermReference struct {
	Node
	ID        Identifier
	Attribute Identifier
	Arguments CallArguments
}

// VariableReference represents the AST node of a reference to a variable
type VariableReference struct {
	Node
	ID Identifier
}

// FunctionReference represents the AST node of a reference to a function
type FunctionReference struct {
	Node
	ID        Identifier
	Arguments CallArguments
}

// CallArguments represents the AST node of arguments passed to a term or function reference
type CallArguments struct {
	Node
	Positional []baseNode // Expression (References & Select)
	Names      []NamedArgument
}

// NamedArgument represents the AST node of a named argument passed to a term or function reference
type NamedArgument struct {
	Node
	Name  Identifier
	Value baseNode // Expression (References & Select)
}

// SelectExpression represents the select AST node
type SelectExpression struct {
	Node
	Selector baseNode // Expression (References & Select)
	Variants []Variant
}

// Variant represents the AST node of a select expression variant
type Variant struct {
	Node
	Key     string
	Value   baseNode // Pattern (Text, Placeable)
	Default bool
}

// Junk represents the AST node of unparsed content
type Junk struct {
	Node
	Content string
}
