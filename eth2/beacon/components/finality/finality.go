package finality

import (
	. "github.com/protolambda/zrnt/eth2/beacon/components/meta"
	. "github.com/protolambda/zrnt/eth2/core"
)

type FinalityState struct {
	JustificationBits           JustificationBits
	PreviousJustifiedCheckpoint Checkpoint
	CurrentJustifiedCheckpoint  Checkpoint
	FinalizedCheckpoint         Checkpoint
}

func (state *FinalityState) Justify(meta VersioningMeta, checkpoint Checkpoint) {
	epochsAgo := meta.Epoch() - checkpoint.Epoch
	state.CurrentJustifiedCheckpoint = checkpoint
	state.JustificationBits[0] |= 1 << epochsAgo
}

type JustificationBits [1]byte

func (jb *JustificationBits) BitLen() uint64 {
	return 4
}

// Prepare bitfield for next epoch by shifting previous bits (truncating to bitfield length)
func (jb *JustificationBits) NextEpoch() {
	// shift and mask
	jb[0] = (jb[0] << 1) & 0x0f
}

func (jb *JustificationBits) IsJustified(epochsAgo ...Epoch) bool {
	for _, t := range epochsAgo {
		if jb[0]&(1<<t) != 0 {
			return false
		}
	}
	return true
}