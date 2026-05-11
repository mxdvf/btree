package btree

import (
	"fmt"
	"testing"
)

func TestNKeys(t *testing.T) {
	n := NewLeafNode()

	k := n.getNKeys()
	if k != 0 {
		t.Fatal("nkeys should have been 0")
	}

	n.incrementNKeys()
	k = n.getNKeys()
	if k != 1 {
		t.Fatal("nkeys should have been 1")
	}
}

func TestLeafNodeInsert1(t *testing.T) {
	n := NewLeafNode()
	err := n.Insert([]byte("kacky-24"), []byte("mehul"))
	if err != nil {
		t.Fatalf("got an error on insertion: %v", err)
	}
	debugPrint(n, 60)
}

func TestLeafNodeInsert2(t *testing.T) {
	n := NewLeafNode()
	for i := range 100 {
		k := fmt.Sprintf("kacky-%d", i)
		err := n.Insert([]byte(k), []byte("mehul"))
		if err != nil {
			t.Fatalf("got an error on insertion: %v", err)
			break
		}
	}
}
