package btree

type pageManager struct {
}

// 3 jobs as fucking simple as that:
// read: given a page number, return the 4096 bytes
// write: given a page number and 4096 bytes, it just persists
// allocate: create a new empty page and return the page number

func (pm *pageManager) hello() {}
