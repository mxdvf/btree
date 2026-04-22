package main

import (
	"fmt"
	"slices"
)

const BRANCHING_FACTOR_T = 2

const MAX_KEYS_PER_NODE = (2 * BRANCHING_FACTOR_T) - 1
const MAX_CHILDREN_PER_NODE = 2 * BRANCHING_FACTOR_T

type Node struct {
	keys     []uint16
	children []*Node
	isLeaf   bool
}

func NewNode(isLeaf bool, keys []uint16) *Node {
	node := &Node{
		keys:     make([]uint16, 0, MAX_KEYS_PER_NODE),
		children: make([]*Node, MAX_CHILDREN_PER_NODE),
		isLeaf:   isLeaf,
	}
	if keys != nil {
		node.keys = append(node.keys, keys...)
	}
	return node
}

type BTree struct {
	root *Node
}

// Case 1A: leaf has t keys --> simply delete it (20)
// Case 1B: leaf has t-1 keys, but sibling has t keys --> swap mechanism (32)

// Case 2A: internal node, left child has t keys --> pred/succ mechanism (11)
// Case 2B: internal node, left child has t-1 BUT right child has t keys --> pred/succ mechanism (11)
// Case 2C: internal node, neither child has t keys --> merging

func (t *BTree) Search(k uint16) bool {
	node := t.root
	for node != nil {
		isKeyExists := t.isKeyExists(node.keys, k)
		if isKeyExists {
			return true
		}
		if !isKeyExists && node.isLeaf {
			return false
		}

		idx := t.calculateAppropriateIdx(node.keys, k)
		node = node.children[idx]
	}

	return false
}

func (t *BTree) isKeyExists(nodeKeys []uint16, k uint16) bool {
	// simple linear search
	var idx int
	for idx = 0; idx < len(nodeKeys); idx++ {
		if k == nodeKeys[idx] {
			return true
		}
	}
	return false
}

func (t *BTree) Insert(k uint16) {
	if len(t.root.keys) == MAX_KEYS_PER_NODE {
		// break the root down before going any further
		t.root = t.splitRoot()
		// perform the insert operation as intended
		t.insertInSubtree(t.root, k)
	} else {
		// perform the insert operation as intended
		t.insertInSubtree(t.root, k)
	}
}

func (t *BTree) splitRoot() *Node {
	// order keys to prepare them for a split
	tempKeys := make([]uint16, MAX_KEYS_PER_NODE)
	copy(tempKeys, t.root.keys)

	// setup the new root and evaluate median
	newRoot := NewNode(false, nil)

	// parent stores the derived median key
	median := len(tempKeys) / 2
	newRoot.keys = append(newRoot.keys, tempKeys[median])

	// setup new child nodes
	left, right := NewNode(t.root.isLeaf, tempKeys[:median]), NewNode(t.root.isLeaf, tempKeys[median+1:])
	newRoot.children[0] = left
	newRoot.children[1] = right

	// if the root is not a leaf, then we must reattach
	// the children of old root as children of new root
	if !t.root.isLeaf {
		copy(left.children, t.root.children[:median+1])
		copy(right.children, t.root.children[median+1:])
	}

	return newRoot
}

func (t *BTree) insertInSubtree(node *Node, k uint16) {
	// Case A: node is a leaf and has space
	if node.isLeaf && len(node.keys) < MAX_KEYS_PER_NODE {
		t.insertInNode(node, k)
		return
	}

	// Case B: node is not a leaf
	if !node.isLeaf {
		// and then proceed
		idx := t.calculateAppropriateIdx(node.keys, k)
		switch {
		// Case B1: if child is not a leaf, keep traversing
		case !node.children[idx].isLeaf:
			// pre-emptively break down an internal node if it's full
			if len(node.children[idx].keys) == MAX_KEYS_PER_NODE {
				t.splitChild(node, node.children[idx], nil)
			}
			// then proceed
			idx = t.calculateAppropriateIdx(node.keys, k)
			t.insertInSubtree(node.children[idx], k)

		// Case B2: if appropriate child (a leaf) has space, insert there
		case node.children[idx].isLeaf && len(node.children[idx].keys) < MAX_KEYS_PER_NODE:
			t.insertInNode(node.children[idx], k)

		// Case B3: if appropriate child (a leaf) does not have space, perform split
		case node.children[idx].isLeaf && len(node.children[idx].keys) == MAX_KEYS_PER_NODE:
			t.splitChild(node, node.children[idx], &k)
		}
	}
}

func (t *BTree) insertInNode(node *Node, k uint16) {
	// Does not matter if any shuffling happens, because by
	// the time we require the indices everything will anyways
	// be settled. Basically this works, no worries on this.
	node.keys = append(node.keys, k)
	slices.Sort(node.keys)
}

func (t *BTree) splitChild(parent *Node, child *Node, k *uint16) {
	// order keys to prepare them for a split
	tempKeys := make([]uint16, MAX_KEYS_PER_NODE)
	copy(tempKeys, child.keys)
	if k != nil {
		idx := t.calculateAppropriateIdx(tempKeys, *k)
		tempKeys = slices.Insert(tempKeys, idx, *k)
	}

	median := len(tempKeys) / 2

	// parent stores the derived median key
	idx := t.calculateAppropriateIdx(parent.keys, tempKeys[median])
	parent.keys = append(parent.keys, 0)
	copy(parent.keys[idx+1:], parent.keys[idx:])
	parent.keys[idx] = tempKeys[median]

	// setup new child nodes
	left, right := NewNode(child.isLeaf, tempKeys[:median]), NewNode(child.isLeaf, tempKeys[median+1:])
	parent.children[idx] = left
	// TODO: the logic here is that instead of creating two nodes
	// create just one, and manipulate the other existing one
	copy(parent.children[idx+1:], parent.children[idx:])
	parent.children[idx+1] = right

	// redistribute the children of child nodes
	if k == nil {
		copy(left.children, child.children[:median+1])
		copy(right.children, child.children[median+1:])
	}
}

// calculateAppropriateIdx returns an int which provides the appropriate
// index within the "sorted" slice of keys. Used for finding appropriate
// child or appropriate position for a key
func (t *BTree) calculateAppropriateIdx(nodeKeys []uint16, k uint16) int {
	// the idea is to return an "appropriate" index for finding
	// the correct child or position for a key
	var idx int
	for idx = 0; idx < len(nodeKeys); idx++ {
		if k < nodeKeys[idx] {
			break
		}
	}
	return idx
}

// print is a recursive function that uses BFS (damn! never thought I would)
// use it in my systems journey)
func (tree *BTree) Print() {
	if tree.root == nil {
		return
	}

	queue := []*Node{tree.root}
	level := 0
	for len(queue) > 0 {
		size := len(queue)

		fmt.Printf("Level %d:\n", level)
		for i := range len(queue) {
			node := queue[i]

			fmt.Print("[")
			for _, k := range node.keys {
				fmt.Printf(" %v ", k)
			}
			fmt.Print("]")

			for _, c := range node.children {
				if c != nil {
					queue = append(queue, c)
				}
			}
		}
		fmt.Println()

		queue = queue[size:] // move to next level
		level++
	}
}

func main() {
	tree := &BTree{
		root: NewNode(true, nil),
	}
	mockInsert(tree)
	tree.Print()
}
