package btree

// func TestNodeNKeys(t *testing.T) {
// 	buf := make([]byte, 4096)
// 	n := NewNode(buf)

// 	k := n.getNKeys()
// 	if k != 0 {
// 		t.Fatal("nkeys should have been 0")
// 	}

// 	n.incrementNKeys()
// 	k = n.getNKeys()
// 	if k != 1 {
// 		t.Fatal("nkeys should have been 1")
// 	}
// }

// func TestNodeLeafNodeInsert1(t *testing.T) {
// 	buf := make([]byte, 4096)
// 	n := NewNode(buf)
// 	err := n.insertInLeafNode([]byte("kacky-24"), []byte("mehul"))
// 	if err != nil {
// 		t.Fatalf("got an error on insertion: %v", err)
// 	}
// }

// func TestNodeLeafNodeInsert2(t *testing.T) {
// 	buf := make([]byte, 4096)
// 	n := NewNode(buf)
// 	for i := range 175 {
// 		k := fmt.Sprintf("kacky-%d", i)
// 		err := n.insertInLeafNode([]byte(k), []byte("mehul"))
// 		if err != nil {
// 			t.Fatalf("got an error on insertion: %v", err)
// 			break
// 		}
// 	}

// 	t.Logf("node is filled to %v bytes\n", n.getSize())

// 	err := n.insertInLeafNode([]byte("a"), []byte("b"))
// 	if err == nil {
// 		t.Fatalf("should've thrown an overflow error: %v", err)
// 	}
// }
