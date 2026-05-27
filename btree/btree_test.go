package btree

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

func init() {
	err := os.MkdirAll("test/", 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func setup(t *testing.T) *BTree {
	filename := fmt.Sprintf("test/test-%v.bin", rand.Int())
	t.Logf("running test case for file: %v", filename)
	tree, err := NewBTree(filename)
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	return tree
}

func TestBtreeInitialize(t *testing.T) {
	tree := setup(t)

	r, err := tree.pm.read(tree.root)
	if err != nil {
		t.Fatal(err.Error())
	}

	if NewNode(r).getType() != NodeTypeLeaf {
		t.Fatal("root should've been a leaf page the very first time")
	}
}

func TestBtreeSimpleInsert1(t *testing.T) {
	tree := setup(t)

	k := []byte("ducky")
	v := []byte("mehul")
	if err := tree.Insert(k, v); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	buf, _ := tree.pm.read(tree.root)
	node := NewNode(buf)

	if node.getNKeys() != 1 {
		t.Fatal("node should have only 1 key")
	}

	k1, v1 := node.getKV(0)
	if res := bytes.Compare(k, k1); res != 0 {
		t.Fatal("keys don't match up")
	}
	if res := bytes.Compare(v, v1); res != 0 {
		t.Fatal("vals don't match up")
	}
}

func TestBtreeSimpleInsert2(t *testing.T) {
	tree := setup(t)

	k := []byte("ducky")
	v := []byte("mehul")
	if err := tree.Insert(k, v); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	buf, _ := tree.pm.read(tree.root)
	node := NewNode(buf)
	if node.getNKeys() != 1 {
		t.Fatal("node should have only 1 key")
	}
	k1, v1 := node.getKV(0)
	if res := bytes.Compare(k, k1); res != 0 {
		t.Fatal("keys don't match up")
	}
	if res := bytes.Compare(v, v1); res != 0 {
		t.Fatal("vals don't match up")
	}

	k = []byte("ducky11")
	v = []byte("mehul11")
	if err := tree.Insert(k, v); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	buf, _ = tree.pm.read(tree.root)
	node = NewNode(buf)
	if node.getNKeys() != 2 {
		t.Fatal("node should have 2 keys")
	}
	k1, v1 = node.getKV(1)
	if res := bytes.Compare(k, k1); res != 0 {
		t.Fatal("keys don't match up")
	}
	if res := bytes.Compare(v, v1); res != 0 {
		t.Fatal("vals don't match up")
	}
}

func TestBtreeFillToBrim(t *testing.T) {
	tree := setup(t)

	var buf []byte
	var node *Node

	for i := range 3 {
		k := strings.Repeat("A", 1337) + "_" + strconv.Itoa(i)
		if err := tree.Insert([]byte(k), []byte("mehul")); err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	k := "z|z|z|z|z|"
	if err := tree.Insert([]byte(k), []byte("mehulA")); err != nil {
		t.Fatalf("got an error on insertion: %v", err)
	}

	buf, _ = tree.pm.read(tree.root)
	node = NewNode(buf)
	if node.getSize() != 4096 {
		t.Fatalf("node still has space, has only occupied %v bytes", node.getSize())
	}
}

func TestBtreeFillUntilRootSplits1Level(t *testing.T) {
	tree := setup(t)

	kNums := []string{"10", "15", "20", "21", "16", "12", "2"}

	for _, kNum := range kNums {
		k := strings.Repeat("A", 1338-len(kNum)) + "_" + kNum
		if err := tree.Insert([]byte(k), []byte("mehul")); err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	tree.print()

	// TODO: search for these keys as well
}

func TestBtreeFillUntilRootSplits2Level(t *testing.T) {
	tree := setup(t)

	kNums := []string{"10", "17", "20", "21", "16", "12", "2", "1", "3", "4", "11"}
	for _, kNum := range kNums {
		k := strings.Repeat("A", 1338-len(kNum)) + "_" + kNum
		if err := tree.Insert([]byte(k), []byte("mehul")); err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	tree.print()

	// TODO: search for these keys as well
}

func TestBtreeUnboundedInsert(t *testing.T) {
	tree := setup(t)

	for i := range 10000 {
		k := strings.Repeat("A", 1338-len(strconv.Itoa(i))) + "_" + strconv.Itoa(i)
		if err := tree.Insert([]byte(k), []byte("mehul")); err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	tree.print()

	// TODO: search for these keys as well
}
