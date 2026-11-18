package patterns

// ShareByBucket turns per-bucket day counts into the percent (0-100) of the
// group's days that landed in each bucket, for the buckets listed in values.
// The group total is the sum over values, so the returned shares sum to 100.
// A group with no days yields 0 for every bucket rather than NaN, so two
// groups of very different sizes can be compared on equal footing.
func ShareByBucket(counts map[int]int, values []int) map[int]float64 {
	shares := make(map[int]float64, len(values))
	total := 0
	for _, value := range values {
		total += counts[value]
	}
	for _, value := range values {
		if total == 0 {
			shares[value] = 0
			continue
		}
		shares[value] = float64(counts[value]) / float64(total) * 100
	}
	return shares
}
