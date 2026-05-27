package btree

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
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
	if len(k)+len(v) > MaxAllowedKVLen {
		return fmt.Errorf("key+value too large")
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
	if rootNode.full() {
		rootNode, err = t.setupNewRoot(k, rootNode)
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

func (t *BTree) setupNewRoot(k []byte, rootNode *Node) (*Node, error) {
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
		t.preemptiveFix(node, k, v)
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
	if childNode.full() {
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
