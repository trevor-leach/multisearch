package ahocorasick

import (
	"container/list"
	"encoding/json"
	"io"

	"github.com/trevor-leach/multisearch/api"
)

var id int = -1

// SearchTrie see https://en.wikipedia.org/wiki/Aho%E2%80%93Corasick_algorithm
// and  http://se.ethz.ch/~meyer/publications/string/string_matching.pdf
type SearchTrie struct {
	id       int
	root     *SearchTrie
	char     rune
	isWord   bool // if this node represents a full word
	children map[rune]*SearchTrie
	lps      *SearchTrie         // longest proper suffix.
	ilps     map[int]*SearchTrie // inverse lps
	ot       map[string]bool     // set of suffixes that are full words in this trie.
}

// New returns an initialized SearchTrie.
func New(searchStrings []string) *SearchTrie {
	id++
	s := new(SearchTrie)
	s.id = id
	s.root = s
	s.children = make(map[rune]*SearchTrie)
	s.ilps = make(map[int]*SearchTrie)
	s.ot = make(map[string]bool)
	s.buildTrie(searchStrings)
	s.buildFailureFn()

	return s
}

// isRoot gets whether s is the root of the trie or not.
func (s *SearchTrie) isRoot() bool {
	return s == s.root
}

// failureFn gets the longest proper suffix for the node, or the root node.
func (s *SearchTrie) failureFn() *SearchTrie {
	if nil == s.lps {
		return s.root
	}

	return s.lps
}

// getChild gets the child corresponding to the specified rune,
// or the root node.
func (s *SearchTrie) getChild(char rune) *SearchTrie {
	if child, ok := s.children[char]; ok {
		return child
	}
	return s.root
}

// Search searches the specified text for the previously added search strings.
func (s SearchTrie) Search(r io.RuneReader) <-chan api.Match {
	ch := make(chan api.Match, 2)
	go func() {
		n := &s
		index := 0
		for {
			char, nbytes, err := r.ReadRune() // returns rune, nbytes, error
			if nil != err {
				break
			}
			index += nbytes

			for !n.isRoot() && n.children[char] == nil {
				n = n.failureFn()
			}
			n = n.getChild(char)

			for t := range n.ot {
				ch <- api.Match{
					Term:     t,
					Location: [2]int{index - len(t), index}}
			}
		}
		close(ch)
	}()
	return ch
}

// AddSearchTerm adds another search string to the Searcher.
func (s SearchTrie) AddSearchTerm(searchTerm string) {
	s.enterInTrie(searchTerm)
}

func (s *SearchTrie) buildTrie(searchStrings []string) {
	for _, str := range searchStrings {
		s.enterInTrie(str)
	}
}

func (s *SearchTrie) enterInTrie(str string) {
	var current, next *SearchTrie
	current = s

	for _, char := range str {
		next = current.getChild(char)
		if next.isRoot() {
			next = current.enterChild(char)
		}
		current = next
	}
	current.isWord = true
	current.enterOutput(str)
}

func (s *SearchTrie) enterChild(char rune) *SearchTrie {
	id++
	child := &SearchTrie{
		id:       id,
		root:     s.root,
		char:     char,
		children: make(map[rune]*SearchTrie),
		ilps:     make(map[int]*SearchTrie),
		ot:       make(map[string]bool)}

	s.children[char] = child
	s.completeFailureFn(char)
	if nil != child.lps {
		child.lps.ilps[child.id] = child
	}
	s.completeInverseFn(child, char)

	return child
}

func (s *SearchTrie) enterOutput(str string) {
	s.ot[str] = true
	for _, x := range s.ilps {
		x.enterOutput(str)
	}
}

// buildLps does a breadth-first (in order of increasing length)
// to calculate the each node's lps (longest proper suffix)
func (s *SearchTrie) buildFailureFn() {
	var nodes list.List
	for _, child := range s.children {
		nodes.PushBack(child)
	}
	for e := nodes.Front(); e != nil; e = e.Next() {
		node := e.Value.(*SearchTrie)
		for char, child := range node.children {
			node.completeFailureFn(char)
			nodes.PushBack(child)
		}
	}
}

func (s *SearchTrie) completeFailureFn(char rune) {
	if s.isRoot() {
		return
	}
	var np *SearchTrie = s.getChild(char)
	var m *SearchTrie = s
	var mp *SearchTrie

	for {
		m = m.failureFn()
		mp = m.getChild(char)
		if m.isRoot() || !mp.isRoot() {
			break
		}
	}
	np.lps = mp
	for k := range mp.ot {
		np.ot[k] = true
	}
}

func (s *SearchTrie) completeInverseFn(np *SearchTrie, char rune) {
	y := s
	for _, x := range y.ilps {
		if xp, ok := x.children[char]; ok {
			delete(xp.failureFn().ilps, xp.id)
			xp.lps = np
			np.ilps[xp.id] = xp
		} else {
			x.completeInverseFn(np, char)
		}
	}
}

// MarshalJSON writes the trie as json, for debugging purposes
func (s SearchTrie) MarshalJSON() ([]byte, error) {
	var m = make(map[string]interface{})
	var children = make(map[string]*SearchTrie)
	var ot = make([]string, len(s.ot))
	//m["char"] = string(s.char)
	m["id"] = s.id

	if nil != s.lps && !s.lps.isRoot() {
		m["lps"] = s.lps.id
	}

	if s.isWord {
		m["isWord"] = s.isWord
	}

	i := 0
	for k := range s.ot {
		ot[i] = k
		i++
	}
	m["ot"] = ot

	for k, v := range s.children {
		children[string(k)] = v
	}
	m["children"] = children

	return json.Marshal(m)
}
