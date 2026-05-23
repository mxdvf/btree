// Package btree implements a persistent B-tree backed by a page file
package btree

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type BTree struct {
	root uint32
	pm   *pageManager
}

func NewBTree(filename string) (*BTree, error) {
	// open the main file
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open the main file: %w", err)
	}
	// initialize root and master pages
	nm := newPageManager(fd)
	root, err := initializeRootAndMasterPage(nm)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize root and master nodes: %v", err)
	}
	// initialize btree
	return &BTree{
		root: root,
		pm:   newPageManager(fd),
	}, nil
}

func initializeRootAndMasterPage(pm *pageManager) (uint32, error) {
	buf, err := pm.read(0)
	switch err {
	// if there's EOF error, then the master page does not exist
	case io.EOF:
		if _, err := pm.allocate(); err != nil {
			return 0, err
		}
		// initialize root page
		root, err := pm.allocate()
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
	// if uint16(len(k)+len(v))+KEY_LEN_SIZE+VAL_LEN_SIZE > PAGE_SIZE/2 {
	// 	return fmt.Errorf("key+value too large")
	// }
	// load the root node from disk
	root, err := t.pm.read(t.root)
	if err != nil {
		return fmt.Errorf("failed to load the root node: %w", err)
	}
	rootNode := NewNode(root)
	// special case for fixing an overfull root node
	if rootNode.full(k, v) { // TODO: this condition is a bit shaky, considering in-mem implementation, it should be something along the lines of "root == 2*t-1" because we shouldn't wait for a key to be inserted before root is split
		newRootPageNum, err := t.splitRoot(rootNode)
		if err != nil {
			return fmt.Errorf("failed to split the root: %w", err)
		}
		t.pointMasterToNewRoot(newRootPageNum)
		// reload the new root and continue
		root, err = t.pm.read(t.root)
		if err != nil {
			return err
		}
		rootNode = NewNode(root)
	}
	// from here, the wrapper takes over (node) and all operations
	// are thus performed on the wrapper
	pageNum, err := t.insertInSubtree(rootNode, k, v)
	if err != nil {
		return fmt.Errorf("failed to insert the key: %w", err)
	}
	// update master page using the pageNum root page
	t.pointMasterToNewRoot(pageNum)
	return nil
}

func (t *BTree) splitRoot(root *Node) (uint32, error) {
	// split root into left and right
	left, right, medianIndex := root.drySplit()
	// persist the right node to disk
	rightPageNum, err := t.copyToNewPage(right)
	if err != nil {
		return 0, fmt.Errorf("could not persist the right node while splitting root: %w", err)
	}
	// persist left child
	leftPageNum, err := t.copyToNewPage(left) // TODO: a new left node should not be created, it should be manipulated to remove unnecessary data
	if err != nil {
		return 0, fmt.Errorf("could not persist the left node while splitting root: %w", err)
	}
	// create new internal root
	buf := make([]byte, PAGE_SIZE)
	newRoot := NewNode(buf)
	// set type to internal
	binary.BigEndian.PutUint16(newRoot.data[0:], NODE_TYPE_INTERNAL)
	// insert median key into new root
	medianKey, medianVal := root.getKV(medianIndex)
	newRoot.insert(medianKey, medianVal)
	// set pointers to left and right children
	newRoot.setPtr(0, leftPageNum)
	newRoot.setPtr(1, rightPageNum)
	// allocate and write new root
	return t.copyToNewPage(newRoot)
}

func (t *BTree) pointMasterToNewRoot(pageNum uint32) error {
	// read master page
	buf, err := t.pm.read(0)
	if err != nil {
		return fmt.Errorf("failed to read the master page: %w", err)
	}
	// update master page pointer to root
	binary.BigEndian.PutUint32(buf[0:], pageNum)
	// write back master page to disk
	if err := t.pm.write(0, buf); err != nil {
		return fmt.Errorf("failed to write to master page: %w", err)
	}
	// also update the in-mem pointer
	t.root = pageNum
	return nil
}

func (t *BTree) insertInSubtree(node *Node, k, v []byte) (uint32, error) {
	// preemptive fix before ever touching a child node
	t.preemptiveFix(node, k, v)

	switch node.getType() {
	case NODE_TYPE_LEAF:
		// simple logic to insert into the node and perform a
		// copy-on-write
		return t.handleInsertionInLeafNode(node, k, v)
	case NODE_TYPE_INTERNAL:
		// orchestrator logic to enter into the correct subtree page
		// recursively insert on that subtree's root, update all page nums
		// upwards

		return t.handleInsertionInInternalNode(node, k, v)
	}

	panic("should not have reached this point")
}

func (t *BTree) preemptiveFix(node *Node, k, v []byte) error {
	// 1. find the appropriate child that you're about to enter into
	idx, _ := node.findInsertPos(k)
	// 2. load that child into a node
	appropriateChildPageNum := node.getPtr(idx)
	appropriateChild, err := t.pm.read(appropriateChildPageNum)
	if err != nil {
		return fmt.Errorf("could not read the appropriate subtree's page: %w", err)
	}
	child := NewNode(appropriateChild)
	// 3. check if it's full, if yes break it down into 2 nodes
	if child.full(k, v) {
		rightChild, leftChild, medianIndex := child.drySplit()
		// persist the right node to disk
		rightPageNum, err := t.copyToNewPage(rightChild)
		if err != nil {
			return fmt.Errorf("could not persist the right node during preemptive fix: %w", err)
		}
		// persist left child
		leftPageNum, err := t.copyToNewPage(leftChild) // TODO: a new left node should not be created, it should be manipulated to remove unnecessary data
		if err != nil {
			return fmt.Errorf("could not persist the left node during preemptive fix: %w", err)
		}
		// insert median key into new root
		medianKey, medianVal := child.getKV(medianIndex)
		idx, err := node.insert(medianKey, medianVal)
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
	if _, err := node.insert(k, v); err != nil {
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
	appropriateSubtree, err := t.pm.read(appropriateSubtreePageNum)
	if err != nil {
		return 0, fmt.Errorf("could not read the appropriate subtree's page: %w", err)
	}
	childPageNum, err := t.insertInSubtree(NewNode(appropriateSubtree), k, v)
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

func (t *BTree) copyToNewPage(node *Node) (uint32, error) {
	pageNum, err := t.pm.allocate()
	if err != nil {
		return 0, err
	}
	// write the updated bytes to the newly allocated page
	if err := t.pm.write(pageNum, node.data); err != nil {
		return 0, err
	}
	// return the newly allocated page num
	return pageNum, nil
}

// func (t *BTree) Search(k []byte) ([]byte, error) {
// 	root, err := t.pm.read(t.root)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read the root: %v", err)
// 	}
// 	rootNode := NewNode(root)
// 	return t.search(rootNode, k)
// }

// func (t *BTree) search(node *Node, target []byte) ([]byte, error) {
// 	// this function will give us an index such that target <= some_key_in_node
// 	idx, _ := node.findInsertPos(target)
// 	// assuming the key is in the node, we will receive an index that is within bounds
// 	// because if the key is out of bounds then it means that we must traverse down. the
// 	// only reason we can receive an out of bound index is because we're using the a helper
// 	// for insertion which can return out of bound index if the key is larger than all keys
// 	// present in the node
// 	nKeys := node.getNKeys()
// 	if idx < nKeys {
// 		k, v := node.getKV(idx)
// 		if res := bytes.Compare(k, target); res == 0 {
// 			return v, nil
// 		}
// 	}
// 	// if the node type we're operating on is a leaf, then we end the search
// 	if node.getType() == NODE_TYPE_LEAF {
// 		return nil, fmt.Errorf("reached the end of the tree and couldn't find the key")
// 	}
// 	// ptr[idx] is always the correct child because as for pointers, they are always 1 more
// 	// than the # of keys, so it works for: (1) idx < nkeys, and (2) idx = nkeys because we will
// 	// always receive the correct pointer
// 	pageNum := node.getPtr(idx)
// 	buf, err := t.pm.read(pageNum)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read page: %v", err)
// 	}
// 	// recursively search the subtree
// 	return t.search(NewNode(buf), target)
// }
