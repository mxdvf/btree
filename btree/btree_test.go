package btree

import (
	"testing"
)

func TestSimple(t *testing.T) {
	n := NewLeafNode()
	n.Insert([]byte("kackyA"), []byte("你"))
	n.Insert([]byte("z"), []byte("A"))
	n.debugPrint()
}
