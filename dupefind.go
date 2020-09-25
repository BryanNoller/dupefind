package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	wg       sync.WaitGroup
	hashList = make(map[string][]string)
	dupeList = make(map[string]interface{})
	lock     sync.Mutex
)

func worker(pathChan <-chan string) {
	defer wg.Done()

	hash := sha256.New()
	for path := range pathChan {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			continue
		}

		hash.Reset()
		if _, err := io.Copy(hash, f); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			sum := string(hash.Sum(nil))
			lock.Lock()
			if _, ok := hashList[sum]; ok {
				hashList[sum] = append(hashList[sum], path)
				dupeList[sum] = nil
			} else {
				hashList[sum] = []string{path}
			}
			lock.Unlock()
		}
		f.Close()
	}
}

func main() {
	pathChan := make(chan string, 100)
	for i := 0; i < runtime.NumCPU()*5; i++ {
		wg.Add(1)
		go worker(pathChan)
	}

	searchPath := "."

	if len(os.Args) > 1 {
		searchPath = os.Args[1]
	}

	if err := filepath.Walk(searchPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			if f.Mode().IsRegular() && f.Size() > 0 {
				pathChan <- path
			}
		}
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	close(pathChan)
	wg.Wait()

	for sum := range dupeList {
		fmt.Println(hashList[sum])
	}
	fmt.Printf("%d Files, %d Duplicates\n", len(hashList), len(dupeList))
}
