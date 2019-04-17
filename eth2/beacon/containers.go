package beacon

import (
	"github.com/protolambda/zrnt/eth2/util/bitfield"
	"github.com/protolambda/zrnt/eth2/util/bls"
	. "github.com/protolambda/zrnt/eth2/util/data_types"
	"github.com/protolambda/zrnt/eth2/util/hex"
	"github.com/protolambda/zrnt/eth2/util/ssz"
)

// NOTE: these containers are going to be moved to sub-packages, per-topic.

type ProposerSlashing struct {
	// Proposer index
	ProposerIndex ValidatorIndex
	// First proposal
	Header1 BeaconBlockHeader
	// Second proposal
	Header2 BeaconBlockHeader
}

type AttesterSlashing struct {
	// First attestation
	Attestation1 IndexedAttestation
	// Second attestation
	Attestation2 IndexedAttestation
}

type Attestation struct {
	// Attester aggregation bitfield
	AggregationBitfield bitfield.Bitfield
	// Attestation data
	Data AttestationData
	// Custody bitfield
	CustodyBitfield bitfield.Bitfield
	// BLS aggregate signature
	AggregateSignature bls.BLSSignature `ssz:"signature"`
}

type AttestationData struct {
	//  LMD GHOST vote
	Slot Slot
	// Root of the signed beacon block
	BeaconBlockRoot Root

	// FFG vote
	SourceEpoch Epoch
	SourceRoot  Root
	TargetRoot  Root

	// Crosslink vote
	Shard             Shard
	PreviousCrosslink Crosslink
	CrosslinkDataRoot Root
}

type AttestationDataAndCustodyBit struct {
	// Attestation data
	Data AttestationData
	// Custody bit
	CustodyBit bool
}

type IndexedAttestation struct {
	// Validator Indices
	CustodyBit0Indexes []ValidatorIndex
	CustodyBit1Indexes []ValidatorIndex
	// Attestation data
	Data AttestationData
	// BLS aggregate signature
	AggregateSignature bls.BLSSignature `ssz:"signature"`
}

type Crosslink struct {
	// Epoch number
	Epoch Epoch
	// Shard data since the previous crosslink
	CrosslinkDataRoot Root
}

type Deposit struct {
	// Branch in the deposit tree
	Proof [DEPOSIT_CONTRACT_TREE_DEPTH]Root
	// Index in the deposit tree
	Index DepositIndex
	// Data
	Data DepositData
}

type DepositData struct {
	// BLS pubkey
	Pubkey bls.BLSPubkey
	// Withdrawal credentials
	WithdrawalCredentials Root
	// Amount in Gwei
	Amount Gwei
	// Container self-signature
	ProofOfPossession bls.BLSSignature `ssz:"signature"`
}

// Let serialized_deposit_data be the serialized form of deposit.deposit_data.
//
// It should equal to:
//  48 bytes for pubkey
//  32 bytes for withdrawal credentials
//  8 bytes for amount
//  96 bytes for proof of possession
//
// This should match deposit_data in the Ethereum 1.0 deposit contract
//  of which the hash was placed into the Merkle tree.
func (d *DepositData) Serialized() []byte {
	return ssz.SSZEncode(d)
}

type VoluntaryExit struct {
	// Minimum epoch for processing exit
	Epoch Epoch
	// Index of the exiting validator
	ValidatorIndex ValidatorIndex
	// Validator signature
	Signature bls.BLSSignature `ssz:"signature"`
}

type Transfer struct {
	// Sender index
	Sender ValidatorIndex
	// Recipient index
	Recipient ValidatorIndex
	// Amount in Gwei
	Amount Gwei
	// Fee in Gwei for block proposer
	Fee Gwei
	// Inclusion slot
	Slot Slot
	// Sender withdrawal pubkey
	Pubkey bls.BLSPubkey
	// Sender signature
	Signature bls.BLSSignature `ssz:"signature"`
}

type Validator struct {
	// BLS public key
	Pubkey bls.BLSPubkey
	// Withdrawal credentials
	WithdrawalCredentials Root
	// Epoch when became eligible for activation
	ActivationEligibilityEpoch Epoch
	// Epoch when validator activated
	ActivationEpoch Epoch
	// Epoch when validator exited
	ExitEpoch Epoch
	// Epoch when validator is eligible to withdraw
	WithdrawableEpoch Epoch
	// Was the validator slashed
	Slashed bool
	// Rounded balance
	HighBalance Gwei
}

func (v *Validator) Copy() *Validator {
	copyV := *v
	return &copyV
}

func (v *Validator) IsActive(epoch Epoch) bool {
	return v.ActivationEpoch <= epoch && epoch < v.ExitEpoch
}

func (v *Validator) IsSlashable(epoch Epoch) bool {
	return v.ActivationEpoch <= epoch && epoch < v.WithdrawableEpoch && !v.Slashed
}

type PendingAttestation struct {
	// Attester aggregation bitfield
	AggregationBitfield bitfield.Bitfield
	// Attestation data
	Data AttestationData
	// Custody bitfield
	CustodyBitfield bitfield.Bitfield
	// Inclusion slot
	InclusionSlot Slot
}

// 32 bits, not strictly an integer, hence represented as 4 bytes
// (bytes not necessarily corresponding to versions)
type ForkVersion [4]byte

func (v *ForkVersion) MarshalJSON() ([]byte, error)    { return hex.EncodeHex((*v)[:]) }
func (v *ForkVersion) UnmarshalJSON(data []byte) error { return hex.DecodeHex(data[1:len(data)-1], (*v)[:]) }
func (v *ForkVersion) String() string                  { return hex.EncodeHexStr((*v)[:]) }

func Int32ToForkVersion(v uint32) ForkVersion {
	return [4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

type Fork struct {
	// Previous fork version
	PreviousVersion ForkVersion
	// Current fork version
	CurrentVersion ForkVersion
	// Fork epoch number
	Epoch Epoch
}

// Return the fork version of the given epoch
func (f Fork) GetVersion(epoch Epoch) ForkVersion {
	if epoch < f.Epoch {
		return f.PreviousVersion
	}
	return f.CurrentVersion
}

type Eth1Data struct {
	// Root of the deposit tree
	DepositRoot Root
	// Total number of deposits
	DepositCount DepositIndex
	// Block hash
	BlockHash Root
}

type Eth1DataVote struct {
	// Data being voted for
	Eth1Data Eth1Data
	// Vote count
	VoteCount uint64
}

type ValidatorRegistry []*Validator

func (vr ValidatorRegistry) IsValidatorIndex(index ValidatorIndex) bool {
	return index < ValidatorIndex(len(vr))
}

func (vr ValidatorRegistry) GetActiveValidatorIndices(epoch Epoch) []ValidatorIndex {
	res := make([]ValidatorIndex, 0, len(vr))
	for i, v := range vr {
		if v.IsActive(epoch) {
			res = append(res, ValidatorIndex(i))
		}
	}
	return res
}

func (vr ValidatorRegistry) GetActiveValidatorCount(epoch Epoch) (count uint64) {
	for _, v := range vr {
		if v.IsActive(epoch) {
			count++
		}
	}
	return
}

type HistoricalBatch struct {
	BlockRoots [SLOTS_PER_HISTORICAL_ROOT]Root
	StateRoots [SLOTS_PER_HISTORICAL_ROOT]Root
}
