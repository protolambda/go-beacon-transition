package views

import (
	"fmt"
	. "github.com/protolambda/zrnt/experimental/tree"
)

type BitVectorType struct {
	BitLength uint64
}

func (cd *BitVectorType) DefaultNode() Node {
	bottomNodeCount := (cd.BitLength + 0xff) >> 8
	depth := GetDepth(bottomNodeCount)
	inner := &Commit{}
	inner.ExpandInplaceDepth(&ZeroHashes[0], depth)
	return inner
}

func (cd *BitVectorType) ViewFromBacking(node Node) View {
	bottomNodeCount := (cd.BitLength + 0xff) >> 8
	depth := GetDepth(bottomNodeCount)
	return &BitVectorView{
		SubtreeView: SubtreeView{
			BackingNode: node,
			depth:       depth,
		},
		BitVectorType: cd,
	}
}

func (cd *BitVectorType) New() *BitVectorView {
	return cd.ViewFromBacking(cd.DefaultNode()).(*BitVectorView)
}

type BitVectorView struct {
	SubtreeView
	*BitVectorType
}

func (cv *BitVectorView) ViewRoot(h HashFn) Root {
	return cv.BackingNode.MerkleRoot(h)
}

// Use .SubtreeView.Get(i) to work with the tree and bypass typing.
func (cv *BitVectorView) Get(i uint64) (bool, error) {
	if i >= cv.BitVectorType.BitLength {
		return false, fmt.Errorf("bitvector has bit length %d, cannot get bit index %d", cv.BitLength, i)
	}
	v, err := cv.SubtreeView.Get(i >> 8)
	if err != nil {
		return false, err
	}
	r, ok := v.(*Root)
	if !ok {
		return false, fmt.Errorf("bitvector bottom node is not a root, cannot get bit from it at bit index %d", i)
	}
	return (r[(i & 0xff) >> 3] >> (i & 7)) & 1 == 1, nil
}

// Use .SubtreeView.Set(i, v) to work with the tree and bypass typing.
func (cv *BitVectorView) Set(i uint64, value bool) error {
	if i >= cv.BitVectorType.BitLength {
		return fmt.Errorf("cannot set item at element index %d, bitvector only has %d bits", i, cv.BitLength)
	}
	v, err := cv.SubtreeView.Get(i >> 8)
	if err != nil {
		return err
	}
	r, ok := v.(*Root)
	if !ok {
		return fmt.Errorf("bitvector bottom node is not a root, cannot set bit at bit index %d", i)
	}
	// copy the old root, do not mutate the immutable.
	newRoot := *r
	if value {
		newRoot[(i & 0xff) >> 3] |= 1 << (i & 7)
	} else {
		newRoot[(i & 0xff) >> 3] &^= 1 << (i & 7)
	}
	return cv.SubtreeView.Set(i, &newRoot)
}
