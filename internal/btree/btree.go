package btree

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mxdvf/superfastkv/internal/pagemanager"
)

const (
	NodeTypeLeaf uint16 = iota
	NodeTypeInternal
)

const (
	MaxAllowedKVLen = 1344 // 1344 bytes, simple math to fit 3 in-line keys in a node to maintain b-tree structure

	PageSize    = 4096 // 4096 bytes
	HeaderSize  = 4    // 2 + 2 = 4 bytes
	PointerSize = 4    // 4 bytes
	OffsetSize  = 2    // 2 bytes
	KeyLenSize  = 2    // 2 bytes
	ValLenSize  = 2    // 2 bytes
)

var (
	ErrOverflow = errors.New("key+value too large")
)

type BTree struct {
	root uint32
	pm   *pagemanager.PageManager
}

func NewBTree(filename string, sync bool) (*BTree, error) {
	// open the main file
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open the main file: %w", err)
	}
	// initialize root and master pages
	pm := pagemanager.NewPageManager(fd, PageSize, sync)
	root, err := initializeRootAndMasterPage(pm)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize root and master nodes: %v", err)
	}
	// initialize btree
	return &BTree{
		root: root,
		pm:   pm,
	}, nil
}

func initializeRootAndMasterPage(pm *pagemanager.PageManager) (uint32, error) {
	buf, err := pm.Read(0)
	switch err {
	// if there's EOF error, then the master page does not exist
	case io.EOF:
		if _, err := pm.Allocate(); err != nil {
			return 0, err
		}
		// initialize root page
		root, err := pm.Allocate()
		return root, err
	// if there's no error then master page exists --> root page also exists
	case nil:
		rootPageNum := binary.BigEndian.Uint32(buf[0:])
		return rootPageNum, nil
	// if there's some error, abort
	default:
		return 0, err
	}
}

func (t *BTree) Insert(k, v []byte) error {
	// validate
	if len(k)+len(v) > MaxAllowedKVLen {
		return ErrOverflow
	}
	// load the root from disk
	var (
		rootNode *Node
		err      error
	)
	if rootNode, err = t.loadAsNode(t.root); err != nil {
		return fmt.Errorf("failed to load and transform the root node: %w", err)
	}
	// perform a split if it's already full
	if rootNode.overflow() {
		rootNode, err = t.setupNewRoot(rootNode)
		if err != nil {
			return fmt.Errorf("failed to setup the new root: %w", err)
		}
	}
	// from here, the wrapper takes over (node) and all operations
	// are thus performed on the wrapper
	pageNum, err := t.insert(rootNode, k, v)
	if err != nil {
		return fmt.Errorf("failed to insert the key: %w", err)
	}
	// update master page using the pageNum root page
	t.pointMasterToNewRoot(pageNum)

	return nil
}

func (t *BTree) setupNewRoot(rootNode *Node) (*Node, error) {
	// split root into left and right
	left, right, medianIndex := rootNode.drySplit()
	// persist the right node to disk
	rightPageNum, err := t.copyToNewPage(right)
	if err != nil {
		return nil, fmt.Errorf("could not persist the right node while splitting root: %w", err)
	}
	// persist left child
	leftPageNum, err := t.copyToNewPage(left) // TODO: a new left node should not be created, it should be manipulated to remove unnecessary data
	if err != nil {
		return nil, fmt.Errorf("could not persist the left node while splitting root: %w", err)
	}
	// create new root
	buf := make([]byte, PageSize)
	newRootNode := NewNode(buf)
	// set type to internal
	newRootNode.setType(NodeTypeInternal)
	// insert median key into new root
	medianKey, medianVal := rootNode.getKV(medianIndex)
	newRootNode.insertSelf(medianKey, medianVal)
	// set pointers to left and right children
	newRootNode.setPtr(0, leftPageNum)
	newRootNode.setPtr(1, rightPageNum)
	// allocate and write new root
	rootPageNum, err := t.copyToNewPage(newRootNode)
	if err != nil {
		return nil, fmt.Errorf("could not persist the right node while splitting root: %w", err)
	}
	// point master to the new root page
	if err = t.pointMasterToNewRoot(rootPageNum); err != nil {
		return nil, fmt.Errorf("failed to point master to the new root: %w", err)
	}
	// return the new root
	return newRootNode, nil
}

func (t *BTree) insert(node *Node, k, v []byte) (uint32, error) {
	// preemptive fix before ever touching a child node
	if node.getType() == NodeTypeInternal {
		if err := t.preemptiveFix(node, k, v); err != nil {
			return 0, err
		}
	}
	// handle insertion appropriately
	switch node.getType() {
	case NodeTypeLeaf:
		// simple logic to insert into the node and perform a
		// copy-on-write
		return t.handleInsertionInLeafNode(node, k, v)
	case NodeTypeInternal:
		// orchestrator logic to enter into the correct subtree page
		// recursively insert on that subtree's root, update all page nums
		// upwards
		return t.handleInsertionInInternalNode(node, k, v)
	}
	panic("should not have reached this point")
}

func (t *BTree) preemptiveFix(node *Node, k, v []byte) error {
	// find the appropriate child that you're about to enter into
	idx, _ := node.findInsertPos(k)
	// load that child into a node
	appropriateChildPageNum := node.getPtr(idx)
	childNode, err := t.loadAsNode(appropriateChildPageNum)
	if err != nil {
		return fmt.Errorf("could not read the appropriate subtree's page: %w", err)
	}
	// check if it's full, if yes break it down into 2 nodes
	if childNode.overflow() {
		leftChildNode, rightChildNode, medianIndex := childNode.drySplit()
		// persist the right node to disk
		rightPageNum, err := t.copyToNewPage(rightChildNode)
		if err != nil {
			return fmt.Errorf("could not persist the right node during preemptive fix: %w", err)
		}
		// persist left child
		leftPageNum, err := t.copyToNewPage(leftChildNode) // TODO: a new left node should not be created, it should be manipulated to remove unnecessary data
		if err != nil {
			return fmt.Errorf("could not persist the left node during preemptive fix: %w", err)
		}
		// insert median key into new root
		medianKey, medianVal := childNode.getKV(medianIndex)
		idx, err := node.insertSelf(medianKey, medianVal)
		if err != nil {
			return fmt.Errorf("failed to insert median key and value during preemptive fix: %w", err)
		}
		// set pointers to left and right children
		node.setPtr(idx, leftPageNum)
		node.setPtr(idx+1, rightPageNum)
	}
	return nil
}

func (t *BTree) handleInsertionInLeafNode(node *Node, k, v []byte) (uint32, error) {
	// attempt insertion on the leaf node
	if _, err := node.insertSelf(k, v); err != nil {
		return 0, err
	}
	// allocate and write to the new page
	pageNum, err := t.copyToNewPage(node)
	if err != nil {
		return 0, fmt.Errorf("failed to allocate when writing back updated bytes to disk: %w", err)
	}
	// return the newly allocated page num
	return pageNum, nil
}

func (t *BTree) handleInsertionInInternalNode(node *Node, k, v []byte) (uint32, error) {
	// figure out which node it should be
	idx, _ := node.findInsertPos(k)
	appropriateSubtreePageNum := node.getPtr(idx)
	// insert into the appropriate subtree
	appropriateSubtree, err := t.loadAsNode(appropriateSubtreePageNum)
	if err != nil {
		return 0, fmt.Errorf("could not read the appropriate subtree's page: %w", err)
	}
	childPageNum, err := t.insert(appropriateSubtree, k, v)
	if err != nil {
		return 0, err
	}
	// we receive the page number of that node and so we now update our pointer
	node.setPtr(idx, childPageNum)
	// this node itself is put to a new location
	pageNum, err := t.copyToNewPage(node)
	if err != nil {
		return 0, fmt.Errorf("failed to allocate when writing back updated bytes to disk: %w", err)
	}
	// and finally we return the pagenum of this
	return pageNum, nil
}

func (t *BTree) Search(k []byte) ([]byte, error) {
	root, err := t.pm.Read(t.root)
	if err != nil {
		return nil, fmt.Errorf("failed to read the root: %v", err)
	}
	rootNode := NewNode(root)
	return t.search(rootNode, k)
}

func (t *BTree) search(node *Node, target []byte) ([]byte, error) {
	// this function will give us an index such that target <= some_key_in_node
	idx, _ := node.findInsertPos(target)
	// assuming the key is in the node, we will receive an index that is within bounds
	// because if the key is out of bounds then it means that we must traverse down. the
	// only reason we can receive an out of bound index is because we're using the a helper
	// for insertion which can return out of bound index if the key is larger than all keys
	// present in the node
	nKeys := node.getNKeys()
	if idx < nKeys {
		k, v := node.getKV(idx)
		if res := bytes.Compare(k, target); res == 0 {
			return v, nil
		}
	}
	// if the node type we're operating on is a leaf, then we end the search
	if node.getType() == NodeTypeLeaf {
		return nil, fmt.Errorf("reached the end of the tree and couldn't find the key")
	}
	// ptr[idx] is always the correct child because as for pointers, they are always 1 more
	// than the # of keys, so it works for: (1) idx < nkeys, and (2) idx = nkeys because we will
	// always receive the correct pointer
	pageNum := node.getPtr(idx)
	buf, err := t.pm.Read(pageNum)
	if err != nil {
		return nil, fmt.Errorf("failed to read page: %v", err)
	}
	// recursively search the subtree
	return t.search(NewNode(buf), target)
}

// func (t *BTree) Delete(k []byte) error {
// 	rootNode, err := t.loadAsNode(t.root)
// 	if err != nil {
// 		return fmt.Errorf("failed to load root node: %w", err)
// 	}
// 	pageNum, err := t.delete(rootNode, k)
// 	if err != nil {
// 		return err
// 	}
// 	return t.pointMasterToNewRoot(pageNum)
// }

// func (t *BTree) delete(node *Node, k []byte) (uint32, error) {
// 	// preemptive fix before ever touching a child node
// 	if node.getType() == NodeTypeInternal {
// 		if err := t.preemptiveDeleteFix(node, k); err != nil {
// 			return 0, err
// 		}
// 	}
// 	switch node.getType() {
// 	case NodeTypeLeaf:
// 		return t.handleDeletionInLeafNode(node, k)
// 	case NodeTypeInternal:
// 		return t.handleDeletionInInternalNode(node, k)
// 	}
// 	panic("should not have reached this point")
// }

// func (t *BTree) preemptiveDeleteFix(node *Node, k []byte) error {
// 	idx, _ := node.findInsertPos(k)
// 	childPageNum := node.getPtr(idx)
// 	childNode, err := t.loadAsNode(childPageNum)
// 	if err != nil {
// 		return fmt.Errorf("could not load child node: %w", err)
// 	}
// 	// child is not underflowing, no fix needed
// 	if !childNode.underflow() {
// 		return nil
// 	}
// 	// try left sibling rotation first
// 	if idx > 0 {
// 		leftSiblingPageNum := node.getPtr(idx - 1)
// 		leftSibling, err := t.loadAsNode(leftSiblingPageNum)
// 		if err != nil {
// 			return fmt.Errorf("could not load left sibling: %w", err)
// 		}
// 		if !leftSibling.underflow() {
// 			return t.rotateRight(node, leftSibling, childNode, idx)
// 		}
// 	}
// 	// try right sibling rotation
// 	if idx < node.getNKeys() {
// 		rightSiblingPageNum := node.getPtr(idx + 1)
// 		rightSibling, err := t.loadAsNode(rightSiblingPageNum)
// 		if err != nil {
// 			return fmt.Errorf("could not load right sibling: %w", err)
// 		}
// 		if !rightSibling.underflow() {
// 			return t.rotateLeft(node, childNode, rightSibling, idx)
// 		}
// 	}
// 	// no rotation possible, merge
// 	if idx > 0 {
// 		return t.mergeChildren(node, idx-1)
// 	}
// 	return t.mergeChildren(node, idx)
// }

// // rotateRight borrows from left sibling:
// // left sibling's max key goes up to parent, parent's key comes down to child
// func (t *BTree) rotateRight(parent, leftSibling, child *Node, idx uint16) error {
// 	// borrow last key from left sibling
// 	borrowedKey, borrowedVal := leftSibling.getKV(leftSibling.getNKeys() - 1)
// 	// get the parent separator key (sits at idx-1 between left sibling and child)
// 	parentKey, parentVal := parent.getKV(idx - 1)
// 	// push parent key down into child at position 0
// 	child.insertAtFront(parentKey, parentVal)
// 	// if left sibling is internal, transfer its rightmost child pointer to child
// 	if leftSibling.getType() == NodeTypeInternal {
// 		danglingPtr := leftSibling.getPtr(leftSibling.getNKeys())
// 		child.shiftPtrsRight()
// 		child.setPtr(0, danglingPtr)
// 	}
// 	// replace parent separator with borrowed key
// 	parent.updateKV(idx-1, borrowedKey, borrowedVal)
// 	// remove last key from left sibling
// 	leftSibling.deleteLast()
// 	// persist all three
// 	leftPageNum, err := t.copyToNewPage(leftSibling)
// 	if err != nil {
// 		return err
// 	}
// 	childPageNum, err := t.copyToNewPage(child)
// 	if err != nil {
// 		return err
// 	}
// 	parent.setPtr(idx-1, leftPageNum)
// 	parent.setPtr(idx, childPageNum)
// 	return nil
// }

// // rotateLeft borrows from right sibling:
// // right sibling's min key goes up to parent, parent's key comes down to child
// func (t *BTree) rotateLeft(parent, child, rightSibling *Node, idx uint16) error {
// 	// borrow first key from right sibling
// 	borrowedKey, borrowedVal := rightSibling.getKV(0)
// 	// get the parent separator key (sits at idx between child and right sibling)
// 	parentKey, parentVal := parent.getKV(idx)
// 	// push parent key down into child at the end
// 	child.insertSelf(parentKey, parentVal)
// 	// if right sibling is internal, transfer its leftmost child pointer to child
// 	if rightSibling.getType() == NodeTypeInternal {
// 		danglingPtr := rightSibling.getPtr(0)
// 		child.setPtr(child.getNKeys(), danglingPtr)
// 		rightSibling.shiftPtrsLeft()
// 	}
// 	// replace parent separator with borrowed key
// 	parent.updateKV(idx, borrowedKey, borrowedVal)
// 	// remove first key from right sibling
// 	rightSibling.deleteFirst()
// 	// persist all three
// 	childPageNum, err := t.copyToNewPage(child)
// 	if err != nil {
// 		return err
// 	}
// 	rightPageNum, err := t.copyToNewPage(rightSibling)
// 	if err != nil {
// 		return err
// 	}
// 	parent.setPtr(idx, childPageNum)
// 	parent.setPtr(idx+1, rightPageNum)
// 	return nil
// }

// // mergeChildren merges children[idx] and children[idx+1] pulling down parent key at idx
// func (t *BTree) mergeChildren(parent *Node, idx uint16) error {
// 	leftPageNum := parent.getPtr(idx)
// 	rightPageNum := parent.getPtr(idx + 1)
// 	leftChild, err := t.loadAsNode(leftPageNum)
// 	if err != nil {
// 		return err
// 	}
// 	rightChild, err := t.loadAsNode(rightPageNum)
// 	if err != nil {
// 		return err
// 	}
// 	// pull parent separator key down into left child
// 	parentKey, parentVal := parent.getKV(idx)
// 	leftChild.insertSelf(parentKey, parentVal)
// 	// merge right child's keys into left child
// 	for i := uint16(0); i < rightChild.getNKeys(); i++ {
// 		k, v := rightChild.getKV(i)
// 		leftChild.insertSelf(k, v)
// 	}
// 	// if internal, transfer right child's pointers to left child
// 	if rightChild.getType() == NodeTypeInternal {
// 		leftNKeys := leftChild.getNKeys()
// 		for i := uint16(0); i <= rightChild.getNKeys(); i++ {
// 			leftChild.setPtr(leftNKeys+i, rightChild.getPtr(i))
// 		}
// 	}
// 	// persist merged left child
// 	newLeftPageNum, err := t.copyToNewPage(leftChild)
// 	if err != nil {
// 		return err
// 	}
// 	// remove parent separator key and right child pointer
// 	parent.deleteAt(idx)
// 	parent.setPtr(idx, newLeftPageNum)
// 	return nil
// }

// func (t *BTree) handleDeletionInLeafNode(node *Node, k []byte) (uint32, error) {
// 	if err := node.deleteKey(k); err != nil {
// 		return 0, err
// 	}
// 	pageNum, err := t.copyToNewPage(node)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return pageNum, nil
// }

// func (t *BTree) handleDeletionInInternalNode(node *Node, k []byte) (uint32, error) {
// 	idx, _ := node.findInsertPos(k)
// 	// check if this internal node itself contains the key
// 	if idx < node.getNKeys() {
// 		existingKey, _ := node.getKV(idx)
// 		if bytes.Compare(existingKey, k) == 0 {
// 			return t.handleKeyInInternalNode(node, k, idx)
// 		}
// 	}
// 	// key is not in this node, recurse into appropriate child
// 	childPageNum := node.getPtr(idx)
// 	childNode, err := t.loadAsNode(childPageNum)
// 	if err != nil {
// 		return 0, err
// 	}
// 	newChildPageNum, err := t.delete(childNode, k)
// 	if err != nil {
// 		return 0, err
// 	}
// 	node.setPtr(idx, newChildPageNum)
// 	return t.copyToNewPage(node)
// }

// func (t *BTree) handleKeyInInternalNode(node *Node, k []byte, idx uint16) (uint32, error) {
// 	leftChildPageNum := node.getPtr(idx)
// 	rightChildPageNum := node.getPtr(idx + 1)
// 	leftChild, err := t.loadAsNode(leftChildPageNum)
// 	if err != nil {
// 		return 0, err
// 	}
// 	rightChild, err := t.loadAsNode(rightChildPageNum)
// 	if err != nil {
// 		return 0, err
// 	}
// 	// case A1: borrow inorder predecessor from left child
// 	if !leftChild.underflow() {
// 		predecessor, predVal := t.inorderPredecessor(leftChild)
// 		node.updateKV(idx, predecessor, predVal)
// 		newLeftPageNum, err := t.delete(leftChild, predecessor)
// 		if err != nil {
// 			return 0, err
// 		}
// 		node.setPtr(idx, newLeftPageNum)
// 		return t.copyToNewPage(node)
// 	}
// 	// case A2: borrow inorder successor from right child
// 	if !rightChild.underflow() {
// 		successor, succVal := t.inorderSuccessor(rightChild)
// 		node.updateKV(idx, successor, succVal)
// 		newRightPageNum, err := t.delete(rightChild, successor)
// 		if err != nil {
// 			return 0, err
// 		}
// 		node.setPtr(idx+1, newRightPageNum)
// 		return t.copyToNewPage(node)
// 	}
// 	// case A3: both children underflowing, merge and delete
// 	if err := t.mergeChildren(node, idx); err != nil {
// 		return 0, err
// 	}
// 	return t.delete(node, k)
// }

// func (t *BTree) inorderPredecessor(node *Node) ([]byte, []byte) {
// 	for node.getType() != NodeTypeLeaf {
// 		pageNum := node.getPtr(node.getNKeys())
// 		node, _ = t.loadAsNode(pageNum)
// 	}
// 	k, v := node.getKV(node.getNKeys() - 1)
// 	return k, v
// }

// func (t *BTree) inorderSuccessor(node *Node) ([]byte, []byte) {
// 	for node.getType() != NodeTypeLeaf {
// 		pageNum := node.getPtr(0)
// 		node, _ = t.loadAsNode(pageNum)
// 	}
// 	k, v := node.getKV(0)
// 	return k, v
// }
