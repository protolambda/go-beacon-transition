package components

import (
	"errors"
	. "github.com/protolambda/zrnt/eth2/core"
	"github.com/protolambda/zssz"
)

type HistoryState struct {
	LatestBlockHeader BeaconBlockHeader
	LatestBlockRoots  [SLOTS_PER_HISTORICAL_ROOT]Root
	LatestStateRoots  [SLOTS_PER_HISTORICAL_ROOT]Root
	HistoricalRoots   []Root
}

var HistoricalBatchSSZ = zssz.GetSSZ((*HistoricalBatch)(nil))

type HistoricalBatch struct {
	BlockRoots [SLOTS_PER_HISTORICAL_ROOT]Root
	StateRoots [SLOTS_PER_HISTORICAL_ROOT]Root
}

// Return the block root at the given slot (a recent one)
func (state *BeaconState) GetBlockRootAtSlot(slot Slot) (Root, error) {
	if !(slot < state.Slot && slot+SLOTS_PER_HISTORICAL_ROOT <= state.Slot) {
		return Root{}, errors.New("cannot get block root for given slot")
	}
	return state.LatestBlockRoots[slot%SLOTS_PER_HISTORICAL_ROOT], nil
}

// Return the block root at a recent epoch
func (state *BeaconState) GetBlockRoot(epoch Epoch) (Root, error) {
	return state.GetBlockRootAtSlot(epoch.GetStartSlot())
}