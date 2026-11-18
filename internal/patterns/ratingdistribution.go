package patterns

import "sort"

// RatingDistributionBucket is how many entries picked one rating.
type RatingDistributionBucket struct {
	Value   int     `json:"value"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// ComputeRatingDistribution counts entries per rating. It returns one bucket
// per value in validValues, ascending (zero-count ratings included). Values
// not in validValues are ignored. Percent is of the entries counted
// (0 when there are none).
func ComputeRatingDistribution(values []int, validValues []int) []RatingDistributionBucket {
	seen := map[int]bool{}
	sorted := make([]int, 0, len(validValues))
	for _, v := range validValues {
		if !seen[v] {
			seen[v] = true
			sorted = append(sorted, v)
		}
	}
	sort.Ints(sorted)

	counts := map[int]int{}
	total := 0
	for _, v := range values {
		if seen[v] {
			counts[v]++
			total++
		}
	}

	buckets := make([]RatingDistributionBucket, 0, len(sorted))
	for _, v := range sorted {
		b := RatingDistributionBucket{Value: v, Count: counts[v]}
		if total > 0 {
			b.Percent = float64(b.Count) / float64(total) * 100
		}
		buckets = append(buckets, b)
	}
	return buckets
}
