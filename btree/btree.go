// Package btree implements a persistent B-tree backed by a page file
package btree

import (
	"fmt"
	"os"
)

type BTree struct {
	root uint32
	pm   *pageManager
}

func NewBTree(filename string) (*BTree, error) {
	// open the main file
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
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
	// initialize master page
	if _, err := pm.allocate(); err != nil {
		return 0, err
	}
	// initialize root page. at this time, we only need to set the node type
	// but since leaf is represented by 0, no need to modify anything
	root, err := pm.allocate()
	return root, err
}

func (t *BTree) Insert(k, v []byte) error {
	// load the root node from disk
	root, err := t.pm.read(t.root)
	if err != nil {
		fmt.Printf("failed to load the root node: %v", err)
	}
	// from here, the wrapper takes over (node) and all operations
	// are thus performed on the wrapper
	node := NewNode(root)
	t.insertInSubtree(node)
	return nil
}

func (t *BTree) insertInSubtree(node *Node) {
	// TODO: hold preemption for now

	// peform the insertion
	switch node.getType() {
	// case a: internal node --> recurse
	case NODE_TYPE_INTERNAL:
		return
	// case b: leaf node --> insert
	case NODE_TYPE_LEAF:
		return
	}
}
