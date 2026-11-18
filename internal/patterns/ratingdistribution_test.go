package patterns

import (
	"math"
	"testing"
)

func distBucket(v, c int, p float64) RatingDistributionBucket {
	return RatingDistributionBucket{Value: v, Count: c, Percent: p}
}

func TestComputeRatingDistribution(t *testing.T) {
	cases := []struct {
		name   string
		values []int
		valid  []int
		want   []RatingDistributionBucket
	}{
		{"empty input keeps zero buckets", nil, []int{1, 2, 3}, []RatingDistributionBucket{distBucket(1, 0, 0), distBucket(2, 0, 0), distBucket(3, 0, 0)}},
		{"counts and percents", []int{3, 3, 3, 1}, []int{1, 2, 3}, []RatingDistributionBucket{distBucket(1, 1, 25), distBucket(2, 0, 0), distBucket(3, 3, 75)}},
		{"ascending even if valid is unsorted", []int{2}, []int{3, 1, 2}, []RatingDistributionBucket{distBucket(1, 0, 0), distBucket(2, 1, 100), distBucket(3, 0, 0)}},
		{"unknown values ignored and excluded from total", []int{9, 1, 0}, []int{1, 2}, []RatingDistributionBucket{distBucket(1, 1, 100), distBucket(2, 0, 0)}},
		{"duplicate valid values collapse", []int{1}, []int{1, 1}, []RatingDistributionBucket{distBucket(1, 1, 100)}},
		{"no valid ratings", []int{1}, nil, []RatingDistributionBucket{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeRatingDistribution(tc.values, tc.valid)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i].Value != tc.want[i].Value || got[i].Count != tc.want[i].Count || math.Abs(got[i].Percent-tc.want[i].Percent) > 1e-9 {
					t.Fatalf("bucket %d: got %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
