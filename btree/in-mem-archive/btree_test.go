package btree

import (
	"math/rand"
	"strings"
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

func TestAdversarial_MergeCascade(t *testing.T) {
	tr := New(2)

	// Fill tree
	for i := 1; i <= 20; i++ {
		tr.Insert(uint16(i))
	}

	// Delete in ascending order → forces merges upward
	for i := 1; i <= 20; i++ {
		tr.Delete(uint16(i))
		auditTreeProperties(t, tr)
	}
}

func TestAdversarial_BorrowEdge(t *testing.T) {
	tr := New(2)

	keys := []uint16{10, 20, 5, 6, 12, 30, 7, 17}
	for _, k := range keys {
		tr.Insert(k)
	}

	// These hit tricky borrow/merge decisions
	for _, k := range []uint16{6, 7, 5} {
		tr.Delete(k)
		auditTreeProperties(t, tr)
	}
}

func TestAdversarial_InternalDelete(t *testing.T) {
	tr := New(2)

	for i := 1; i <= 15; i++ {
		tr.Insert(uint16(i))
	}

	// delete non-leaf keys
	for _, k := range []uint16{8, 4, 12} {
		tr.Delete(k)
		auditTreeProperties(t, tr)
	}
}

func TestAdversarial_RootShrink(t *testing.T) {
	tr := New(2)

	for i := 1; i <= 10; i++ {
		tr.Insert(uint16(i))
	}

	for i := 1; i <= 9; i++ {
		tr.Delete(uint16(i))
		auditTreeProperties(t, tr)
	}
}

func TestAdversarial_Oscillation(t *testing.T) {
	tr := New(2)

	for i := 0; i < 50; i++ {
		tr.Insert(uint16(i))
	}

	for i := 0; i < 200; i++ {
		k := uint16(i % 50)

		tr.Delete(k)
		tr.Insert(k)

		auditTreeProperties(t, tr)
	}
}

func TestChurn(t *testing.T) {
	tr := New(3)
	for i := 0; i < 100; i++ {
		tr.Insert(uint16(i))
	}

	r := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		tr.Delete(uint16(r.Intn(1000)))
		tr.Insert(uint16(r.Intn(1000)))
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

	tr.Delete(5)
	t.Log(tr)

	tr.Delete(6)
	t.Log(tr)

	tr.Delete(12)
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

func TestCriticalDelete1(t *testing.T) {
	tr := New(2)

	keys := []uint16{20, 10, 12, 24, 6, 31, 32, 18, 26, 25, 27, 2, 48, 1, 21, 22, 4,
		5, 90, 92, 100, 102, 104, 107, 108, 110, 105, 33, 34, 35}
	for _, key := range keys {
		tr.Insert(key)
	}

	tr.Delete(105)
	tr.Delete(107)
	tr.Delete(90)
	got := tr.String()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Level 0:\n")
	sb.WriteString("[ 12  24  48 ] {0:2}  {1:1}  {2:2}  {3:2} \n")
	sb.WriteString("Level 1:\n")
	sb.WriteString("[ 2  6 ] {0:1}  {1:2}  {2:1} [ 20 ] {0:1}  {1:2} [ 31  33 ] {0:3}  {1:1}  {2:2} [ 102  108 ] {0:2}  {1:1}  {2:1} \n")
	sb.WriteString("Level 2:\n")
	sb.WriteString("[ 1 ][ 4  5 ][ 10 ][ 18 ][ 21  22 ][ 25  26  27 ][ 32 ][ 34  35 ][ 92  100 ][ 104 ][ 110 ]\n")
	want := sb.String()

	if got != want {
		t.Fatalf("mismatch:\n got:%s\n want: %s\n", got, want)
	}

	auditTreeProperties(t, tr)
}

func TestCriticalDelete2(t *testing.T) {
	tr := New(2)

	keys := []uint16{20, 10, 12, 24, 6, 31, 32, 18, 26, 25, 27, 2, 48, 1, 21, 22, 4,
		5, 90, 92, 100, 102, 104, 107, 108, 110, 105, 33, 34, 35}
	for _, key := range keys {
		tr.Insert(key)
	}

	tr.Delete(105)
	tr.Delete(107)
	got := tr.String()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Level 0:\n")
	sb.WriteString("[ 12  24  48 ] {0:2}  {1:1}  {2:2}  {3:3} \n")
	sb.WriteString("Level 1:\n")
	sb.WriteString("[ 2  6 ] {0:1}  {1:2}  {2:1} [ 20 ] {0:1}  {1:2} [ 31  33 ] {0:3}  {1:1}  {2:2} [ 92  102  108 ] {0:1}  {1:1}  {2:1}  {3:1} \n")
	sb.WriteString("Level 2:\n")
	sb.WriteString("[ 1 ][ 4  5 ][ 10 ][ 18 ][ 21  22 ][ 25  26  27 ][ 32 ][ 34  35 ][ 90 ][ 100 ][ 104 ][ 110 ]\n")
	want := sb.String()

	if got != want {
		t.Fatalf("mismatch:\n got:%s\n want: %s\n", got, want)
	}

	auditTreeProperties(t, tr)
}

func TestCriticalDelete3(t *testing.T) {
	tr := New(2)

	keys := []uint16{20, 10, 12, 24, 6, 31, 32, 18, 26, 25, 27, 2, 48, 1, 21, 22, 4,
		5, 90, 92, 100, 102, 104, 107, 108, 110, 105, 33, 34, 35}
	for _, key := range keys {
		tr.Insert(key)
	}

	tr.Delete(5)
	tr.Delete(6)
	got := tr.String()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Level 0:\n")
	sb.WriteString("[ 48 ] {0:2}  {1:1} \n")
	sb.WriteString("Level 1:\n")
	sb.WriteString("[ 12  24 ] {0:1}  {1:1}  {2:2} [ 102 ] {0:1}  {1:1} \n")
	sb.WriteString("Level 2:\n")
	sb.WriteString("[ 2 ] {0:1}  {1:2} [ 20 ] {0:1}  {1:2} [ 31  33 ] {0:3}  {1:1}  {2:2} [ 92 ] {0:1}  {1:1} [ 107 ] {0:2}  {1:2} \n")
	sb.WriteString("Level 3:\n")
	sb.WriteString("[ 1 ][ 4  10 ][ 18 ][ 21  22 ][ 25  26  27 ][ 32 ][ 34  35 ][ 90 ][ 100 ][ 104  105 ][ 108  110 ]\n")
	want := sb.String()

	if got != want {
		t.Fatalf("mismatch:\n got:%s\n want: %s\n", got, want)
	}

	auditTreeProperties(t, tr)
}

func TestCriticalDelete4(t *testing.T) {
	tr := New(2)

	keys := []uint16{20, 10, 12, 24, 6, 31, 32, 18, 26, 25, 27, 2, 48, 1, 21, 22, 4,
		5, 90, 92, 100, 102, 104, 107, 108, 110, 105, 33, 34, 35}
	for _, key := range keys {
		tr.Insert(key)
	}

	tr.Delete(5)
	tr.Delete(6)
	tr.Delete(12)
	got := tr.String()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Level 0:\n")
	sb.WriteString("[ 48 ] {0:1}  {1:1} \n")
	sb.WriteString("Level 1:\n")
	sb.WriteString("[ 24 ] {0:3}  {1:2} [ 102 ] {0:1}  {1:1} \n")
	sb.WriteString("Level 2:\n")
	sb.WriteString("[ 2  10  20 ] {0:1}  {1:1}  {2:1}  {3:2} [ 31  33 ] {0:3}  {1:1}  {2:2} [ 92 ] {0:1}  {1:1} [ 107 ] {0:2}  {1:2} \n")
	sb.WriteString("Level 3:\n")
	sb.WriteString("[ 1 ][ 4 ][ 18 ][ 21  22 ][ 25  26  27 ][ 32 ][ 34  35 ][ 90 ][ 100 ][ 104  105 ][ 108  110 ]\n")
	want := sb.String()

	if got != want {
		t.Fatalf("mismatch:\n got:%s\n want: %s\n", got, want)
	}

	auditTreeProperties(t, tr)
}

func BenchmarkInsert(b *testing.B) {
	tr := New(32) // Use a realistic degree
	for i := 0; i < b.N; i++ {
		tr.Insert(uint16(i % 65535))
	}
}

func FuzzBTree1(f *testing.F) {
	f.Add(uint16(10), uint16(1))
	f.Add(uint16(20), uint16(2))
	f.Add(uint16(30), uint16(3))

	f.Fuzz(func(t *testing.T, key uint16, op uint16) {
		tr := New(3)
		ref := make(map[uint16]struct{})

		for i := 0; i < 100; i++ {
			k := key + uint16(i)
			action := (op + uint16(i)) % 3

			switch action {
			case 0:
				tr.Insert(k)
				ref[k] = struct{}{}

			case 1:
				tr.Delete(k)
				delete(ref, k)

			case 2:
				_, exists := ref[k]
				found := tr.Search(k)
				if found != exists {
					t.Fatalf("search mismatch for %d: got=%v want=%v\nTree:%s",
						k, found, exists, tr)
				}
			}

			auditTreeProperties(t, tr)
		}
	})
}

func FuzzMinDegreeBTree2(f *testing.F) {
	f.Add(uint16(10), uint16(1))

	f.Fuzz(func(t *testing.T, key uint16, op uint16) {
		tr := New(2)
		ref := make(map[uint16]struct{})

		for i := 0; i < 100; i++ {
			k := key + uint16(i)
			action := (op + uint16(i)) % 3

			switch action {
			case 0:
				tr.Insert(k)
				ref[k] = struct{}{}

			case 1:
				tr.Delete(k)
				delete(ref, k)

			case 2:
				_, exists := ref[k]
				found := tr.Search(k)
				if found != exists {
					t.Fatalf("search mismatch for %d: got=%v want=%v\nTree:%s",
						k, found, exists, tr)
				}
			}

			auditTreeProperties(t, tr)
		}
	})
}

// auditTreeProperties is a validator that checks if the invariants of
// BTree are still maintained
func auditTreeProperties(t *testing.T, tr *BTree) {
	if tr.root == nil {
		return
	}

	var (
		height = -1
		tdeg   = tr.degree
	)

	var checkNode func(n *Node, depth int, min, max *uint16)

	checkNode = func(n *Node, depth int, min, max *uint16) {
		if n == nil {
			t.Fatalf("nil node encountered")
		}

		// invariant: key count constraints
		if n != tr.root {
			if len(n.keys) < tdeg-1 {
				t.Errorf("node has too few keys: %v", n.keys)
			}
		}
		if len(n.keys) > 2*tdeg-1 {
			t.Errorf("node has too many keys: %v", n.keys)
		}

		// invariant: sorted + range check
		for i := 0; i < len(n.keys); i++ {
			if i > 0 && n.keys[i-1] > n.keys[i] {
				t.Errorf("keys not sorted: %v", n.keys)
			}

			if min != nil && n.keys[i] <= *min {
				t.Errorf("key %d violates min bound %d", n.keys[i], *min)
			}
			if max != nil && n.keys[i] >= *max {
				t.Errorf("key %d violates max bound %d", n.keys[i], *max)
			}
		}

		// invariant: leaf depth consistency
		if n.isLeaf {
			if height == -1 {
				height = depth
			} else if height != depth {
				t.Errorf("leaves at different depths: got %d want %d", depth, height)
			}
			return
		}

		// invariant: children count
		if len(n.children) != len(n.keys)+1 {
			t.Errorf("node has %d keys but %d children", len(n.keys), len(n.children))
		}

		// invariant: recurse with bounds
		for i, child := range n.children {
			var newMin, newMax *uint16

			if i > 0 {
				newMin = &n.keys[i-1]
			} else {
				newMin = min
			}

			if i < len(n.keys) {
				newMax = &n.keys[i]
			} else {
				newMax = max
			}

			checkNode(child, depth+1, newMin, newMax)
		}
	}

	checkNode(tr.root, 0, nil, nil)

	// invariant: root special rule
	if !tr.root.isLeaf && len(tr.root.children) < 2 {
		t.Errorf("non-leaf root must have at least 2 children")
	}
}
