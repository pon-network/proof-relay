package client

import (
	"context"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	commonTypes "github.com/bsn-eng/pon-golang-types/common"
	"github.com/ethereum/go-ethereum/common"

	beaconData "github.com/pon-pbs/bbRelay/beaconinterface/data"
)

type BeaconClientInstance interface {
	BaseEndpoint() string

	// get methods
	GetSlotProposerMap(uint64) (beaconData.SlotProposerMap, error)
	SyncStatus() (*beaconTypes.SyncStatusData, error)
	GetValidatorList(uint64) ([]*beaconTypes.ValidatorData, error)
	GetValidatorIndex([]string) (*beaconTypes.GetValidatorsResponse, error)
	Genesis() (*beaconTypes.GenesisData, error)
	GetWithdrawals(uint64) (*beaconTypes.Withdrawals, error)
	Randao(uint64) (*common.Hash, error)
	GetBlockHeader(slot uint64) (*beaconTypes.BlockHeaderData, error)
	GetCurrentBlockHeader() (*beaconTypes.BlockHeaderData, error)
	GetForkVersion(slot uint64, head bool) (forkName string, forkVersion string, err error)

	// post methods
	PublishBlock(context.Context, commonTypes.VersionedSignedBeaconBlock) error

	// subscription methods
	SubscribeToHeadEvents(context.Context, chan beaconTypes.HeadEventData)
	SubscribeToPayloadAttributesEvents(context.Context, chan beaconTypes.PayloadAttributesEvent)
}
