package main

import (
	"runtime"
	"testing"
)

func Benchmark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := HistogramASCII("./", runtime.NumCPU())
		if err != nil {
			b.Fatal(err)
		}
	}
}
