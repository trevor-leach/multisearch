# multisearch

A library and command-line tool for efficiently searching a text for a number of search terms.


## Command-Line Usage

```console
user:~$ multisearch --help
Multisearch searches for multiple terms in some text.

Usage:

    multisearch [--termfile path]... [--searchpath path [-r]] [-i] [search_term...]

Options:

  -help
        Print this help text and exit.
  -i    Perform a case-insensitive search
  -output string
        File where results are written.  Defaults to stdout.
  -r    Search all subdirectories of searchpath.
  -searchpath string
        File in which to search.  If a directory, contained files are searched.
  -termfile value
        File containing search terms, one per line.  May be specified multiple times.
```

If no searchpath is specified, then stdin is searched.

### Examples

Searching stdin for individually specified terms:
```console
user:~$ echo "you can do it" | multisearch -i A An Can
Term    Start   End
a       5       6
an      5       7
can     4       7
```

Searching all files in a directory for terms from a file:
```console
user:~$ multisearch --termfile ./terms.txt --searchpath ./someShakespeare/
someShakespeare/The Comedy of Errors.txt   wayward 74608   74615
someShakespeare/The Two Gentlemen of Verona.txt    wayward 21168   21175
someShakespeare/Macbeth.txt        yawn    54318   54322
someShakespeare/Macbeth.txt        wayward 64737   64744
someShakespeare/Macbeth.txt        usurper 110335  110342
someShakespeare/Much Ado about Nothing.txt wayward 35020   35027
someShakespeare/Much Ado about Nothing.txt yawn    130285  130289
someShakespeare/As You Like It.txt usurper 9800    9807
someShakespeare/As You Like It.txt usurper 45163   45170
someShakespeare/As You Like It.txt quintessence    77033   77045
someShakespeare/As You Like It.txt wayward 113714  113721
someShakespeare/Twelfth Night.txt  quaffing        22130   22138
someShakespeare/Twelfth Night.txt  yeoman  63370   63376
someShakespeare/Titus Andronicus.txt       younglings      99058   99068
someShakespeare/Romeo and Juliet.txt       wayward 117787  117794
```

## Api Usage

The api consists of a `Searcher` interface, which once configured with search terms, can be reused to search texts.

It returns a channel of `Match` objects, which contain the term found and its location.
```go
type Searcher interface {
	AddSearchTerm(searchTerm string)
	Search(r io.RuneReader) <-chan Match
}
type Match struct {
	Term  string
	Location [2]int
}
```

There is a single implementation `Searcher` which uses the [Aho-Corasick](https://en.wikipedia.org/wiki/Aho%E2%80%93Corasick_algorithm) algorithm:
```go
import (
	"strings"
	"github.com/trevor-leach/multisearch/api"
	"github.com/trevor-leach/multisearch/api/ahocorasick"
)
func example() {
	var searches []string = 

	var s api.Searcher = *ahocorasick.New([]string{"a", "can"})
	s.AddSearchTerm("an")

	for match := range s.Search(strings.NewReader("you can do it!")) {
		// so something with your api.Match instance
	}
}
```