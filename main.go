package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

func main() {
	searcher := Searcher{}
	err := searcher.Load("completeworks.txt")
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch(searcher))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Listening on port %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Searcher struct {
	CompleteWorks string
	SuffixArray    *suffixarray.Index
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}

		results := searcher.Search(query[0])
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

func (s *Searcher) Load(filename string) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	s.CompleteWorks = string(dat)
	s.SuffixArray = suffixarray.New(dat)
	return nil
}


//Finds the target string and extracts the full line of dialogue that it comes from.
func (s *Searcher) Search(query string) []string {
	idxs := s.SuffixArray.Lookup([]byte(query), -1)
	results := []string{}
	if idxs == nil {
		results = append(results, "No Results Found")
		return results
	}
	linebreakBytes := []byte("\r\n\r\n")
	var resultsFinderWG sync.WaitGroup
	resultsFinderWG.Add(len(idxs))
	for _, idx := range idxs {
		go func(s *Searcher, idx int){
			defer resultsFinderWG.Done()

			lineStart := -1
			lineEnd := -1
			searchidxstart := idx
			searchidxend := idx

			for lineStart < 0 {
				bytesToCheck := []byte(s.CompleteWorks[searchidxstart-4:searchidxstart])
				if (bytes.Contains(bytesToCheck, linebreakBytes)) || (searchidxstart == 0) {
					lineStart = searchidxstart
				} else {
					searchidxstart--
				}
			}

			for lineEnd < 0 {
				bytesToCheck := []byte(s.CompleteWorks[searchidxend:searchidxend+4])
				if (bytes.Contains(bytesToCheck, linebreakBytes)) ||  (searchidxend + 1 == len(s.CompleteWorks)){
					lineEnd = searchidxend
				} else {
					searchidxend++
				}
			}
			results = append(results, s.CompleteWorks[lineStart:lineEnd])
		}(s, idx)
	}

	resultsFinderWG.Wait()
	return results
}
