package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	numWorkers = 50
	bufSize    = 1024
)

type result struct {
	err   error
	ascii map[byte]int
}

func walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)

	go func() {
		defer close(paths)
		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			select {
			case paths <- path:
			case <-done:
				return errors.New("walk canceled")
			}
			return nil
		})
	}()

	return paths, errc

}

func fileRead(path string) (map[byte]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[byte]int)

	// read file by chunk
	reader := bufio.NewReader(f)
	buf := make([]byte, bufSize)
	for {
		n, err := reader.Read(buf)
		if err != nil {

			if err != io.EOF {
				return result, err
			}

			break
		}

		// increment ascii symbols
		for i := 0; i < n; i++ {
			result[buf[i]] += 1
		}

	}

	return result, nil
}

func asciiCounter(done <-chan struct{}, paths <-chan string, c chan<- result) {
	for path := range paths {
		data, err := fileRead(path)
		select {
		case c <- result{ascii: data, err: err}:
		case <-done:
			return
		}
	}
}

func HistogramASCII(root string) (map[byte]int, error) {
	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, root)
	c := make(chan result)
	var wg sync.WaitGroup

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			asciiCounter(done, paths, c)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c)
	}()

	ascii := make(map[byte]int, 256)
	for i := 0; i < 256; i++ {
		ascii[uint8(i)] = 0
	}

	for r := range c {
		if r.err != nil {
			return nil, r.err
		}

		for ch := range r.ascii {
			ascii[ch] += r.ascii[ch]
		}
	}

	if err := <-errc; err != nil {
		return nil, err
	}
	return ascii, nil
}

func main() {

	ascii, err := HistogramASCII(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(ascii)
}
