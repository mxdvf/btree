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
	if t != nil {
		t.Logf("running test case for file: %v", filename)
	}
	tree, err := NewBTree(filename, false)
	if err != nil {
		t.Fatalf("cannot initialize tree: %v", err)
	}

	return tree
}

func TestBtreeInitialize(t *testing.T) {
	tree := setup(t)

	r, err := tree.pm.Read(tree.root)
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

	buf, _ := tree.pm.Read(tree.root)
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

	buf, _ := tree.pm.Read(tree.root)
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

	buf, _ = tree.pm.Read(tree.root)
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

func TestBtreeFillUntilRootSplits1Level(t *testing.T) {
	tree := setup(t)

	kNums := []string{"10", "15", "20", "21", "16", "12", "2"}

	for _, kNum := range kNums {
		k := strings.Repeat("A", 1338-len(kNum)) + "_" + kNum
		if err := tree.Insert([]byte(k), []byte("mehul")); err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	for _, kNum := range kNums {
		k := strings.Repeat("A", 1338-len(kNum)) + "_" + kNum
		v, err := tree.Search([]byte(k))
		if err != nil {
			t.Fatal(err)
		}
		if v == nil {
			t.Fatalf("value exists because the key has been inserted: %v", k)
		}
	}
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

	for _, kNum := range kNums {
		k := strings.Repeat("A", 1338-len(kNum)) + "_" + kNum
		v, err := tree.Search([]byte(k))
		if err != nil {
			t.Fatal(err)
		}
		if v == nil {
			t.Fatalf("value exists because the key has been inserted: %v", k)
		}
	}
}

func TestBtreeUnboundedInsert(t *testing.T) {
	tree := setup(t)

	for i := range 10000 {
		k := strings.Repeat("A", 1338-len(strconv.Itoa(i))) + "_" + strconv.Itoa(i)
		if err := tree.Insert([]byte(k), []byte("mehul")); err != nil {
			t.Fatalf("got an error on insertion: %v", err)
		}
	}

	for i := range 10000 {
		k := strings.Repeat("A", 1338-len(strconv.Itoa(i))) + "_" + strconv.Itoa(i)
		v, err := tree.Search([]byte(k))
		if err != nil {
			t.Fatal(err)
		}
		if v == nil {
			t.Fatalf("value exists because the key has been inserted: %v", k)
		}
	}
}

func BenchmarkInsert(b *testing.B) {
	tr := setup(nil)
	for i := 0; i < b.N; i++ {
		k, v := []byte(fmt.Sprintf("kacky-%v", i)), []byte("mehul")
		if err := tr.Insert(k, v); err != nil && err != ErrOverflow {
			b.Fatal("insertion failed: %w", err)
		}
	}
}
