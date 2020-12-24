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
    "regexp"
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
	Sonnets string
    Plays   string
	SonnetSuffixArray    *suffixarray.Index
    PlaySuffixArray      *suffixarray.Index
}

type SafeMap struct{
    mut sync.Mutex
    resSet map[string]bool
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}

		Sonnetresults := searcher.SonnetSearch(query[0])
        Playresults   := searcher.PlaySearch(query[0])

        results := append([]string{"SONNET RESULTS \n\n"}, Sonnetresults...)
        results = append(results,  []string{"\n\n PLAY RESULTS \n\n"}...)
        results = append(results, Playresults...)
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


		// sBuf := &bytes.Buffer{}
		// sEnc := json.NewEncoder(sBuf)
		// sErr := sEnc.Encode(Sonnetresults)

  //       pBuf := &bytes.Buffer{}
  //       pEnc := json.NewEncoder(pBuf)
  //       pErr := pEnc.Encode(Playresults)
		// if pErr != nil || sErr != nil {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte("encoding failure"))
		// 	return
		// }
		// w.Header().Set("Content-Type", "application/json")
  //       w.Write([]byte("SONNET RESULTS \n\n"))
		// w.Write(sBuf.Bytes())
  //       w.Write([]byte("\n\n PLAY RESULTS \n\n"))
  //       w.Write(pBuf.Bytes())
	}
}

func (s *Searcher) Load(filename string) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}

    //preprocessing step
    completeWorks := string(dat)
    slicingSuffixArr := suffixarray.New(dat)
    sonnetLocs := slicingSuffixArr.Lookup([]byte("THE SONNETS"), -1) //There will only be 2 of these, kinda hacky approach
    sonnetEnd := slicingSuffixArr.Lookup([]byte("THE END"), -1) //Will only be 1 of these; also very hacky
    playEnd := slicingSuffixArr.Lookup([]byte("FINIS"), -1) //Will only be 1 of these; also very hacky
    var sonnetStart int
    if sonnetLocs[0] > sonnetLocs[1] {
        sonnetStart = sonnetLocs[0]
    } else {
        sonnetStart = sonnetLocs[1]
    }

    //setting up searcher
	s.Sonnets = completeWorks[sonnetStart:sonnetEnd[0]+7]
	s.SonnetSuffixArray = suffixarray.New([]byte(completeWorks[sonnetStart:sonnetEnd[0]+7]))

    s.Plays = completeWorks[sonnetEnd[0]+7:playEnd[0]+5]
    s.PlaySuffixArray = suffixarray.New([]byte(completeWorks[sonnetEnd[0]+7:playEnd[0]+5]))
	return nil
}


func (s *Searcher) SonnetSearch(query string) []string {
    return s.Search(query, "[0-9]+\r\n\r\n", true)
}

func (s *Searcher) PlaySearch(query string) []string {
    return s.Search(query, "\r\n\r\n[A-Z]+.", false)
}

//Finds the target string and extracts the full line of dialogue that it comes from.
func (s *Searcher) Search(query string, lineBreakRegex string, isSonnet bool) []string {
    var works string
    var idxs []int
    var windowSize int
    if isSonnet {
        works = s.Sonnets
        windowSize = 5 //basically magic, could be more robust. Equivalent to "1\r\n\r\n", the smallest header in the sonnets
        idxs  = s.SonnetSuffixArray.Lookup([]byte(query), -1)
    } else{
        works = s.Plays
        windowSize = 6 //basically magic, could be more robust.  Equivalent to "\r\n\r\n[A-Z].", the smallest predecessor for a play dialogue
        idxs = s.PlaySuffixArray.Lookup([]byte(query), -1)
    }

    results := []string{}
    if idxs == nil {
        results = append(results, "No Results Found")
        return results
    }

    //New Set of strings to help with de-duplication
    safemap := SafeMap{}
    safemap.resSet = make(map[string]bool)

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
                bytesToCheck := []byte(works[searchidxstart-windowSize:searchidxstart])
                matchRes, _ := regexp.Match(lineBreakRegex, bytesToCheck)
                if (matchRes || searchidxstart == 0) {
                    if isSonnet {
                        lineStart = searchidxstart
                    } else {
                        lineStart = searchidxstart - windowSize
                    }
                    
                } else {
                    searchidxstart--
                }
            }

            for lineEnd < 0 {
                bytesToCheck := []byte(works[searchidxend:searchidxend+windowSize])
                matchRes, _ := regexp.Match(lineBreakRegex, bytesToCheck)
                if (matchRes ||  searchidxend + 1 == len(works)){
                    if isSonnet {
                        lineEnd = searchidxend - windowSize
                    } else {
                        lineEnd = searchidxend
                    }
                    
                } else {
                    searchidxend++
                }
            }
            safemap.mut.Lock()
            safemap.resSet[works[lineStart:lineEnd]] = true
            safemap.mut.Unlock()
        }(s, idx)
    }

	resultsFinderWG.Wait()
    for key, _ := range safemap.resSet {
        results = append(results, key)
    }
	return results
}


