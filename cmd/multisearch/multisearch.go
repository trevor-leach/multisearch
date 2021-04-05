package main

import (
	"bufio"
	//"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/trevor-leach/multisearch/api"
	"github.com/trevor-leach/multisearch/api/ahocorasick"
)

type lowerRuneReader struct {
	rr io.RuneReader
}

func (lrr *lowerRuneReader) ReadRune() (r rune, size int, err error) {
	r, size, err = lrr.rr.ReadRune()
	if nil == err {
		r = unicode.ToLower(r)
	}
	return r, size, err
}

type sliceStr []string

func (i *sliceStr) String() string {
	return strings.Join(*i, " ")
}

func (i *sliceStr) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func usage() {
	fmt.Print("Multisearch searches for multiple terms in some text.\n\n")
	fmt.Print("Usage:\n\n    multisearch [--termfile path]... [--searchpath path [-r]] [-i] [search_term...]\n\n")
	fmt.Print("Options:\n\n")
	flag.PrintDefaults()
}

func main() {
	var terms []string
	var termFiles sliceStr
	var searchFile string
	var outFile string
	var recursive bool
	var help bool
	var caseInsensitive bool

	flag.Var(&termFiles, "termfile", "File containing search terms, one per line.  May be specified multiple times.")
	flag.StringVar(&searchFile, "searchpath", "", "File in which to search.  If a directory, contained files are searched.")
	flag.StringVar(&outFile, "output", "", "File where results are written.  Defaults to stdout.")
	flag.BoolVar(&recursive, "r", false, "Search all subdirectories of searchpath.")
	flag.BoolVar(&help, "help", false, "Print this help text and exit.")
	flag.BoolVar(&caseInsensitive, "i", false, "Perform a case-insensitive search")
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}

	for _, termfile := range termFiles {
		f, err := os.Open(termfile)
		if nil != err {
			fmt.Fprintf(os.Stderr, "opening term file %q: %e\n", termfile, err)
			os.Exit(1)
		}
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			term := strings.TrimSpace(scanner.Text())
			if "" == term {
				continue
			}
			if caseInsensitive {
				term = strings.ToLower(term)
			}
			terms = append(terms, term)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "reading term file %q: %e\n", termfile, err)
			os.Exit(1)
		}
		f.Close()
	}

	for _, term := range flag.Args() {
		term = strings.TrimSpace(term)
		if "" == term {
			continue
		}
		if caseInsensitive {
			term = strings.ToLower(term)
		}
		terms = append(terms, term)
	}

	if 0 == len(terms) {
		fmt.Fprintln(os.Stderr, "no search terms specified")
		usage()
		os.Exit(1)
	}

	var out io.Writer = os.Stdout

	if "" != outFile {
		info, err := os.Stat(outFile)
		if nil == err {
			if info.IsDir() {
				fmt.Fprintf(os.Stderr, "output file %q is a directory\n", outFile)
				os.Exit(1)
			} else {
				fmt.Fprintf(os.Stderr, "output file %q already exists\n", outFile)
				os.Exit(1)
			}
		} else if os.IsNotExist(err) {

			fout, er := os.Create(outFile)
			if nil != er {
				fmt.Fprintf(os.Stderr, "opening output file %q: %e \n", outFile, er)
				os.Exit(1)
			}
			defer fout.Close()
			out = fout
		} else {
			fmt.Fprintf(os.Stderr, "output file %q: %e \n", outFile, err)
			os.Exit(1)
		}
	}

	var searcher api.Searcher = ahocorasick.New(terms)
	var outCh = make(chan string, 16)
	var errCh = make(chan string, 16)
	var wg, wg2 sync.WaitGroup

	if "" == searchFile {
		if recursive {
			fmt.Println("\"-r\" option may only be specified along with \"-searchpath\"")
			os.Exit(1)
		}

		wg.Add(1)
		var rr io.RuneReader = bufio.NewReader(os.Stdin)
		if caseInsensitive {
			rr = &lowerRuneReader{rr: rr}
		}

		go doSearch(searcher, rr, outCh, &wg)
	} else {
		info, err := os.Stat(searchFile)
		if nil != err {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "search file %q does not exist\n", searchFile)
				os.Exit(1)
			} else {
				fmt.Fprintf(os.Stderr, "search file %q: %e \n", searchFile, err)
				os.Exit(1)
			}
		}

		if info.IsDir() {
			searchFile = filepath.Clean(searchFile)
			filepath.Walk(searchFile, func(path string, currentInfo os.FileInfo, err error) error {
				if nil != err {
					errCh <- fmt.Sprintf("%q: %e", path, err)
					return nil
				}

				if os.SameFile(info, currentInfo) {
					return nil
				}
				if currentInfo.IsDir() {
					if !recursive {
						return filepath.SkipDir
					}
					return nil
				}
				wg.Add(1)
				go doFileSearch(searcher, path, outCh, errCh, caseInsensitive, &wg)

				return nil
			})
		} else {
			fin, er := os.Open(searchFile)
			if nil != er {
				fmt.Fprintf(os.Stderr, "opening search file %q: %e \n", outFile, er)
				os.Exit(1)
			}
			defer fin.Close()
			var rr io.RuneReader = bufio.NewReader(fin)
			if caseInsensitive {
				rr = &lowerRuneReader{rr: rr}
			}
			wg.Add(1)
			go doSearch(searcher, rr, outCh, &wg)
		}
	}

	wg2.Add(1)
	go func() {
		defer wg2.Done()
		for s := range outCh {
			fmt.Fprint(out, s)
		}
		//fmt.Println("exiting for loop")
	}()

	wg2.Add(1)
	go func() {
		defer wg2.Done()
		for s := range errCh {
			fmt.Fprint(os.Stderr, s)
		}
		//fmt.Println("exiting for loop")
	}()

	wg.Wait()
	close(outCh)
	close(errCh)
	wg2.Wait()

	// m := map[string]interface{}{
	// 	"terms":           terms,
	// 	"termfiles":       termFiles.String(),
	// 	"searchFile":      searchFile,
	// 	"recursive":       recursive,
	// 	"outFile":         outFile,
	// 	"caseInsensitive": caseInsensitive,
	// }
	// o, _ := json.MarshalIndent(m, "", "  ")
	// fmt.Fprintln(out, string(o))
}

func doFileSearch(searcher api.Searcher, path string, out, errCh chan<- string, caseInsensitive bool, wg *sync.WaitGroup) {
	defer wg.Done()
	fin, er := os.Open(path)
	if nil != er {
		errCh <- fmt.Sprintf("opening search file %q: %e \n", path, er)
		return
	}
	defer fin.Close()
	var rr io.RuneReader = bufio.NewReader(fin)
	if caseInsensitive {
		rr = &lowerRuneReader{rr: rr}
	}
	//out <- fmt.Sprintln("Name\tKeyword\tStart\tEnd")
	for match := range searcher.Search(rr) {
		out <- fmt.Sprintf("%s\t%s\t%d\t%d\n", path, match.Term, match.Location[0], match.Location[1])
	}
}

func doSearch(searcher api.Searcher, in io.RuneReader, out chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	out <- fmt.Sprintln("Term\tStart\tEnd")
	for match := range searcher.Search(in) {
		out <- fmt.Sprintf("%s\t%d\t%d\n", match.Term, match.Location[0], match.Location[1])
	}
}
