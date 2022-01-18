package ast

// Node represents an interface that every AST node type implements to act as a super type
type Node interface {
	node()
}

// Base represents the base structure that every AST node embeds
type Base struct {
	Type nodeType `json:"type"`
	Span [2]uint  `json:"-"`
}

func (_ *Base) node() {}

// Resource represents the AST node of the whole FLT source (the parent node of the final AST)
type Resource struct {
	Base
	Body []Node `json:"body"` // Message, Term, Comment
}

// Identifier represents the identifier AST node
type Identifier struct {
	Base
	Name string `json:"name"`
}

// Comment represents the single-# comment AST node
type Comment struct {
	Base
	Content string `json:"content"`
}

// GroupComment represents the double-# comment AST node
type GroupComment struct {
	Base
	Content string `json:"content"`
}

// ResourceComment represents the triple-# comment AST node
type ResourceComment struct {
	Base
	Content string `json:"content"`
}

// Message represents the message declaration AST node
type Message struct {
	Base
	ID         *Identifier  `json:"id"`
	Value      *Pattern     `json:"value"`
	Attributes []*Attribute `json:"attributes"`
	Comment    *Comment     `json:"comment"`
}

// Term represents the term declaration AST node
type Term struct {
	Base
	ID         *Identifier  `json:"id"`
	Value      *Pattern     `json:"value"`
	Attributes []*Attribute `json:"attributes"`
	Comment    *Comment     `json:"comment"`
}

// Attribute represents the AST node of an attribute of a message or term
type Attribute struct {
	Base
	ID    *Identifier `json:"id"`
	Value *Pattern    `json:"value"`
}

// Pattern represents the pattern AST node consisting of text and placeables
type Pattern struct {
	Base
	Elements []Node `json:"elements"` // Pattern elements (Text & Placeable)
}

// Text represents a simple text AST node
type Text struct {
	Base
	Value string `json:"value"`
}

// Placeable represents the placeable AST node
type Placeable struct {
	Base
	Expression Node `json:"expression"` // Expression (References & Select)
}

// StringLiteral represents a literal string AST node
type StringLiteral struct {
	Base
	Value string `json:"value"`
}

// NumberLiteral represents a literal number AST node
type NumberLiteral struct {
	Base
	Value string `json:"value"`
}

// MessageReference represents the AST node of a reference to a message
type MessageReference struct {
	Base
	ID        *Identifier `json:"id"`
	Attribute *Identifier `json:"attribute"`
}

// TermReference represents the AST node of a reference to a term
type TermReference struct {
	Base
	ID        *Identifier    `json:"id"`
	Attribute *Identifier    `json:"attribute"`
	Arguments *CallArguments `json:"arguments"`
}

// VariableReference represents the AST node of a reference to a variable
type VariableReference struct {
	Base
	ID *Identifier `json:"id"`
}

// FunctionReference represents the AST node of a reference to a function
type FunctionReference struct {
	Base
	ID        *Identifier    `json:"id"`
	Arguments *CallArguments `json:"arguments"`
}

// CallArguments represents the AST node of arguments passed to a term or function reference
type CallArguments struct {
	Base
	Positional []Node           `json:"positional"` // Expression (References & Select)
	Named      []*NamedArgument `json:"named"`
}

// NamedArgument represents the AST node of a named argument passed to a term or function reference
type NamedArgument struct {
	Base
	Name  *Identifier `json:"name"`
	Value Node        `json:"value"` // Literal
}

// SelectExpression represents the select AST node
type SelectExpression struct {
	Base
	Selector Node       `json:"selector"` // Expression (References & Select)
	Variants []*Variant `json:"variants"`
}

// Variant represents the AST node of a select expression variant
type Variant struct {
	Base
	Key     Node     `json:"key"` // Identifier or NumberLiteral
	Value   *Pattern `json:"value"`
	Default bool     `json:"default"`
}

// Junk represents the AST node of unparsed content
type Junk struct {
	Base
	Content     string   `json:"content"`
	Annotations []string `json:"annotations"`
}
