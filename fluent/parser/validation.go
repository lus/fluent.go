package parser

// isEntryStart checks if a character is valid to be the start of a new entry
func isEntryStart(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '#' || char == '-'
}

// isIdentifierStart checks if a character is valid to be the start of an identifier
func isIdentifierStart(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

// isIdentifierFollowing checks if a character is valid to be part of an identifier
func isIdentifierFollowing(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-'
}
