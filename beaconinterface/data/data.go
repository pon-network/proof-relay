package beaconinterface

import (
	"sync"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	"github.com/ethereum/go-ethereum/common"
)

type BeaconData struct {
	CurrentSlot  uint64
	CurrentEpoch uint64
	CurrentForkVersion string

	CurrentHead beaconTypes.HeadEventData

	Mu                       sync.Mutex
	RandaoMap                RandaoMap
	SlotProposerMap          SlotProposerMap
	SlotPayloadAttributesMap SlotPayloadAttributesMap
	AllValidatorsByPubkey    AllValidatorsByPubkeyMap
	AllValidatorsByIndex     AllValidatorsByIndexMap

	HeadSlotC          chan beaconTypes.HeadEventData
	PayloadAttributesC chan beaconTypes.PayloadAttributesEvent
}

type RandaoMap map[uint64]common.Hash
type SlotProposerMap map[uint64]beaconTypes.ProposerDutyData
type SlotPayloadAttributesMap map[uint64]beaconTypes.PayloadAttributesEventData
type AllValidatorsByPubkeyMap map[string]beaconTypes.ValidatorData
type AllValidatorsByIndexMap map[uint64]beaconTypes.ValidatorData
