package parser

import "unicode"

// anyOf checks whether a rune matches another one from the specified set
func anyOf(val rune, set ...rune) bool {
	for _, toCompare := range set {
		if val == toCompare {
			return true
		}
	}
	return false
}

// hasLowercase checks whether the given set of runes contains a lowercase letter
func hasLowercase(set []rune) bool {
	for _, char := range set {
		if unicode.IsLetter(char) && unicode.IsLower(char) {
			return true
		}
	}
	return false
}
