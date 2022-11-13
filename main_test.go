package main

import (
	"testing"
)

func Benchmark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := HistogramASCII("./")
		if err != nil {
			b.Fatal(err)
		}
	}
}
