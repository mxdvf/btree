package btree

import (
	"bytes"
	"encoding/binary"
)

type Node struct {
	// wire format:
	// type  |  nkeys |   pointers  |  offset-array	 |		     key-values
	//  2B   |   2B   |  nkeys * 4B |  	nkeys * 2B	 |  [klen: 2B][k][vlen: 2B][v]
	data []byte
}

// assumptions:
// 1. duplicate keys are rewritten
// 2. offset list contains relative positioning: check kvPos()
// 3. offset list points to the start of the next to-be-inserted KV pair (a logically empty space to start from)

func NewNode(buf []byte) *Node {
	return &Node{data: buf}
}

func (node *Node) getSize() uint16 {
	lastIdx := node.getNKeys() - 1
	kvRelativeLen := node.getOffset(lastIdx) + node.getKVLen(lastIdx)
	return node.getHeaderAndMetadataLen() + kvRelativeLen
}

func (node *Node) getType() uint16 {
	return binary.BigEndian.Uint16(node.data[0:])
}

func (node *Node) setType(t uint16) {
	binary.BigEndian.PutUint16(node.data[0:], t)
}

func (node *Node) getNKeys() uint16 {
	return binary.BigEndian.Uint16(node.data[2:])
}

func (node *Node) incrementNKeys() {
	binary.BigEndian.PutUint16(node.data[2:], node.getNKeys()+1)
}

func (node *Node) decrementNKeys() {
	binary.BigEndian.PutUint16(node.data[2:], node.getNKeys()-1)
}

func (node *Node) getHeaderAndMetadataLen() uint16 {
	return HeaderSize + PointerSize*(node.getNKeys()+1) + OffsetSize*node.getNKeys()
}

func (node *Node) offsetPos(idx uint16) uint16 {
	return HeaderSize + PointerSize*(node.getNKeys()+1) + OffsetSize*idx
}

func (node *Node) getOffset(idx uint16) uint16 {
	pos := node.offsetPos(idx)
	return binary.BigEndian.Uint16(node.data[pos:])
}

func (node *Node) kvPos(idx uint16) uint16 {
	return HeaderSize + PointerSize*(node.getNKeys()+1) + OffsetSize*node.getNKeys() + node.getOffset(idx)
}

func (node *Node) getKV(idx uint16) ([]byte, []byte) {
	// following a seek based approach, we extract what we need
	// and move the seek forward
	pos := node.kvPos(idx)
	// perform action & move forward
	klen := binary.BigEndian.Uint16(node.data[pos:])
	pos += KeyLenSize
	// perform action & move forward
	key := node.data[pos : pos+klen]
	pos += klen
	// perform action & move forward
	vlen := binary.BigEndian.Uint16(node.data[pos:])
	pos += ValLenSize
	// perform action & move forward
	val := node.data[pos : pos+vlen]
	return key, val
}

func (node *Node) setKV(k, v []byte, pos uint16) {
	// a seek based approach
	binary.BigEndian.PutUint16(node.data[pos:], uint16(len(k)))
	pos += KeyLenSize
	// perform action & move forward
	copy(node.data[pos:pos+uint16(len(k))], k)
	pos += uint16(len(k))
	// perform action & move forward
	binary.BigEndian.PutUint16(node.data[pos:], uint16(len(v)))
	pos += ValLenSize
	// perform action
	copy(node.data[pos:pos+uint16(len(v))], v)
}

func (node *Node) getKVLen(idx uint16) uint16 {
	pos := node.kvPos(idx)
	klen := binary.BigEndian.Uint16(node.data[pos:])
	vlen := binary.BigEndian.Uint16(node.data[pos+KeyLenSize+klen:])
	return klen + vlen + KeyLenSize + ValLenSize
}

func (node *Node) ptrPos(idx uint16) uint16 {
	return HeaderSize + PointerSize*idx
}

func (node *Node) getPtr(idx uint16) uint32 {
	// TODO: why would i ever use idx=nkeys(), think about it, i guess i am missing something
	pos := HeaderSize + PointerSize*idx
	return binary.BigEndian.Uint32(node.data[pos:])
}

func (node *Node) setPtr(idx uint16, ptr uint32) {
	pos := HeaderSize + PointerSize*idx
	binary.BigEndian.PutUint32(node.data[pos:], ptr)
}

func (node *Node) getTotalLenIfInserted(k, v []byte) uint16 {
	return uint16(len(k)+len(v)) + OffsetSize + PointerSize + KeyLenSize + ValLenSize
}

func (n *Node) overflow() bool {
	// a node overflows when it no longer has room for the worst-case key
	// (aka one that's 1344B) that could be promoted from its child into itself
	// during a split which means a median key of max size. this was a bug that
	// took me 6 days to figure out, although i admit it was an oversight on my part.
	return n.getSize() > PageSize-MaxAllowedKVLen
}

func (node *Node) underflow() bool {
	// minimum degree t=2, so minimum keys = t-1 = 1
	// a node underflows when it has only 1 key and we need to delete from it
	return node.getNKeys() <= 1
}

// below are almost all insertion related methods

func (node *Node) drySplit() (*Node, *Node, uint16) {
	// check for the median key
	medianIndex := node.getNKeys() / 2
	// initialize a new node
	leftNode := NewNode(make([]byte, 4096)) // TODO: should not create a new left node, instead manipulate the left node itself to make space
	rightNode := NewNode(make([]byte, 4096))
	// set node type
	rightNode.setType(node.getType())
	leftNode.setType(node.getType())
	// from here on, we will operate on the left half of each component of the node
	// i.e [0, medianIndex), but remember pointers is always 1 more than nkeys
	for idx := uint16(0); idx < medianIndex; idx++ {
		k, v := node.getKV(idx)
		leftNode.insertSelf(k, v)
	}
	// pointer always goes one more than the nunber of keys, hence different loop
	for idx := uint16(0); idx < medianIndex+1; idx++ {
		leftNode.setPtr(idx, node.getPtr(idx))
	}
	// from here, we will operate on the right half of each component of the node
	// i.e [medianIndex+1, nkeys), but again pointers is always 1 more than nkeys
	for idx := medianIndex + 1; idx < node.getNKeys(); idx++ {
		k, v := node.getKV(idx)
		rightNode.insertSelf(k, v)
	}
	for idx := medianIndex + 1; idx < node.getNKeys()+1; idx++ {
		rightNode.setPtr(idx-medianIndex-1, node.getPtr(idx))
	}
	// return
	return leftNode, rightNode, medianIndex
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

func (node *Node) insertSelf(k, v []byte) (uint16, error) {
	if node.getSize()+node.getTotalLenIfInserted(k, v) >= PageSize {
		panic("illegal node, it should have been split by a preemptive fix")
	}
	// find insertion point
	insertIdx, insertPos := node.findInsertPos(k)
	// increment nkeys
	node.incrementNKeys()
	// make space for ptr, offset, then kv
	kvLen := uint16(len(k) + len(v) + KeyLenSize + ValLenSize)
	node.makeSpace(insertIdx, insertPos, kvLen)
	// put kv there
	insertPos += PointerSize + OffsetSize
	node.setKV(k, v, insertPos)
	// fix offsets for everyone using a relative offset pos
	insertPos -= node.getHeaderAndMetadataLen() // update insertPos to a relative offset before updating the list
	node.reEvaluateOffsetList(insertIdx, insertPos, kvLen)
	return insertIdx, nil
}

func (node *Node) makeSpace(idx, pos, kvLen uint16) {
	// make space for new kv's pointer
	ptrPos := node.ptrPos(idx)
	copy(node.data[ptrPos+PointerSize:], node.data[ptrPos:])
	clear(node.data[ptrPos : ptrPos+PointerSize])
	// make space for new kv's offset
	offsetPos := node.offsetPos(idx)
	copy(node.data[offsetPos+OffsetSize:], node.data[offsetPos:])
	clear(node.data[offsetPos : offsetPos+OffsetSize])
	// make space for the kv pair
	pos += PointerSize + OffsetSize
	copy(node.data[pos+kvLen:], node.data[pos:])
	clear(node.data[pos : pos+kvLen])
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

// below are almost all deletion related methods
