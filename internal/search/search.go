// Package search provides substring filtering and fuzzy key suggestions.
package search

import "strings"

// Filter returns the subset of keys that contain term (case-insensitive).
func Filter(term string, keys []string) []string {
	term = strings.ToLower(term)
	var out []string
	for _, k := range keys {
		if strings.Contains(strings.ToLower(k), term) {
			out = append(out, k)
		}
	}
	return out
}

type candidate struct {
	key      string
	distance int
	substr   bool
}

// Suggest returns up to maxResults keys from candidates that are closest to
// query, ranked by a combined score: substring matches first, then by
// Levenshtein distance. Candidates with a distance greater than the adaptive
// threshold are excluded.
func Suggest(query string, candidates []string, maxResults int) []string {
	if len(candidates) == 0 {
		return nil
	}

	q := strings.ToLower(query)
	threshold := adaptiveThreshold(query)

	var ranked []candidate
	for _, c := range candidates {
		cl := strings.ToLower(c)
		sub := strings.Contains(cl, q) || strings.Contains(q, cl)
		dist := levenshtein(q, cl)
		if sub || dist <= threshold {
			ranked = append(ranked, candidate{c, dist, sub})
		}
	}

	// Sort: substring matches first, then by ascending distance.
	sortCandidates(ranked)

	out := make([]string, 0, maxResults)
	for i := 0; i < len(ranked) && i < maxResults; i++ {
		out = append(out, ranked[i].key)
	}
	return out
}

// adaptiveThreshold allows more distance for longer queries.
func adaptiveThreshold(query string) int {
	switch {
	case len(query) <= 4:
		return 1
	case len(query) <= 8:
		return 2
	default:
		return 3
	}
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two rows to keep memory O(min(la,lb)).
	if la < lb {
		ra, rb = rb, ra
		la, lb = lb, la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			if ra[i-1] == rb[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = 1 + min3(prev[j], curr[j-1], prev[j-1])
			}
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func sortCandidates(s []candidate) {
	// Insertion sort — candidate list is always small.
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && candidateLess(s[j], s[j-1]); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func candidateLess(a, b candidate) bool {
	if a.substr != b.substr {
		return a.substr // substring matches rank higher
	}
	return a.distance < b.distance
}
