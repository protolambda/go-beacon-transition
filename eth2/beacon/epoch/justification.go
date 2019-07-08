package epoch

import (
	. "github.com/protolambda/zrnt/eth2/beacon/components"
	. "github.com/protolambda/zrnt/eth2/core"
)

func ProcessEpochJustification(state *BeaconState) {
	currentEpoch := state.Epoch()
	if currentEpoch <= GENESIS_EPOCH+1 {
		return
	}
	previousEpoch := state.PreviousEpoch()
	// epoch numbers are trusted, no errors
	previousBoundaryBlockRoot, _ := state.GetBlockRoot(previousEpoch)
	currentBoundaryBlockRoot, _ := state.GetBlockRoot(currentEpoch)

	oldPreviousJustifiedEpoch := state.PreviousJustifiedEpoch
	oldCurrentJustifiedEpoch := state.CurrentJustifiedEpoch

	previousEpochBoundaryAttesterIndices := state.Validators.FilterUnslashed(state.GetAttesters(
		state.PreviousEpochAttestations,
		func(att *AttestationData) bool {
			return att.TargetRoot == previousBoundaryBlockRoot
		}))

	currentEpochBoundaryAttesterIndices := state.Validators.FilterUnslashed(state.GetAttesters(
		state.CurrentEpochAttestations,
		func(att *AttestationData) bool {
			return att.TargetRoot == currentBoundaryBlockRoot
		}))

	// Rotate current into previous
	state.PreviousJustifiedEpoch = state.CurrentJustifiedEpoch
	state.PreviousJustifiedRoot = state.CurrentJustifiedRoot
	// Rotate the justification bitfield up one epoch to make room for the current epoch
	state.JustificationBitfield <<= 1

	// Get the sum balances of the boundary attesters, and the total balance at the time.
	previousEpochBoundaryAttestingBalance := state.Validators.GetTotalEffectiveBalanceOf(previousEpochBoundaryAttesterIndices)
	previousTotalBalance := state.Validators.GetTotalEffectiveBalanceOf(state.Validators.GetActiveValidatorIndices(currentEpoch - 1))
	currentEpochBoundaryAttestingBalance := state.Validators.GetTotalEffectiveBalanceOf(currentEpochBoundaryAttesterIndices)
	currentTotalBalance := state.Validators.GetTotalEffectiveBalanceOf(state.Validators.GetActiveValidatorIndices(currentEpoch))

	// > Justification
	// If the previous epoch gets justified, fill the second last bit
	if previousEpochBoundaryAttestingBalance*3 >= previousTotalBalance*2 {
		state.CurrentJustifiedEpoch = previousEpoch
		state.CurrentJustifiedRoot = previousBoundaryBlockRoot
		state.JustificationBitfield |= 1 << 1
	}
	// If the current epoch gets justified, fill the last bit
	if currentEpochBoundaryAttestingBalance*3 >= currentTotalBalance*2 {
		state.CurrentJustifiedEpoch = currentEpoch
		state.CurrentJustifiedRoot = currentBoundaryBlockRoot
		state.JustificationBitfield |= 1 << 0
	}
	// > Finalization
	bitf := state.JustificationBitfield
	// The 2nd/3rd/4th most recent epochs are all justified, the 2nd using the 4th as source
	if (bitf>>1)&7 == 7 && state.PreviousJustifiedEpoch+3 == currentEpoch {
		state.FinalizedEpoch = oldPreviousJustifiedEpoch
		state.FinalizedRoot, _ = state.GetBlockRoot(state.FinalizedEpoch)
	}
	// The 2nd/3rd most recent epochs are both justified, the 2nd using the 3rd as source
	if (bitf>>1)&3 == 3 && state.PreviousJustifiedEpoch+2 == currentEpoch {
		state.FinalizedEpoch = oldPreviousJustifiedEpoch
		state.FinalizedRoot, _ = state.GetBlockRoot(state.FinalizedEpoch)
	}
	// The 1st/2nd/3rd most recent epochs are all justified, the 1st using the 3rd as source
	if (bitf>>0)&7 == 7 && state.CurrentJustifiedEpoch+2 == currentEpoch {
		state.FinalizedEpoch = oldCurrentJustifiedEpoch
		state.FinalizedRoot, _ = state.GetBlockRoot(state.FinalizedEpoch)
	}
	// The 1st/2nd most recent epochs are both justified, the 1st using the 2nd as source
	if (bitf>>0)&3 == 3 && state.CurrentJustifiedEpoch+1 == currentEpoch {
		state.FinalizedEpoch = oldCurrentJustifiedEpoch
		state.FinalizedRoot, _ = state.GetBlockRoot(state.FinalizedEpoch)
	}
}