package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	bufSize = 1024
)

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

func fileRead(path string) (<-chan []byte, <-chan error) {
	out := make(chan []byte)
	errc := make(chan error)

	go func(path string) {
		defer close(out)

		f, err := os.Open(path)
		if err != nil {
			errc <- err
			return
		}

		defer func() {
			_ = f.Close()
		}()

		// read file by chunk
		reader := bufio.NewReader(f)
		buf := make([]byte, bufSize)

		for {
			_, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					errc <- err
				}

				break

			}

			out <- buf
		}
	}(path)

	return out, errc
}

func asciiCounter(done <-chan struct{}, paths <-chan string, c chan<- []byte) {
	for path := range paths {
		func(path string) {
			out, errc := fileRead(path)
			for {
				select {
				case o, ok := <-out:
					if !ok {
						return
					}
					c <- o
				case err := <-errc:
					fmt.Println(err)
					return
				case <-done:
					return
				}
			}
		}(path)
	}

}

func HistogramASCII(root string) (map[byte]int, error) {
	done := make(chan struct{})
	defer close(done)

	numWorkers := runtime.NumCPU()

	paths, errc := walkFiles(done, root)
	c := make(chan []byte)
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
		for _, ch := range r {
			ascii[ch] += 1
		}
	}

	if err := <-errc; err != nil {
		return nil, err
	}

	return ascii, nil
}

func main() {

	result, err := HistogramASCII(os.Args[1])
	if err != nil {
		panic(err)
		return
	}

	fmt.Println(result)

}
