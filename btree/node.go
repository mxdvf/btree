package btree

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	NODE_TYPE_LEAF uint16 = iota
	NODE_TYPE_INTERNAL
)

const (
	PAGE_SIZE = 4096 // 4096 bytes

	HEADER_SIZE  = 4 // 2 + 2 = 4 bytes
	PTR_SIZE     = 4 // 4 bytes
	OFFSET_SIZE  = 2 // 2 bytes
	KEY_LEN_SIZE = 2 // 2 bytes
	VAL_LEN_SIZE = 2 // 2 bytes
)

type Node struct {
	// wire format:
	// type  |  nkeys |   pointers  |  offset-array	 |		     key-values
	//  2B   |   2B   |  nkeys * 4B |  	nkeys * 2B	 |  [klen: 2B][k][vlen: 2B][v]
	data []byte
}

// assumptions:
// 1. duplicate keys are not allowed
// 2. offset list contains relative positioning: check kvPos()
// 3. offset list points to the start of the next KV pair (a logically empty space to start from)

func NewNode(buf []byte) *Node {
	return &Node{data: buf}
}

func debugPrint(node *Node, top int) string {
	res := fmt.Sprintf("printing top %d bytes (0-%d)\n", top, top-1)
	res += fmt.Sprintf("%v", node.data[:top])
	return res
}

func (node *Node) getSize() uint16 {
	lastIdx := node.getNKeys() - 1
	kvRangeLen := node.getOffset(lastIdx) + node.getKVLen(lastIdx)
	return node.getHeaderAndMetadataLen() + kvRangeLen
}

func (node *Node) getType() uint16 {
	return binary.BigEndian.Uint16(node.data[0:])
}

func (node *Node) getNKeys() uint16 {
	return binary.BigEndian.Uint16(node.data[2:])
}

func (node *Node) incrementNKeys() {
	binary.BigEndian.PutUint16(node.data[2:], node.getNKeys()+1)
}

func (node *Node) getHeaderAndMetadataLen() uint16 {
	return HEADER_SIZE + PTR_SIZE*(node.getNKeys()+1) + OFFSET_SIZE*node.getNKeys()
}

func (node *Node) offsetPos(idx uint16) uint16 {
	return HEADER_SIZE + PTR_SIZE*(node.getNKeys()+1) + OFFSET_SIZE*idx
}

func (node *Node) getOffset(idx uint16) uint16 {
	pos := node.offsetPos(idx)
	return binary.BigEndian.Uint16(node.data[pos:])
}

func (node *Node) kvPos(idx uint16) uint16 {
	return HEADER_SIZE + PTR_SIZE*(node.getNKeys()+1) + OFFSET_SIZE*node.getNKeys() + node.getOffset(idx)
}

func (node *Node) getKV(idx uint16) ([]byte, []byte) {
	// following a seek based approach, we extract what we need
	// and move the seek forward
	pos := node.kvPos(idx)
	// perform action & move forward
	klen := binary.BigEndian.Uint16(node.data[pos:])
	pos += KEY_LEN_SIZE
	// perform action & move forward
	key := node.data[pos : pos+klen]
	pos += klen
	// perform action & move forward
	vlen := binary.BigEndian.Uint16(node.data[pos:])
	pos += VAL_LEN_SIZE
	// perform action & move forward
	val := node.data[pos : pos+vlen]
	return key, val
}

func (node *Node) putKV(k, v []byte, pos uint16) {
	// a seek based approach
	binary.BigEndian.PutUint16(node.data[pos:], uint16(len(k)))
	pos += KEY_LEN_SIZE
	// perform action & move forward
	copy(node.data[pos:pos+uint16(len(k))], k)
	pos += uint16(len(k))
	// perform action & move forward
	binary.BigEndian.PutUint16(node.data[pos:], uint16(len(v)))
	pos += VAL_LEN_SIZE
	// perform action
	copy(node.data[pos:pos+uint16(len(v))], v)
}

func (node *Node) shiftKVRight(totalKVLen, pos uint16) {
	copy(node.data[pos+totalKVLen:], node.data[pos:])
	clear(node.data[pos : pos+totalKVLen])
}

func (node *Node) getKVLen(idx uint16) uint16 {
	pos := node.kvPos(idx)
	klen := binary.BigEndian.Uint16(node.data[pos:])
	vlen := binary.BigEndian.Uint16(node.data[pos+KEY_LEN_SIZE+klen:])
	return klen + vlen + KEY_LEN_SIZE + VAL_LEN_SIZE
}

func (node *Node) ptrPos(idx uint16) uint16 {
	return HEADER_SIZE + PTR_SIZE*idx
}

func (node *Node) getPtr(idx uint16) uint32 {
	pos := HEADER_SIZE + PTR_SIZE*idx
	return binary.BigEndian.Uint32(node.data[pos:])
}

func (node *Node) setPtr(idx uint16, ptr uint32) {
	pos := HEADER_SIZE + PTR_SIZE*idx
	binary.BigEndian.PutUint32(node.data[pos:], ptr)
}

func (node *Node) findInsertPos(target []byte) (uint16, uint16) {
	if node.getNKeys() <= 0 {
		return 0, node.kvPos(0)
	}
	// loop over all keys to find the appropriate insertion position
	var idx uint16
	for idx = 0; idx < node.getNKeys(); idx++ {
		k, _ := node.getKV(idx)
		if res := bytes.Compare(target, k); res == -1 || res == 0 {
			return idx, node.kvPos(idx)
		}
	}
	// for keys that are the largest in the range would be inserted
	// after the last kv pair and hence the offset needs to be calculated
	// manually
	return idx, node.kvPos(idx-1) + node.getKVLen(idx-1)
}

func (node *Node) shiftPtrAndOffsetRight(idx uint16) {
	// make space for new kv's pointer
	ptrPos := node.ptrPos(idx)
	copy(node.data[ptrPos+PTR_SIZE:], node.data[ptrPos:])
	clear(node.data[ptrPos : ptrPos+PTR_SIZE])
	// make space for new kv's offset
	offsetPos := node.offsetPos(idx)
	copy(node.data[offsetPos+OFFSET_SIZE:], node.data[offsetPos:])
	clear(node.data[offsetPos : offsetPos+OFFSET_SIZE])
}

func (node *Node) reEvaluateOffsetList(idx, calculatedPos, totalLen uint16) {
	for i := uint16(0); i < node.getNKeys(); i++ {
		pos := node.offsetPos(i)
		offsetBeforeUpdate := node.getOffset(i)
		switch {
		// anything at idx, simply insert the calculated offset.
		// there's no workaround for this beecause when inserting a key, we calculated the offset
		// for the very first time. think about this.
		case i == idx:
			binary.BigEndian.PutUint16(node.data[pos:], calculatedPos)
		// anything before idx requires no update
		// anything after idx we just need to add totalKVLen
		case i > idx:
			binary.BigEndian.PutUint16(node.data[pos:], offsetBeforeUpdate+totalLen)
		}
	}
}

func (node *Node) getTotalLenPostInsert(k, v []byte) uint16 {
	return uint16(len(k)+len(v)) + OFFSET_SIZE + PTR_SIZE + KEY_LEN_SIZE + VAL_LEN_SIZE
}

func (node *Node) insert(k, v []byte) (uint16, error) {
	// if node.full(k, v) {
	// 	return 0, fmt.Errorf("node does not have enough space")
	// }
	insertIdx, insertPos := node.findInsertPos(k)
	// increment nkeys (do not re-order, everything
	// after this line depends on it being here)
	node.incrementNKeys()
	// make space for pointer, offset
	node.shiftPtrAndOffsetRight(insertIdx)
	insertPos += PTR_SIZE + OFFSET_SIZE // update to new position, also when node has 1 key, it has 2 pointers and 1 offset
	// make space for the new key
	totalLen := uint16(len(k) + len(v) + KEY_LEN_SIZE + VAL_LEN_SIZE)
	node.shiftKVRight(totalLen, insertPos)
	// insert kv at that position
	node.putKV(k, v, insertPos)
	// update offset list with the newly added kv pair and also fix other offsets
	insertPos -= node.getHeaderAndMetadataLen() // update insertPos to a relative offset before updating the list
	node.reEvaluateOffsetList(insertIdx, insertPos, totalLen)
	return insertIdx, nil
}

func (node *Node) drySplit() (*Node, *Node, uint16) {
	// check for the median key
	medianIndex := node.getNKeys() / 2
	// initialize a new node
	leftNode := NewNode(make([]byte, 4096)) // TODO: should not create a new left node
	rightNode := NewNode(make([]byte, 4096))
	// set node type
	binary.BigEndian.PutUint16(rightNode.data[0:], node.getType())
	binary.BigEndian.PutUint16(leftNode.data[0:], node.getType())
	// from here on, we will operate on right half of each component of the node
	// using (medianIndex+1) which involves extracting kv range, offset list,
	// pointers and also reducing nkeys
	for idx := uint16(0); idx < medianIndex; idx++ {
		k, v := node.getKV(idx)
		leftNode.insert(k, v)
		leftNode.setPtr(idx, node.getPtr(idx))
	}
	for idx := medianIndex + 1; idx < node.getNKeys(); idx++ {
		// kv range work
		k, v := node.getKV(idx)
		rightNode.insert(k, v) // TODO: this is very slow, instead you must try to copy entire range at once and shift it to the new node
		// pointer work
		rightNode.setPtr(idx-medianIndex-1, node.getPtr(idx))
	}
	// return
	return leftNode, rightNode, medianIndex
}

func (n *Node) full(k, v []byte) bool {
	return n.getSize()+n.getTotalLenPostInsert(k, v) > PAGE_SIZE
}
