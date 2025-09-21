//go:build jstream

package compare_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/bcicen/jstream"
)

// jstream: stream array elements without building the whole slice
func Benchmark_ParseOnly_jstream_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec := jstream.NewDecoder(bytes.NewReader(data), 1)
		for mv := range dec.Stream() {
			if mv.Value == nil {
				b.Fatal("nil")
			}
		}
		// drain the decoder error if any
		if err := dec.Err(); err != nil && err != io.EOF {
			b.Fatal(err)
		}
	}
}
