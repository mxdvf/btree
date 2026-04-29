package btree

import (
	"math/rand"
	"testing"
)

func TestNew(t *testing.T) {
	degree := 3
	tr := New(degree)

	if tr == nil {
		t.Fatalf("New(%d) returned nil", degree)
	}
	if tr.degree != degree {
		t.Errorf("Expected degree %d, got %d", degree, tr.degree)
	}
	if tr.root == nil {
		t.Error("Root should not be nil after initialization")
	}
}

func TestBasicOperations(t *testing.T) {
	tr := New(3)
	keys := []uint16{10, 20, 5, 15, 30}

	for _, k := range keys {
		tr.Insert(k)
		if !tr.Search(k) {
			t.Errorf("Key %d not found after insertion", k)
		}
	}

	if tr.Search(100) {
		t.Error("Search found key 100, which was never inserted")
	}

	tr.Delete(20)
	if tr.Search(20) {
		t.Error("Key 20 still found after deletion")
	}
}

func TestSplittingAndMerging(t *testing.T) {
	tr := New(2)

	for i := uint16(1); i <= 20; i++ {
		tr.Insert(i)
		if !tr.Search(i) {
			t.Fatalf("Failed at key %d during sequential insert", i)
		}
	}

	auditTreeProperties(t, tr)

	for i := uint16(20); i >= 1; i-- {
		tr.Delete(i)
		if tr.Search(i) {
			t.Errorf("Key %d still exists after deletion", i)
		}
	}
}

func TestRandomStress(t *testing.T) {
	tr := New(3)
	count := 500
	inserted := make(map[uint16]bool)

	for i := 0; i < count; i++ {
		val := uint16(rand.Intn(65535))
		if !inserted[val] {
			tr.Insert(val)
			inserted[val] = true
		}
	}

	auditTreeProperties(t, tr)

	for k := range inserted {
		if !tr.Search(k) {
			t.Errorf("Lost key %d during stress test", k)
		}
	}
}

// auditTreeProperties is a validator that checks if the invariants of
// BTree are still maintained
func auditTreeProperties(t *testing.T, tr *BTree) {
	if tr.root == nil {
		return
	}

	height := -1
	var checkNode func(n *Node, currentDepth int)

	checkNode = func(n *Node, currentDepth int) {
		for i := 0; i < len(n.keys)-1; i++ {
			if n.keys[i] > n.keys[i+1] {
				// t.Logf("-------")
				// tr.Print()
				// t.Logf("-------")
				t.Errorf("Keys not sorted in node: %v", n.keys)
			}
		}

		if n.isLeaf {
			if height == -1 {
				height = currentDepth
			} else if height != currentDepth {
				t.Errorf("Tree is not balanced! Leaf found at depth %d, expected %d", currentDepth, height)
			}
			return
		}

		if len(n.children) != len(n.keys)+1 {
			t.Errorf("Node has %d keys but %d children", len(n.keys), len(n.children))
		}

		for _, child := range n.children {
			checkNode(child, currentDepth+1)
		}
	}

	checkNode(tr.root, 0)
}

func TestChurn(t *testing.T) {
	tr := New(3)
	for i := 0; i < 1000; i++ {
		tr.Insert(uint16(i))
	}
	for i := 0; i < 100000; i++ {
		tr.Delete(uint16(rand.Intn(1000)))
		tr.Insert(uint16(rand.Intn(1000)))
	}
	auditTreeProperties(t, tr)
}

func TestCustom(t *testing.T) {
	tr := New(2)

	keys := []uint16{20, 10, 12, 24, 6, 31, 32, 18, 26, 25, 27, 2, 48, 1, 21, 22, 4,
		5, 90, 92, 100, 102, 104, 107, 108, 110, 105, 33, 34, 35}
	for _, key := range keys {
		tr.Insert(key)
	}
	t.Log(tr)

	tr.Delete(108)
	t.Log(tr)

	// tr.Delete(100)
	// t.Log(tr)

	// tree.Delete(102)
	// tree.Print()
	// fmt.Println()
	// fmt.Println("------")

	// tree.Delete(105)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(100)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(32)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(31)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(5)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(2)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(26)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(24)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(104)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(90) // TEST CASE FAILED: reason being that during preemption, 90 should be filled with 100 and 102 should come up
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(107)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(90)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// tree.Delete(31)
	// tree.Print()
	// fmt.Println()
	// fmt.Println()

	// LEFTOFF: at t.Delete(31), the merging is a bit messed up, my code and I are both confused which childs are to be picked up
	// and which ones are to be used
	// Also must think of when to execute preemptive strategy, if an internal node has the key, performing a strategy there seems to be a bit foolish because the internal
	// node itself has its own mitigation strategies
}

func BenchmarkInsert(b *testing.B) {
	tr := New(32) // Use a realistic degree
	for i := 0; i < b.N; i++ {
		tr.Insert(uint16(i % 65535))
	}
}

func FuzzBTree(f *testing.F) {
	tr := New(3)
	f.Add(uint16(10))
	f.Fuzz(func(t *testing.T, key uint16) {
		tr.Insert(key)

		auditTreeProperties(t, tr)

		if !tr.Search(key) {
			t.Errorf("Could not find %d", key)
		}
	})
}

func FuzzBTreeWithMinDegree(f *testing.F) {
	tr := New(2)
	f.Add(uint16(10))
	f.Fuzz(func(t *testing.T, key uint16) {
		tr.Insert(key)

		auditTreeProperties(t, tr)

		if !tr.Search(key) {
			t.Errorf("Could not find %d", key)
		}
	})
}
