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
	_, err := n.insert(k, v)
	if err != nil {
		t.Fatalf("got an error on insertion: %v", err)
	}

	t.Log(debugPrint(n, 100))

	k1, v1 := n.getKV(0)
	if string(k) != string(k1) || string(v) != string(v1) {
		t.Fatalf("first kv mismatch")
	}
}

func TestNodeLeafNodeInsert2(t *testing.T) {
	buf := make([]byte, 4096)
	n := NewNode(buf)
	_, err := n.insert([]byte("ducky"), []byte("mehul"))
	if err != nil {
		t.Fatalf("got an error on insertion: %v", err)
	}

	t.Log(debugPrint(n, 100))

	_, err = n.insert([]byte("ducky11"), []byte("mehul11"))
	if err != nil {
		t.Fatalf("got an error on second insertion: %v", err)
	}

	t.Log(debugPrint(n, 100))

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
		_, err := n.insert(k, v)
		if err != nil {
			t.Fatalf("got an error on insertion: %v", err)
			break
		}
	}

	t.Logf("node is filled to %v bytes\n", n.getSize())
	k1, v1 := []byte("ducky-175"), []byte("mehul")
	t.Logf("and about to insert a kv pair of post-insert size: %v\n", n.getTotalLenPostInsert(k1, v1))

	_, err := n.insert(k1, v1)
	if err == nil {
		t.Fatalf("should've thrown an overflow error: %v", err)
	}
}
