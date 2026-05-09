package btree

import (
	"encoding/binary"
	"fmt"
)

const (
	NODE_TYPE_LEAF int = iota
	NODE_TYPE_INTERNAL
)

const (
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

func NewLeafNode() *Node {
	return newNode(NODE_TYPE_LEAF)
}

func NewInternalNode() *Node {
	return newNode(NODE_TYPE_INTERNAL)
}

func newNode(t int) *Node {
	n := &Node{data: make([]byte, 4096)}
	binary.BigEndian.PutUint16(n.data[0:], uint16(t))
	return n
}

func (n *Node) Insert(k, v []byte) {
	// if nkeys=0, insert manually because most probably this is the first key ever
	if n.getNKeys() <= 0 {
		n.insertFirstKV(k, v)
		// TODO: update offset list
		n.incrementNKeys()
		return
	}
	// else, insert maintaing the invariants
	n.insert(k, v)
}

func (n *Node) getNKeys() int {
	return int(binary.BigEndian.Uint16(n.data[2:]))
}

func (n *Node) insertFirstKV(k, v []byte) {
	start := HEADER_SIZE + PTR_SIZE + OFFSET_SIZE
	binary.BigEndian.PutUint16(n.data[start:], uint16(len(k)))

	start = start + KEY_LEN_SIZE
	end := start + len(k)
	copy(n.data[start:end+1], k)

	start = end
	binary.BigEndian.PutUint16(n.data[start:], uint16(len(v)))

	start = start + VAL_LEN_SIZE
	end = start + len(v)
	copy(n.data[start:end+1], v)
}

func (n *Node) incrementNKeys() {
	binary.BigEndian.PutUint16(n.data[2:], uint16(n.getNKeys())+1)
}

func (n *Node) insert(k, v []byte) {
	// 0. make this logic robust so we could shift entire insertFirstKV over here otherwise there would be duplicate logic
	// we can create a if here itself, if nkeys=0, no need to calculate index it would be given 0 because anyways if a very small
	// key comes in then it might be possible that we put it at 0 so it's fine to handle everything in one place

	// 1. figure out where to put the key
	// think cases: it could be before the first key or even after the first key so think hard think hard (but
	// remember to maintain both skillfully a right/left shift will help significantly)

	// 2. as soon as idx is found, first shift everything by 6 to make space for pointer and offset, this is simple and deterministic

	// 3. then insert the kv and use the returned offset which will then be inserted at the same idx calculated above in (2) into offset-array

	// 4. increment nkeys
}

func (n *Node) debugPrint() {
	fmt.Println("only showing top 50 bytes:")
	fmt.Println(n.data[:50])
}
