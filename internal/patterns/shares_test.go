package patterns

import (
	"math"
	"testing"
)

func shareCounts(pairs ...int) map[int]int {
	counts := map[int]int{}
	for i := 0; i+1 < len(pairs); i += 2 {
		counts[pairs[i]] = pairs[i+1]
	}
	return counts
}

func TestShareByBucket(t *testing.T) {
	tests := []struct {
		name   string
		counts map[int]int
		values []int
		want   map[int]float64
	}{
		{"empty group is all zero", shareCounts(), []int{1, 2, 3}, map[int]float64{1: 0, 2: 0, 3: 0}},
		{"single bucket is 100", shareCounts(2, 5), []int{1, 2, 3}, map[int]float64{1: 0, 2: 100, 3: 0}},
		{"even split", shareCounts(1, 2, 3, 2), []int{1, 2, 3}, map[int]float64{1: 50, 2: 0, 3: 50}},
		{"thirds", shareCounts(1, 1, 2, 1, 3, 1), []int{1, 2, 3}, map[int]float64{1: 100.0 / 3, 2: 100.0 / 3, 3: 100.0 / 3}},
		{"group size does not matter", shareCounts(1, 20, 3, 80), []int{1, 3}, map[int]float64{1: 20, 3: 80}},
		{"counts outside values are ignored", shareCounts(1, 1, 9, 50), []int{1, 2}, map[int]float64{1: 100, 2: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShareByBucket(tt.counts, tt.values)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, w := range tt.want {
				if math.Abs(got[k]-w) > 1e-9 {
					t.Errorf("bucket %d: got %v, want %v", k, got[k], w)
				}
			}
		})
	}
}
