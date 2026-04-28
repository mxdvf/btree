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
		// TODO: instead of creating a new root node, you should keep the old one intact, add two children and just swap them out
		// in this way you don't need to even check for the root separately, it would just work using t.insertInSubtree
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
}

func (t *BTree) delete(node *Node, k uint16) {
	// edge cases:

	// parent is internal node, child is leaf
	// - key is in child -> it will work
	// - key is in parent - premature exit, combining and borrowing already works

	// parent is internal node, child is internal node
	// - key is in child -› fix it if it's not stable, recurse
	// - key is in parent ->

	fmt.Println("key toh bata de saala", node.keys, k)

	if !node.isLeaf {
		if slices.Contains(node.keys, k) {
			idx := slices.Index(node.keys, k)
			fmt.Println("nateeja kya nikla that matters", len(node.children[idx].keys) <= t.degree-1 && len(node.children[idx+1].keys) <= t.degree-1)
			if len(node.children[idx].keys) <= t.degree-1 && len(node.children[idx+1].keys) <= t.degree-1 {
				leftChild := node.children[idx]
				rightChild := node.children[idx+1]
				// Case B3: combining
				fmt.Println("jhol??", k, idx)
				// add keys from right child to the left child and don't ignore the parent
				leftChild.keys = append(leftChild.keys, node.keys[idx])
				leftChild.keys = append(leftChild.keys, rightChild.keys...)

				// remove the key from parent
				copy(node.keys[idx:], node.keys[idx+1:])
				node.keys = node.keys[:len(node.keys)-1]

				// delete the right child and shift left all children
				node.children[idx+1] = nil
				copy(node.children[idx+1:], node.children[idx+2:])
				node.children = node.children[:len(node.children)-1]

				// transfer all children of (deleted) right child over to the (preserved) left child
				leftChild.children = append(leftChild.children, rightChild.children...)

				fmt.Println("----ohhhhhhh dooodleeeeee")
				t.Print()
				fmt.Println()

				idx := calculateAppropriateIdx(node.keys, k)
				t.delete(node.children[idx], k)
				return
			} else {
				// parent itself has the key
				t.deleteFromInternalNode(node, k) // TODO: ask what if its children are internal? what if they are leaf?
			}
		} else {
			// parent does not have the key, we don't even know if child
			// has it so let's just fix this shit and recurse
			idx := calculateAppropriateIdx(node.keys, k)
			// even child does not have the key
			// child has the key
			if len(node.children[idx].keys) <= t.degree-1 {
				fmt.Println("aaye ho meri zindagi mein tum bahar bann ke", node.children[idx].keys)
				t.rotateOrMerge(node, idx) // TODO: edge case -- `node` is root + a merge happens = root should point to the new root
			}

			// Since this node doesn't have keys, recurse
			idx = calculateAppropriateIdx(node.keys, k)
			t.delete(node.children[idx], k)
			return
		}
	}

	// Case C: leaf node contains the key
	if node.isLeaf && slices.Contains(node.keys, k) {
		t.deleteFromLeafNode(node, k)
		return
	}
}

func (t *BTree) deleteFromLeafNode(node *Node, k uint16) {
	idx := slices.Index(node.keys, k)
	node.keys[idx] = node.keys[len(node.keys)-1]
	node.keys = node.keys[:len(node.keys)-1]
	slices.Sort(node.keys)
}

func (t *BTree) deleteFromInternalNode(node *Node, k uint16) {
	// We first retrieve the appropriate position in the key space
	// to check for left and right child
	idx := slices.Index(node.keys, k)

	fmt.Println("abhi maza aayega na bidhooooooo, PEHLE HI KAAM KHATAM KARDO", node.keys, k)
	fmt.Println()
	t.Print()
	fmt.Println()

	// Case B1: predecessor mechanism
	leftChild := node.children[idx]
	if leftChild != nil && len(leftChild.keys) > t.degree-1 {
		maxKey := leftChild.keys[len(leftChild.keys)-1]
		node.keys[idx] = maxKey
		leftChild.keys = leftChild.keys[:len(leftChild.keys)-1]
		return
	}

	// Case B2: successor mechanism
	rightChild := node.children[idx+1]
	if rightChild != nil && len(rightChild.keys) > t.degree-1 {
		minKey := rightChild.keys[0]
		node.keys[idx] = minKey
		rightChild.keys = rightChild.keys[1:]
		return
	}

	// Case B3: combining
	fmt.Println("jhol??", k, idx)
	// add keys from right child to the left child and ignore parent because
	// it anyways needs to be removed at the end
	leftChild.keys = append(leftChild.keys, rightChild.keys...)

	// remove the key from parent
	copy(node.keys[idx:], node.keys[idx+1:])
	node.keys = node.keys[:len(node.keys)-1]

	// delete the right child and shift left all children
	node.children[idx+1] = nil
	copy(node.children[idx+1:], node.children[idx+2:])
	node.children = node.children[:len(node.children)-1]

	// transfer all children of (deleted) right child over to the (preserved) left child
	leftChild.children = append(leftChild.children, rightChild.children...)
}

func (t *BTree) rotateOrMerge(node *Node, idx int) {
	// Setup
	parent := node
	child := parent.children[idx]

	// Left sibling exists and has enough keys for a rotation
	if idx-1 >= 0 && len(parent.children[idx-1].keys) > t.degree-1 {
		leftSibling := parent.children[idx-1]
		fmt.Println("LEFT SIBLING KISKA AUR KON HAI BHAIIII", leftSibling.keys, idx-1, parent.keys, child.keys, idx)

		// Get the predecessor and remove it
		keys := leftSibling.keys
		predecessor := keys[len(keys)-1]
		leftSibling.keys = keys[:len(keys)-1]

		// Get the parent key and replace it with the predecessor
		parentKey := parent.keys[idx-1]
		parent.keys[idx-1] = predecessor

		// Add the parent key in the child node
		child.keys = append([]uint16{parentKey}, child.keys...)

		if !leftSibling.isLeaf {
			// Add the children of left sibling to the right sibling ONLY IN CASE OF INTERNAL NODES
			unstableChildren := leftSibling.children[len(leftSibling.children)-1]
			child.children = append([]*Node{unstableChildren}, child.children...)
			leftSibling.children = leftSibling.children[:len(leftSibling.children)-1]
		}
		fmt.Println("aftermath?", leftSibling.keys, idx-1, parent.keys, child.keys, idx)
		return
	}

	// Right sibling exists and has enough keys for a rotation
	if idx+1 <= len(parent.children)-1 && len(parent.children[idx+1].keys) > t.degree-1 {
		rightSibling := parent.children[idx+1]

		// Get the successor and remove it
		keys := rightSibling.keys
		successor := keys[0]
		rightSibling.keys = keys[1:]

		// Get the parent key and replace it with the predecessor
		parentKey := parent.keys[idx]
		parent.keys[idx] = successor

		// Add the parent key in the child node
		child.keys = append(child.keys, parentKey)

		if !rightSibling.isLeaf {
			// Add the children of right sibling to the left sibling
			unstableChildren := rightSibling.children[0]
			child.children = append(child.children, unstableChildren)
			rightSibling.children = rightSibling.children[1:]
		}
		return
	}

	fmt.Println("let's see if it reaches here", parent.keys, child.keys, idx)

	// Perform a merge of the parent, the child and one of its sibling. Always
	// attempt to use left sibling first
	if idx-1 >= 0 {
		siblingNode := parent.children[idx-1]

		// Set the parent key in the sibling
		parentKey := parent.keys[idx-1]
		siblingNode.keys = append(siblingNode.keys, parentKey)

		// Set the child keys in the sibling
		childKeys := parent.children[idx].keys
		siblingNode.keys = append(siblingNode.keys, childKeys...)

		// Set the child's children to the sibling
		siblingNode.children = append(siblingNode.children, child.children...)

		// Shift left the other children (aka child's siblings) of the parent
		parent.children[idx] = nil
		copy(parent.children[idx:], parent.children[idx+1:])
		parent.children = parent.children[:len(parent.children)-1]

		// Remove the key from the parent
		copy(parent.keys[idx-1:], parent.keys[idx:])
		parent.keys = parent.keys[:len(parent.keys)-1]
		return
	}

	// When left sibling DNE, attempt right
	if idx+1 <= len(parent.children)-1 {
		siblingNode := parent.children[idx+1]

		// Set the parent in the sibling
		parentKey := parent.keys[idx]
		siblingNode.keys = append([]uint16{parentKey}, siblingNode.keys...)

		// Set the child keys in the sibling
		childKeys := parent.children[idx].keys
		siblingNode.keys = append(childKeys, siblingNode.keys...)

		// Set the child's children to the sibling
		siblingNode.children = append(child.children, siblingNode.children...)

		// Shift left the other children (aka child's siblings) of the parent
		parent.children[idx] = nil
		copy(parent.children[idx:], parent.children[idx+1:]) // TODO: extremely fishy???? how can it be same as above case
		parent.children = parent.children[:len(parent.children)-1]

		// Remove the key from the parent
		copy(parent.keys[idx-1:], parent.keys[idx:]) // TODO: extremely fishy???? how can it be same as above case
		parent.keys = parent.keys[:len(parent.keys)-1]
		return
	}
}

// func (t *BTree) delete(node *Node, k uint16) {
// 	// Case A: preemptively fix when `node` is an internal node
// 	// 1. Don't need separate logic when `node` is leaf because this does it anyway
// 	// 2. Should make sure any node that is manipulated has enough keys
// 	if !node.isLeaf {
// 		idx := calculateAppropriateIdx(node.keys, k) // TODO: what's the difference between k < X and k <= X inside this function
// 		fmt.Println("konsi key chal rhi hai bhai", node.keys, k)

// 		// Perform a fix if the next appropriate child has less keys
// 		if !slices.Contains(node.keys, k) &&

// 		if slices.Contains(node.keys, k) {
// 			t.deleteFromInternalNode(node, k)
// 			return
// 		}

// 		if !slices.Contains(node.keys, k) {
//
// 		}
// 	}

//

//
// }

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
			for i, c := range node.children {
				fmt.Printf(" {%d:%v} ", i, len(c.keys))
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
	fmt.Println()

	// FAILED: for a perfect delete, first a preemptive fix happens that brings 48 with 92 and replaces 49 with 33
	// along with 33's right children into left children of 48
	// now delete happens that performs a simple borrow from
	tree.Delete(92)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(102)
	tree.Print()
	fmt.Println()
	fmt.Println("------")

	tree.Delete(105)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(100)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(32)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(31)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(5)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(2)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(26)
	tree.Print()
	fmt.Println()
	fmt.Println()

	tree.Delete(24)
	tree.Print()
	fmt.Println()
	fmt.Println()

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
