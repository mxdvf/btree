package btree

import (
	"fmt"
	"testing"
)

func TestNodeNKeys(t *testing.T) {
	buf := make([]byte, 4096)
	n := NewNode(buf)

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

func TestNodeLeafNodeInsert1(t *testing.T) {
	buf := make([]byte, 4096)
	n := NewNode(buf)

	k, v := []byte("ducky-24"), []byte("mehul")
	_, err := n.insertSelf(k, v)
	if err != nil {
		t.Fatalf("got an error on insertion: %v", err)
	}

	k1, v1 := n.getKV(0)
	if string(k) != string(k1) || string(v) != string(v1) {
		t.Fatalf("first kv mismatch")
	}
}

func TestNodeLeafNodeInsert2(t *testing.T) {
	buf := make([]byte, 4096)
	n := NewNode(buf)
	_, err := n.insertSelf([]byte("ducky"), []byte("mehul"))
	if err != nil {
		t.Fatalf("got an error on insertion: %v", err)
	}

	_, err = n.insertSelf([]byte("ducky11"), []byte("mehul11"))
	if err != nil {
		t.Fatalf("got an error on second insertion: %v", err)
	}

	k, v := n.getKV(0)
	if string(k) != "ducky" || string(v) != "mehul" {
		t.Fatalf("first kv mismatch")
	}
	k, v = n.getKV(1)
	if string(k) != "ducky11" || string(v) != "mehul11" {
		t.Fatalf("second kv mismatch")
	}
}

func TestNodeLeafNodeInsert3(t *testing.T) {
	buf := make([]byte, 4096)
	n := NewNode(buf)
	for i := range 174 {
		k, v := []byte(fmt.Sprintf("ducky-%d", i)), []byte("mehul")
		_, err := n.insertSelf(k, v)
		if err != nil {
			t.Fatalf("got an error on insertion: %v", err)
			break
		}
	}

	t.Logf("node is filled to %v/%v bytes\n", n.getSize(), PageSize)
	k1, v1 := []byte("ducky-175"), []byte("mehul")
	t.Logf("and about to insert a kv pair of post-insert size: %v\n", n.getTotalLenIfInserted(k1, v1))

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				// check it's the right panic
				if msg, ok := r.(string); ok && msg == "illegal node, it should have been split by a preemptive fix" {
					panicked = true
				} else {
					t.Errorf("wrong panic: %v", r)
				}
			}
		}()
		n.insertSelf(k1, v1)
	}()

	if !panicked {
		t.Fatal("expected a panic on overflow but got none")
	}
}

// func TestNodeDeleteAt(t *testing.T) {
// 	buf := make([]byte, 4096)
// 	n := NewNode(buf)
// 	keys := []string{"apple", "banana", "cherry", "date", "elderberry"}
// 	for _, k := range keys {
// 		n.insertSelf([]byte(k), []byte("val"))
// 	}
// 	// delete middle key
// 	n.deleteAt(1) // should remove "cherry"

// 	if n.getNKeys() != 4 {
// 		t.Fatalf("expected 4 keys, got %d", n.getNKeys())
// 	}
// 	k, _ := n.getKV(2)
// 	if string(k) != "date" {
// 		t.Fatalf("expected 'date' at idx 2, got %s", k)
// 	}
// }
