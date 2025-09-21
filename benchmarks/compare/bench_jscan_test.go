//go:build jscan

package compare_test

import (
	"testing"

	"github.com/romshark/jscan"
)

// jscan: iterate tokens/values
func Benchmark_ParseOnly_jscan_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		it := jscan.Begin()
		if err := it.Feed(data); err != nil {
			b.Fatal(err)
		}
		for it.Next() {
			_ = it.Value()
		}
		if err := it.Err(); err != nil {
			b.Fatal(err)
		}
	}
}
