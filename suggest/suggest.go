package suggest

import "github.com/agext/levenshtein"

// String suggests a string that closely matches one of the candidates.
//
//   The maximum difference depends on the input string. Users of the package
//   should not rely on this heuristic as it may change.
//
// If no close match is found, an empty string is returned.
func String(want string, candidates []string) string {
	// Maximum characters that can differ
	maxDist := len(want) / 5
	if maxDist == 0 {
		maxDist = 1
	}

	var str string
	dist := maxDist + 1

	for _, cand := range candidates {
		if want == cand {
			// Exact match.
			return want
		}
		d := levenshtein.Distance(want, cand, nil)
		if d < dist {
			str = cand
			dist = d
		}
	}

	if dist > maxDist {
		// No match within the maximum distance.
		return ""
	}

	return str
}
