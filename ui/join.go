package ui

import "strings"

// Join concatenates the elements of its first argument to create a single string.
// The separator string sep is placed between elements in the resulting string.
func Join(elems []string, sep string) string {
	// Filter out empty strings before joining
	var validElems []string
	for _, elem := range elems {
		if elem != "" {
			validElems = append(validElems, elem)
		}
	}
	return strings.Join(validElems, sep)
}
