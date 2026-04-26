package main

import (
	"fmt"
	"slices"
)

type Node struct {
	keys     []uint16
	children []*Node
	isLeaf   bool
}

type BTree struct {
	root   *Node
	degree int
}

func New(degree int) *BTree {
	tree := &BTree{nil, degree}
	tree.root = tree.createNode(true)
	return tree
}

func (t *BTree) Insert(k uint16) {
	if len(t.root.keys) == 2*t.degree-1 {
		t.root = t.splitRoot()
		t.insertInSubtree(t.root, k)
	} else {
		t.insertInSubtree(t.root, k)
	}
}

func (t *BTree) createNode(isLeaf bool) *Node {
	// TODO: let's see maybe we can have 0 length for children also, otherwise revert back to old implementation
	// would need to add more guardrails but at least there's no ambiguity
	return &Node{
		keys:     make([]uint16, 0, 2*t.degree-1),
		children: make([]*Node, 0, 2*t.degree),
		isLeaf:   isLeaf,
	}
}

func (t *BTree) splitRoot() *Node {
	// create a new root
	newRoot := t.createNode(false)
	// new root stores the median key
	median := len(t.root.keys) / 2
	newRoot.keys = append(newRoot.keys, t.root.keys[median])
	// setup new child nodes
	left, right := t.createNode(t.root.isLeaf), t.createNode(t.root.isLeaf)
	left.keys = append(left.keys, t.root.keys[:median]...)
	right.keys = append(right.keys, t.root.keys[median+1:]...)
	// append children of old root to new root
	newRoot.children = append(newRoot.children, left, right)
	// if the root is not a leaf, then we must reattach
	// the children of old root to the new root
	if !t.root.isLeaf {
		left.children = append([]*Node(nil), t.root.children[:median+1]...)
		right.children = append([]*Node(nil), t.root.children[median+1:]...)
	}
	return newRoot
}

func (t *BTree) split(node *Node, idx int) {
	// setup
	parent := node
	oldChild := node.children[idx]
	// fetch child's median key
	median := len(oldChild.keys) / 2
	t.insertInNode(parent, oldChild.keys[median])
	// setup a new child node aka sibling for the split
	newChild := t.createNode(oldChild.isLeaf)
	// add the keys to the new child
	t.insertInNode(newChild, oldChild.keys[median+1:]...)
	// remove the keys from old child
	oldChild.keys = oldChild.keys[:median]
	// reattach the new child to its parent
	parent.children = append(parent.children, nil)
	if idx+1 <= len(parent.children)-1 {
		copy(parent.children[idx+2:], parent.children[idx+1:])
		parent.children[idx+1] = newChild
	}
	// if the child was an internal node, redistribute the old child and new child amongst themselves
	if !newChild.isLeaf {
		newChild.children = append([]*Node(nil), oldChild.children[median+1:]...)
		oldChild.children = oldChild.children[:median+1]
	}
}

func (t *BTree) insertInSubtree(node *Node, k uint16) {
	// Preemptively breakdown overfull child nodes: working proactively
	// on child so that we have access to the parent
	idx := calculateAppropriateIdx(node.keys, k)
	if len(node.children) > 0 && len(node.children[idx].keys) == 2*t.degree-1 {
		t.split(node, idx)
	}
	// Case A: internal node
	// Simply move to the next node (does not matter if it's internal or leaf)
	// because the preemptive breakdown will handle it anyway before any insertion
	if !node.isLeaf {
		idx = calculateAppropriateIdx(node.keys, k)
		t.insertInSubtree(node.children[idx], k)
		return
	}
	// Case B: leaf node
	// It's also the appropriate space to insert the key because:
	// 1. Due to the preemptive breakdown, it is guaranteed to have space
	// 2. Due to recursive nature, if we reach here, it means it's the right node
	if node.isLeaf {
		t.insertInNode(node, k)
		return
	}
}

func (t *BTree) insertInNode(node *Node, k ...uint16) {
	// It's fine for now to just append and sort because
	// it would anyways end up at the same place
	node.keys = append(node.keys, k...)
	slices.Sort(node.keys)
}

func (t *BTree) Search(k uint16) bool {
	node := t.root
	for node != nil {
		isKeyExists := slices.Contains(node.keys, k)
		if isKeyExists {
			return true
		}
		if !isKeyExists && node.isLeaf {
			return false
		}
		idx := calculateAppropriateIdx(node.keys, k)
		node = node.children[idx]
	}
	return false
}

func (tree *BTree) Print() {
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
		queue = queue[size:]
		level++
	}
}

func main() {
	tree := New(2)
	mockInsert(tree)
	tree.Print()
}

// func (t *BTree) Delete(k uint16) {
// 	node := t.root
// 	t.delete(nil, node, k)
// 	// TODO: assume root is all good yeah
// }

// func (t *BTree) delete(parent, node *Node, k uint16) {
// 	// we preemptively fix this node = we're about to get into a node which has t-1 keys,
// 	// before we remove anything from it, we must first bring it to at least t keys to avoid
// 	// complexive fixing after deletion
// 	if parent != nil && len(node.keys) <= MIN_KEYS_PER_NODE {
// 		idx := t.calculateAppropriateIdx(parent.keys, k)
// 		// if left sibling has enough keys, perform rotation
// 		if idx-1 >= 0 && len(parent.children[idx-1].keys) > MIN_KEYS_PER_NODE {
// 		}

// 		// if right sibling has enough keys, perform rotation
// 		if idx+1 <= len(parent.children)-1 && len(parent.children[idx+1].keys) > MIN_KEYS_PER_NODE {
// 		}

// 		// peform merging
// 	}

// 	// if does not contain, recurse to next child
// 	if !slices.Contains(node.keys, k) {
// 		idx := t.calculateAppropriateIdx(node.keys, k)
// 		t.delete(node, node.children[idx], k)
// 		return
// 	}
// }

// func (t *BTree) delete(parent, node *Node, k uint16) {
//
// 	if parent != nil && len(node.keys) <= MIN_KEYS_PER_NODE {
// 		// check if left sibling has t keys
// 		// check if right sibling has t keys
// 		// merge
// 	}

// 	// if the node does not have the key, recurse to the next possible child
// 	if !slices.Contains(node.keys, k) {
// 		// idx := t.calculateAppropriateIdx(node.keys, k)
// 		// t.delete(node.children[idx], k)
// 		return
// 	}

// 	// else, the node has the key, let's check for the different scenarios:
// 	idx := slices.Index(node.keys, k)
// 	switch {
// 	// Case A: if it's a leaf and has t keys --> delete
// 	case node.isLeaf && len(node.keys) > MIN_KEYS_PER_NODE:
// 		t.deleteKey(node, k)

// 	// Case B: if it's an internal node and left child has t keys --> predecessor mechanism
// 	case !node.isLeaf && len(node.children[idx].keys) > MIN_KEYS_PER_NODE:
// 		childKeys := node.children[idx].keys
// 		predecessor := childKeys[len(childKeys)-1]
// 		node.keys[idx] = predecessor
// 		t.deleteKey(node.children[idx], predecessor)

// 	// Case C: if it's an internal node + left child does not have t keys + right child does --> successor mechanism
// 	case !node.isLeaf && len(node.children[idx+1].keys) > MIN_KEYS_PER_NODE:
// 		childKeys := node.children[idx+1].keys
// 		successor := childKeys[0]
// 		node.keys[idx] = successor
// 		t.deleteKey(node.children[idx+1], successor)

// 	// Case D: if it's an internal node + neither child has t keys --> perform merging of left, right and the key followed by removing the key
// 	default:
// 		t.deleteKey(node, k)
// 		node.children[idx].keys = append(node.children[idx].keys, node.children[idx+1].keys...)
// 		node.children[idx+1] = nil
// 	}
// }

// func (t *BTree) deleteKey(node *Node, k uint16) {
// 	idx := slices.Index(node.keys, k)
// 	copy(node.keys[idx:], node.keys[idx+1:])
// 	node.keys = node.keys[:len(node.keys)-1]
// }
