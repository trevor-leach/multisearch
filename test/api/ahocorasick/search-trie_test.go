package ahocorasick

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/trevor-leach/multisearch/api"
	"github.com/trevor-leach/multisearch/api/ahocorasick"
)

func TestNew(t *testing.T) {
	var text string = "you can do it!"
	var searches []string = []string{"an", "a", "can"}
	expected := map[api.Match]bool{
		{Term: "a", Location: [2]int{5, 6}}:   true,
		{Term: "an", Location: [2]int{5, 7}}:  true,
		{Term: "can", Location: [2]int{4, 7}}: true,
	}

	var s api.Searcher = *ahocorasick.New(searches[1:])
	s.AddSearchTerm(searches[0])

	if s == nil {
		t.Errorf("Couldn't create new ahocorasick Searcher implementation")
	}

	b, _ := json.MarshalIndent(s, "", "  ")
	t.Logf("%s", b)

	for m := range s.Search(strings.NewReader(text)) {
		ms, _ := json.Marshal(m)
		if _, ok := expected[m]; !ok {
			t.Errorf("unexpected match: %s", ms)
		} else {
			t.Logf("%s", ms)
		}
		delete(expected, m)
	}

	for m := range expected {
		ms, _ := json.Marshal(m)
		t.Errorf("Missed expected match: %s", ms)
	}
}
