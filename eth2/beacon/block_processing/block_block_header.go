package block_processing

import (
	"errors"
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zrnt/eth2/util/bls"
	"github.com/protolambda/zrnt/eth2/util/ssz"
)

func ProcessBlockHeader(state *beacon.BeaconState, block *beacon.BeaconBlock) error {
	// Verify that the slots match
	if block.Slot != state.Slot {
		return errors.New("slot of block does not match slot of state")
	}
	// Verify that the parent matches
	if block.PreviousBlockRoot != ssz.SigningRoot(state.LatestBlockHeader) {
		return errors.New("previous block root does not match root from latest state block header")
	}
	// Save current block as the new latest block
	state.LatestBlockHeader = block.GetTemporaryBlockHeader()

	propIndex := state.GetBeaconProposerIndex(state.Slot)
	proposer := state.ValidatorRegistry[propIndex]
	// Verify proposer is not slashed
	if proposer.Slashed {
		return errors.New("cannot accept block header from slashed proposer")
	}
	// Block signature
	if !bls.BlsVerify(
		proposer.Pubkey,
		ssz.SigningRoot(block),
		block.Signature,
		beacon.GetDomain(state.Fork, state.Epoch(), beacon.DOMAIN_BEACON_BLOCK)) {
		return errors.New("block signature invalid")
	}
	return nil
}
