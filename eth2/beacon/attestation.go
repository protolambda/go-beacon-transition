package beacon

import (
	"context"
	"errors"
	"fmt"
	"github.com/protolambda/ztyp/tree"
	. "github.com/protolambda/ztyp/view"
	"sort"
)

func (c *Phase0Config) BlockAttestations() ListTypeDef {
	return ListType(c.Attestation(), c.MAX_ATTESTATIONS)
}

func (c *Phase0Config) Attestation() *ContainerTypeDef {
	return ContainerType("Attestation", []FieldDef{
		{"aggregation_bits", c.CommitteeBits()},
		{"data", AttestationDataType},
		{"signature", BLSSignatureType},
	})
}

type Attestation struct {
	AggregationBits CommitteeBitList
	Data            AttestationData
	Signature       BLSSignature
}

func (a *Attestation) HashTreeRoot(hFn tree.HashFn) Root {
	return hFn.HashTreeRoot(&a.AggregationBits, &a.Data, a.Signature)
}

type Attestations struct {
	Items []Attestation
	Limit uint64
}

func (li *Attestations) HashTreeRoot(hFn tree.HashFn) Root {
	length := uint64(len(li.Items))
	return hFn.ComplexListHTR(func(i uint64) tree.HTR {
		if i < length {
			return &li.Items[i]
		}
		return nil
	}, length, li.Limit)
}

func (spec *Spec) ProcessAttestations(ctx context.Context, epc *EpochsContext, state *BeaconStateView, ops []Attestation) error {
	for i := range ops {
		select {
		case <-ctx.Done():
			return TransitionCancelErr
		default: // Don't block.
			break
		}
		if err := spec.ProcessAttestation(state, epc, &ops[i]); err != nil {
			return err
		}
	}
	return nil
}

func (spec *Spec) ProcessAttestation(state *BeaconStateView, epc *EpochsContext, attestation *Attestation) error {
	data := &attestation.Data

	// Check slot
	currentSlot, err := state.Slot()
	if err != nil {
		return err
	}
	if !(currentSlot <= data.Slot+spec.SLOTS_PER_EPOCH) {
		return errors.New("attestation slot is too old")
	}
	if !(data.Slot+spec.MIN_ATTESTATION_INCLUSION_DELAY <= currentSlot) {
		return errors.New("attestation is too new")
	}

	currentEpoch := spec.SlotToEpoch(currentSlot)
	previousEpoch := currentEpoch.Previous()

	// Check target
	if data.Target.Epoch < previousEpoch {
		return errors.New("attestation data is invalid, target is too far in past")
	} else if data.Target.Epoch > currentEpoch {
		return errors.New("attestation data is invalid, target is in future")
	}
	// And if it matches the slot
	if data.Target.Epoch != spec.SlotToEpoch(data.Slot) {
		return errors.New("attestation data is invalid, slot epoch does not match target epoch")
	}

	// Check committee index
	if commCount, err := epc.GetCommitteeCountAtSlot(data.Slot); err != nil {
		return err
	} else if uint64(data.Index) >= commCount {
		return errors.New("attestation data is invalid, committee index out of range")
	}

	// Check source
	if data.Target.Epoch == currentEpoch {
		currentJustified, err := state.CurrentJustifiedCheckpoint()
		if err != nil {
			return err
		}
		currJustRaw, err := currentJustified.Raw()
		if err != nil {
			return err
		}
		if data.Source != currJustRaw {
			return errors.New("attestation source does not match current justified checkpoint")
		}
	} else {
		previousJustified, err := state.PreviousJustifiedCheckpoint()
		if err != nil {
			return err
		}
		prevJustRaw, err := previousJustified.Raw()
		if err != nil {
			return err
		}
		if data.Source != prevJustRaw {
			return errors.New("attestation source does not match previous justified checkpoint")
		}
	}

	// Check signature and bitfields
	committee, err := epc.GetBeaconCommittee(data.Slot, data.Index)
	if err != nil {
		return err
	}
	if indexedAtt, err := attestation.ConvertToIndexed(committee); err != nil {
		return fmt.Errorf("attestation could not be converted to an indexed attestation: %v", err)
	} else if err := spec.ValidateIndexedAttestation(epc, state, indexedAtt); err != nil {
		return fmt.Errorf("attestation could not be verified in its indexed form: %v", err)
	}

	// TODO pending attestation to att node, append to tree
	proposerIndex, err := epc.GetBeaconProposer(currentSlot)
	if err != nil {
		return err
	}
	// Cache pending attestation
	pendingAttestationRaw := PendingAttestation{
		Data:            *data,
		AggregationBits: attestation.AggregationBits,
		InclusionDelay:  currentSlot - data.Slot,
		ProposerIndex:   proposerIndex,
	}
	pendingAttestation := pendingAttestationRaw.View()

	if data.Target.Epoch == currentEpoch {
		atts, err := state.CurrentEpochAttestations()
		if err != nil {
			return err
		}
		if err := atts.Append(pendingAttestation); err != nil {
			return err
		}
	} else {
		atts, err := state.PreviousEpochAttestations()
		if err != nil {
			return err
		}
		if err := atts.Append(pendingAttestation); err != nil {
			return err
		}
	}
	return nil
}

// Convert attestation to (almost) indexed-verifiable form
func (attestation *Attestation) ConvertToIndexed(committee []ValidatorIndex) (*IndexedAttestation, error) {
	bitLen := attestation.AggregationBits.Bits.BitLen()
	if uint64(len(committee)) != bitLen {
		return nil, fmt.Errorf("committee size does not match bits size: %d <> %d", len(committee), bitLen)
	}

	participants := make([]ValidatorIndex, 0, len(committee))
	for i := uint64(0); i < bitLen; i++ {
		if attestation.AggregationBits.Bits.GetBit(i) {
			participants = append(participants, committee[i])
		}
	}
	sort.Slice(participants, func(i int, j int) bool {
		return participants[i] < participants[j]
	})

	return &IndexedAttestation{
		AttestingIndices: CommitteeIndicesList{
			Indices: participants,
			Limit:   attestation.AggregationBits.BitLimit,
		},
		Data:             attestation.Data,
		Signature:        attestation.Signature,
	}, nil
}
