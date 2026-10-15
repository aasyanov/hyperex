package hyperex

import "sort"

func sortStrings(x []string) {
	sort.Strings(x)
}

func sortSlice(x any, less func(i, j int) bool) {
	sort.Slice(x, less)
}
