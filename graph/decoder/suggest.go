package decoder

import (
	"sort"

	"github.com/agext/levenshtein"
	"github.com/func/func/graph"
)

// Suggest suggest the name of a provisioner that closely matches the requested
// name. Returns an empty string if no match was found.
func suggestResource(want string, ctx *graph.DecodeContext) string {
	maxdist := 5

	type suggestion struct {
		str  string
		dist int
	}

	var list []suggestion
	for name := range ctx.Resources {
		dist := levenshtein.Distance(want, name, nil)
		if dist <= maxdist {
			list = append(list, suggestion{str: name, dist: dist})
		}
	}

	if len(list) == 0 {
		return ""
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].dist < list[j].dist
	})

	return list[0].str
}
