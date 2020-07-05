package api

import (
	"io"
)

// Match represents a search hit.
// It contains the searched-for word, and the location where it was found.
// Location is a 2 element int array with the start index (inclusive),
// and the end index (exclusive).
type Match struct {
	Term     string
	Location [2]int
}

// Searcher is a widget that can search some Text for a set of search terms.
type Searcher interface {
	AddSearchTerm(searchTerm string)
	Search(r io.RuneReader) <-chan Match
}
