package compare

// ElementsMatch asserts that the specified x is equal to specified y
// ignoring the order of the elements.
//
// If there are duplicate elements, the number of appearances of each of them
// in both lists should match.
//
//	compare.ElementsMatch([1, 3, 2, 3], [1, 3, 3, 2])
func ElementsMatch(x, y []int64) bool {
	if len(x) != len(y) {
		return false
	}

	diff := make(map[int64]int, len(x))
	for _, _x := range x {
		diff[_x]++
	}

	for _, _y := range y {
		if _, ok := diff[_y]; !ok {
			return false
		}

		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}

	if len(diff) == 0 {
		return true
	}

	return false
}
