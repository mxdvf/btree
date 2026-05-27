package btree

import (
	"encoding/binary"
	"fmt"
)

func (t *BTree) print() {
	// performing a standard bfs
	queue := []uint32{t.root}
	level := 0
	for len(queue) != 0 {
		queueLen := len(queue)
		for i := range queueLen {
			pageNum := queue[i]
			buf, _ := t.pm.read(pageNum)
			n := NewNode(buf)
			// visual print logic
			fmt.Printf("=-----==-----Level: %d-----==-----= (node size: %v)\n", level, n.getSize())
			fmt.Println(string(n.data))
			fmt.Println("=-------==------==------==-------=")
			// only append children if the current node is internal
			if n.getType() == NodeTypeInternal {
				for idx := range n.getNKeys() + 1 {
					queue = append(queue, n.getPtr(idx))
				}
			}
		}
		// important: shift the queue forward so we don't re-process nodes
		queue = queue[queueLen:]
		level++
	}
}

func (t *BTree) loadAsNode(pageNum uint32) (*Node, error) {
	// load the root node from disk
	root, err := t.pm.read(pageNum)
	if err != nil {
		return nil, fmt.Errorf("failed to load the root node: %w", err)
	}
	// transform to node
	return NewNode(root), err
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
