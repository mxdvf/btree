package btree

import (
	"fmt"
	"slices"
	"strings"
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
	// TODO: get rid of this search and somehow incorporate logic within insert itself, write becomes freakingly slow
	if t.Search(k) {
		return
	}

	if len(t.root.keys) == 2*t.degree-1 {
		t.root = t.splitRoot()
		// TODO: instead of creating a new root node, you should keep the old one intact, add two children and just swap them out
		// in this way you don't need to even check for the root separately, it would just work using t.insertInSubtree
	}
	t.insertInSubtree(t.root, k)
}

func (t *BTree) createNode(isLeaf bool) *Node {
	return &Node{
		keys:     make([]uint16, 0, 2*t.degree-1),
		children: make([]*Node, 0, 2*t.degree),
		isLeaf:   isLeaf,
	}
}

func (t *BTree) splitRoot() *Node {
	// Create a new root
	newRoot := t.createNode(false)
	// New root stores the median key
	median := len(t.root.keys) / 2
	newRoot.keys = append(newRoot.keys, t.root.keys[median])
	// Setup new child nodes
	left, right := t.createNode(t.root.isLeaf), t.createNode(t.root.isLeaf)
	left.keys = append(left.keys, t.root.keys[:median]...)
	right.keys = append(right.keys, t.root.keys[median+1:]...)
	// Append children of old root to new root
	newRoot.children = append(newRoot.children, left, right)
	// If the root is not a leaf, then we must reattach
	// the children of old root to the new root
	if !t.root.isLeaf {
		left.children = append([]*Node(nil), t.root.children[:median+1]...)
		right.children = append([]*Node(nil), t.root.children[median+1:]...)
	}
	return newRoot
}

func (t *BTree) split(node *Node, idx int) {
	// Setup
	parent := node
	oldChild := node.children[idx]
	// Fetch child's median key
	median := len(oldChild.keys) / 2
	t.insertInNode(parent, oldChild.keys[median])
	// Setup a new child node aka sibling for the split
	newChild := t.createNode(oldChild.isLeaf)
	// Add the keys to the new child
	t.insertInNode(newChild, oldChild.keys[median+1:]...)
	// Remove the keys from old child
	oldChild.keys = oldChild.keys[:median]
	// Reattach the new child to its parent
	parent.children = append(parent.children, nil)
	if idx+1 <= len(parent.children)-1 {
		copy(parent.children[idx+2:], parent.children[idx+1:])
		parent.children[idx+1] = newChild
	}
	// If the child was an internal node, redistribute the old child and new child amongst themselves
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

func (t *BTree) Delete(k uint16) {
	t.delete(t.root, k)
	// If root became empty and has a child, the child is the new root
	if len(t.root.keys) == 0 && !t.root.isLeaf {
		t.root = t.root.children[0]
	}
}

func (t *BTree) delete(node *Node, k uint16) {
	// Preemptive Fix: fix the internal node it might recurse into
	// 1. Either perform a left/right rotation using a sibling
	// 2. Or combine the left/right sibling, parent and the node
	if !node.isLeaf && !slices.Contains(node.keys, k) {
		idx := calculateAppropriateIdx(node.keys, k)
		if len(node.children[idx].keys) <= t.degree-1 {
			t.preemptiveFix(node, idx)
		}
		// Simply recurse to the next appropriate subtree
		idx = calculateAppropriateIdx(node.keys, k)
		t.delete(node.children[idx], k)
		return
	}
	// Case A: internal node and it has the key
	// 1. If left/right immediate child have enough, borrow from "inorder predecessor/successor" (read about inorder, took 3 days to realise this)
	// 2. If both immediate child don't have enougb keys, consolidate with the parent
	if !node.isLeaf && slices.Contains(node.keys, k) {
		subtree, keyToBeDeleted := t.deleteFromInternalNode(node, k)
		t.delete(subtree, keyToBeDeleted)
		return
	}
	// Case B: leaf node and it has the key
	if node.isLeaf && slices.Contains(node.keys, k) {
		t.deleteFromLeafNode(node, k)
		return
	}
}

func (t *BTree) preemptiveFix(node *Node, idx int) {
	parent := node
	child := parent.children[idx]
	// Preemptive Fix 1A: child's left sibling has enough space, so we right rotate
	if idx-1 >= 0 && len(parent.children[idx-1].keys) > t.degree-1 {
		t.preemptiveRightRotate(parent, child, idx)
		return
	}
	// Preemptive Fix 1B: child's right sibling has enough space, so we left rotate
	if idx+1 <= len(parent.children)-1 && len(parent.children[idx+1].keys) > t.degree-1 {
		t.preemptiveLeftRotate(parent, child, idx)
		return
	}
	// Preemptive Fix 2A: merge the child with its left sibling and parent
	// No need to check for the length of its left sibling because we know it's less than required
	if idx-1 >= 0 {
		t.preemptiveLeftCombine(parent, child, idx)
		return
	}
	// Preemptive Fix 2B: merge the child with its right sibling and parent
	if idx+1 <= len(parent.children)-1 {
		t.preemptiveRightCombine(parent, child, idx)
		return
	}
}

func (t *BTree) preemptiveRightRotate(parent, child *Node, idx int) {
	childLeftSibling := parent.children[idx-1]
	// Get the predecessor and remove it
	keys := childLeftSibling.keys
	predecessor := keys[len(keys)-1]
	childLeftSibling.keys = keys[:len(keys)-1]
	// Get the parent key and replace it with the predecessor
	parentKey := parent.keys[idx-1]
	parent.keys[idx-1] = predecessor
	// Add the parent key in the child node
	child.keys = append([]uint16{parentKey}, child.keys...)
	// If the sibling is an internal node, transfer children
	if !childLeftSibling.isLeaf {
		// Add the children of child's left sibling to the child
		danglingGrandChild := childLeftSibling.children[len(childLeftSibling.children)-1]
		child.children = append([]*Node{danglingGrandChild}, child.children...)
		childLeftSibling.children = childLeftSibling.children[:len(childLeftSibling.children)-1]
	}
}

func (t *BTree) preemptiveLeftRotate(parent, child *Node, idx int) {
	childRightSibling := parent.children[idx+1]
	// Get the successor and remove it
	keys := childRightSibling.keys
	successor := keys[0]
	childRightSibling.keys = keys[1:]
	// Get the parent key and replace it with the successor
	parentKey := parent.keys[idx]
	parent.keys[idx] = successor
	// Add the parent key in the child node
	child.keys = append(child.keys, parentKey)
	// If the sibling is an internal node, transfer children
	if !childRightSibling.isLeaf {
		// Add the children of child's right sibling to the child
		danglingGrandChild := childRightSibling.children[0]
		child.children = append(child.children, danglingGrandChild)
		childRightSibling.children = childRightSibling.children[1:]
	}
}

func (t *BTree) preemptiveLeftCombine(parent, child *Node, idx int) {
	// Establish left sibling node
	childLeftSibling := parent.children[idx-1]
	// Transfer keys of parent and child to its left sibling
	parentKey := parent.keys[idx-1]
	childLeftSibling.keys = append(childLeftSibling.keys, parentKey)
	childLeftSibling.keys = append(childLeftSibling.keys, child.keys...)
	// Remove key from parent using left shift
	copy(parent.keys[idx-1:], parent.keys[idx:])
	parent.keys = parent.keys[:len(parent.keys)-1]
	// Remove child
	copy(parent.children[idx:], parent.children[idx+1:])
	parent.children = parent.children[:len(parent.children)-1]
	// If the child is an internal node, transfer children
	if !child.isLeaf {
		childLeftSibling.children = append(childLeftSibling.children, child.children...)
	}
}

func (t *BTree) preemptiveRightCombine(parent, child *Node, idx int) {
	// Establish right sibling node
	childRightSibling := parent.children[idx+1]
	// Transfer keys of parent and child to its left sibling
	parentKey := parent.keys[idx]
	childRightSibling.keys = append([]uint16{parentKey}, childRightSibling.keys...)
	childRightSibling.keys = append(child.keys, childRightSibling.keys...)
	// Remove key from parent using right shift
	copy(parent.keys[1:], parent.keys[:idx])
	parent.keys = parent.keys[1:]
	// Remove child using right shift
	copy(parent.children[1:], parent.children[:idx])
	parent.children = parent.children[1:]
	// If the child is an internal node, transfer children
	if !child.isLeaf {
		childRightSibling.children = append(child.children, childRightSibling.children...)
	}
}

func (t *BTree) deleteFromInternalNode(node *Node, k uint16) (*Node, uint16) {
	idx := slices.Index(node.keys, k)
	// Case A1: If its immediate left child has enough space, perform a simple borrow
	if len(node.children[idx].keys) > t.degree-1 {
		predecessor := t.borrowFromInorderPredecessor(node, idx)
		return node.children[idx], predecessor
	}
	// Case A2: If its immediate right child has enough space, perform a simple borrow
	if len(node.children[idx+1].keys) > t.degree-1 {
		successor := t.borrowFromInorderSuccessor(node, idx)
		return node.children[idx+1], successor
	}
	// Case A3: If its children don't have enough space, a merge is inevitable
	// newChild is the new child in which the key was inserted into, we capture this
	// so we can get into it recursively in order to the delete the key `k`
	newChild := t.merge(node, idx)
	return newChild, k
}

func (t *BTree) borrowFromInorderPredecessor(node *Node, idx int) uint16 {
	// Continously traverse the right subtree of the left child to find the maximum key
	child := node.children[idx]
	for !child.isLeaf {
		child = child.children[len(child.children)-1]
	}
	// Fetch the predecessor
	predecessor := child.keys[len(child.keys)-1]
	// Replace parent's key `node.keys[idx]` with the predecessor
	node.keys[idx] = predecessor
	return predecessor
}

func (t *BTree) borrowFromInorderSuccessor(node *Node, idx int) uint16 {
	// Continously traverse the left subtree of the right child to find the minimum key
	child := node.children[idx+1]
	for !child.isLeaf {
		child = child.children[0]
	}
	// Fetch the successor (aka minimum key)
	successor := child.keys[0]
	// Replace parent's key `node.keys[idx]` with the predecessor
	node.keys[idx] = successor
	return successor
}

func (t *BTree) merge(node *Node, idx int) *Node {
	// Case A3: merging with children
	parent := node
	leftChild := node.children[idx]
	rightChild := node.children[idx+1]
	// Append the keys of the parent and the right child into the left child
	parentKey := parent.keys[idx]
	leftChild.keys = append(leftChild.keys, parentKey)
	leftChild.keys = append(leftChild.keys, rightChild.keys...)
	// Remove the appropriate key from the parent
	copy(parent.keys[idx:], parent.keys[idx+1:])
	parent.keys = parent.keys[:len(parent.keys)-1]
	// Remove the right child
	copy(parent.children[idx+1:], parent.children[idx+2:])
	parent.children = parent.children[:len(parent.children)-1]
	// If the child is not a leaf, shift all children of the right child to the left child
	if !leftChild.isLeaf {
		leftChild.children = append(leftChild.children, rightChild.children...)
	}
	// Return the subtree root (child) in which the fittings happened
	return node.children[idx]
}

func (t *BTree) deleteFromLeafNode(node *Node, k uint16) {
	idx := slices.Index(node.keys, k)
	node.keys[idx] = node.keys[len(node.keys)-1]
	node.keys = node.keys[:len(node.keys)-1]
	slices.Sort(node.keys)
}

func (t *BTree) String() string {
	var sb strings.Builder
	sb.WriteString("\n")
	queue := []*Node{t.root}
	level := 0
	for len(queue) > 0 {
		size := len(queue)
		sb.WriteString(fmt.Sprintf("Level %d:\n", level))
		for i := range len(queue) {
			node := queue[i]
			sb.WriteString("[")
			for _, k := range node.keys {
				sb.WriteString(fmt.Sprintf(" %v ", k))
			}
			sb.WriteString("]")
			for i, c := range node.children {
				sb.WriteString(fmt.Sprintf(" {%d:%v} ", i, len(c.keys)))
				if c != nil {
					queue = append(queue, c)
				}
			}
		}
		sb.WriteString("\n")
		queue = queue[size:]
		level++
	}
	return sb.String()
}
