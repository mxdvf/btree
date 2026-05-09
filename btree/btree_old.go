package btree

// func (n *Node) getNKeys() int {
// 	return int(binary.BigEndian.Uint16(n.data[2:]))
// }

// func (n *Node) addNKeys() {
// 	binary.BigEndian.PutUint16(n.data[2:], uint16(n.getNKeys()+1))
// }

// // TODO: we're assuming the key will fit into the node
// // TODO: we're assuming the key to be inserted is unique
// // TODO: we're assuming the key is not large enough to destroy a node (let's say node is empty and it occupies all the space)
// func (n *Node) insertKvPair(k, v []byte) {
// 	idxKV, offsetKV := n.getAppropriateIdx(k)

// 	n.insert(k, v, offsetKV)

// 	shift := n.postInsert()

// 	n.reEvaluateOffsetList(idxKV, offsetKV+shift)

// 	fmt.Println("intermediate checking -------------")
// 	n.debugPrint()
// 	fmt.Println("intermediate checking -------------")
// }

// func (n *Node) getAppropriateIdx(target []byte) (int, int) {
// 	// TODO: switch to binary search, right now
// 	// this is standard linear search
// 	var offsetKV, i int
// 	for i = 0; i < n.getNKeys(); i++ {
// 		offsetKV = n.getOffset(i)

// 		key := n.getKeyByOffset(offsetKV)
// 		fmt.Println("key ke swaad", key, i, offsetKV)

// 		if res := bytes.Compare(target, key); res == -1 || res == 0 {
// 			fmt.Println("obvio bhai", target, key)
// 			break
// 		}
// 	}

// 	// the array is empty, this is the first key to be added
// 	if offsetKV == 0 {
// 		offsetKV = HEADER_SIZE
// 	}

// 	return i, offsetKV
// }

// func (n *Node) getOffset(i int) int {
// 	pos := HEADER_SIZE + n.getNKeys()*PTR_SIZE + (i * OFFSET_SIZE)
// 	fmt.Println("isse pehle ki koi date nhi mil sakti", pos, i)
// 	fmt.Println("wow - ", int(binary.BigEndian.Uint16(n.data[pos:])))
// 	// n.debugPrint()
// 	return int(binary.BigEndian.Uint16(n.data[pos:]))
// }

// func (n *Node) getKeyByOffset(offset int) []byte {
// 	keyLen := binary.BigEndian.Uint16(n.data[offset:])
// 	start := offset + KEY_LEN_SIZE
// 	end := offset + KEY_LEN_SIZE + int(keyLen)
// 	return n.data[start:end]
// }

// func (n *Node) insert(k, v []byte, offset int) {
// 	nextOffset := offset + KEY_LEN_SIZE + len(k) + VAL_LEN_SIZE + len(v)
// 	copy(n.data[nextOffset:], n.data[offset:])

// 	start := offset
// 	binary.BigEndian.PutUint16(n.data[start:], uint16(len(k)))

// 	start = start + KEY_LEN_SIZE
// 	end := start + len(k)
// 	copy(n.data[start:end+1], k)

// 	start = start + len(k)
// 	binary.BigEndian.PutUint16(n.data[start:], uint16(len(v)))

// 	start = start + VAL_LEN_SIZE
// 	end = start + len(v)
// 	copy(n.data[start:end+1], v)

// 	// return end - offset // total length
// }

// func (n *Node) postInsert() int {
// 	// shift everything by (PTR_SIZE + OFFSET_SIZE = 6B) starting at the first KV pair
// 	offset := HEADER_SIZE + n.getNKeys()*PTR_SIZE + n.getNKeys()*OFFSET_SIZE
// 	shift := PTR_SIZE + OFFSET_SIZE
// 	copy(n.data[offset+shift:], n.data[offset:])
// 	clear(n.data[offset : offset+shift+1])
// 	// add 1 key
// 	n.addNKeys()

// 	fmt.Println("DEKH TOH LOON EK BAAR", offset+shift)

// 	return shift
// }

// func (n *Node) reEvaluateOffsetList(idx, offsetKV int) {
// 	offsetIdx := HEADER_SIZE + n.getNKeys()*PTR_SIZE + (idx)*OFFSET_SIZE
// 	binary.BigEndian.PutUint16(n.data[offsetIdx:], uint16(offsetKV))
// }
