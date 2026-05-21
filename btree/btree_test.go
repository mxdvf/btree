package btree

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func init() {
	err := os.MkdirAll("test/", 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func TestBtreeInitialize(t *testing.T) {
	tree, err := NewBTree("test/test.bin")
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

func TestBtreeSimpleInsert1(t *testing.T) {
	filename := fmt.Sprintf("test/test-%v.bin", rand.Int())
	t.Logf("running test case for file: %v", filename)
	tree, err := NewBTree(filename)
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	k := []byte("kacky")
	v := []byte("mehul")
	if err = tree.Insert(k, v); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	t.Log(tree.root)
	buf, _ := tree.pm.read(tree.root)
	t.Log(buf[:100])
}

func TestBtreeSimpleInsert2(t *testing.T) {
	filename := fmt.Sprintf("test/test-%v.bin", rand.Int())
	t.Logf("running test case for file: %v", filename)
	tree, err := NewBTree(filename)
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	buf, err := tree.pm.read(tree.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("looking at root(%v) before inserting: %v", tree.root, buf[:100])

	k := []byte("kacky")
	v := []byte("mehul")
	if err = tree.Insert(k, v); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	buf, err = tree.pm.read(tree.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("looking at root(%v) after inserting one pair: %v", tree.root, buf[:100])

	k = []byte("kacky11")
	v = []byte("mehul11")
	if err = tree.Insert(k, v); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	buf, err = tree.pm.read(tree.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("looking at root(%v) after inserting two pairs: %v", tree.root, buf[:100])
}

func TestBtreeNonSplitMultipleKeys(t *testing.T) {
	filename := fmt.Sprintf("test/test-%v.bin", rand.Int())
	t.Logf("running test case for file: %v", filename)
	tree, err := NewBTree(filename)
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	for i := range 174 {
		k := fmt.Sprintf("kacky-%d", i)
		err := tree.Insert([]byte(k), []byte("mehul"))
		if err != nil {
			t.Fatalf("got an error on insertion: %v", err)
			break
		}
	}
}

func TestBtreeSplitRoot(t *testing.T) {
	filename := fmt.Sprintf("test/test-%v.bin", rand.Int())
	t.Logf("running test case for file: %v", filename)
	tree, err := NewBTree(filename)
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	var keyCount uint16 = 0

	for i := range 174 {
		k := fmt.Sprintf("kacky-%d", i)
		err := tree.Insert([]byte(k), []byte("mehul"))
		if err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
		keyCount++
	}

	k1, v1 := []byte("kacky-175"), []byte("mehul")
	tree.Insert(k1, v1)
	keyCount++
	// ---- the root split has happened by now ---- //

	// reading child 1
	buf, _ := tree.pm.read(179)
	left := NewNode(buf)

	// reading child 2
	buf, _ = tree.pm.read(176)
	right := NewNode(buf)

	// naive check
	if left.getNKeys()+right.getNKeys()+1 != keyCount {
		t.Fatal("nope something went seriously wrong in between, some keys got lost")
	}
}

func TestBtreeSplitInternalNode(t *testing.T) {
	filename := fmt.Sprintf("test/test-%v.bin", rand.Int())
	t.Logf("running test case for file: %v", filename)
	tree, err := NewBTree(filename)
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	for i := range 174 {
		k := fmt.Sprintf("kacky-%d", i)
		err := tree.Insert([]byte(k), []byte("mehul"))
		if err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	k1, v1 := []byte("kacky-175"), []byte("mehul")
	tree.Insert(k1, v1)
	// ---- root split happened here ---- //

	// buf, _ := tree.pm.read(tree.root)
	// root := NewNode(buf)
	// fmt.Println(root.data)

	// // reading child 1
	// buf, _ = tree.pm.read(179)
	// left := NewNode(buf)
	// fmt.Println(left.data)

	// // reading child 2
	// buf, _ = tree.pm.read(176)
	// right := NewNode(buf)
	// fmt.Println(right.data)

	for i := range 91 {
		k := fmt.Sprintf("backy-%d", i)
		err := tree.Insert([]byte(k), []byte("mehul"))
		if err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	k1, v1 = []byte("a"), []byte("mehul")
	err = tree.Insert(k1, v1)
	if err != nil {
		t.Log("lemme have a look")
	}

	k1, v1 = []byte("a2"), []byte("mehul")
	err = tree.Insert(k1, v1)
	if err != nil {
		t.Log("lemme have a look")
	}

	// TODO: fix this test

	buf, _ := tree.pm.read(tree.root)
	root := NewNode(buf)
	fmt.Println(root.data)

	// reading child 1
	buf, _ = tree.pm.read(367)
	left := NewNode(buf)
	fmt.Println(left.data)

	// reading child 2
	buf, _ = tree.pm.read(363)
	right := NewNode(buf)
	fmt.Println(right.data)
}
