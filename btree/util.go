package btree

// calculateAppropriateIdx returns the index in the sorted slice for
// for which k < keys[idx], which means to find the appropriate position
// in the slice if it were to be inserted
func calculateAppropriateIdx(keys []uint16, k uint16) int {
	var idx int
	for idx = 0; idx < len(keys); idx++ {
		if k <= keys[idx] {
			break
		}
	}
	return idx
}
