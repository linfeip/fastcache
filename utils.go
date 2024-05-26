package fastcache

import (
	"math/big"
	"unsafe"
)

// NodeTo node data convert to *T
func NodeTo[T any](node *DataNode) *T {
	dataPtr := uintptr(unsafe.Pointer(node)) + sizeOfDataNode
	return (*T)(unsafe.Pointer(dataPtr))
}

func ToDataNode(base uintptr, offset uint64) *DataNode {
	if offset == 0 {
		return nil
	}
	return (*DataNode)(unsafe.Pointer(base + uintptr(offset)))
}

func findNextPrime(n int) int {
	for {
		if big.NewInt(int64(n)).ProbablyPrime(0) {
			return n
		}
		n++
	}
}
