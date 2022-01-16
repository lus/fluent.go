package fluent

import (
	"github.com/lus/fluent.go/parser"
	"github.com/lus/fluent.go/parser/ast"
)

// Resource represents a collection of messages and terms extracted out of a FTL source
type Resource struct {
	messages []*ast.Message
	terms    []*ast.Term
}

// NewResource parses the given source string and assembles its entries into a new Resource object.
// Besides the Resource object, this method also returns all errors the parser stumbled upon during parsing.
// As long as Resource.IsEmpty does not return false, at least something could be parsed successfully.
func NewResource(source string) (*Resource, []*parser.Error) {
	// Parse the source string into an AST
	parsed, errs := parser.New(source).Parse()

	resource := &Resource{
		messages: make([]*ast.Message, 0),
		terms:    make([]*ast.Term, 0),
	}

	// Add messages and terms to the resource; junk and comments are ignored
	for _, entry := range parsed.Body {
		if message, ok := entry.(*ast.Message); ok {
			resource.messages = append(resource.messages, message)
		}
		if term, ok := entry.(*ast.Term); ok {
			resource.terms = append(resource.terms, term)
		}
	}

	return resource, errs
}

// IsEmpty returns if no terms and no messages are present in the resource.
// This can be the case if the parser could not parse any valid messages and terms.
func (resource *Resource) IsEmpty() bool {
	return len(resource.messages) == 0 && len(resource.terms) == 0
}
