package btree

import (
	"encoding/binary"
	"testing"
)

func TestInitializeBtree(t *testing.T) {
	tree, err := NewBTree("test.bin")
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	r, err := tree.pm.read(tree.root)
	if err != nil {
		t.Fatal(err.Error())
	}

	ntype := binary.BigEndian.Uint16(r[0:])
	if ntype != NODE_TYPE_LEAF {
		t.Fatal("root should've been a leaf page the very first time")
	}
}
