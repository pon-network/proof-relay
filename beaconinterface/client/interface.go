package client

import (
	"context"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	beaconData "github.com/bsn-eng/pon-wtfpl-relay/beaconinterface/data"
	"github.com/ethereum/go-ethereum/common"
)

type BeaconClientInstance interface {
	BaseEndpoint() string

	// get methods
	GetSlotProposerMap(uint64) (beaconData.SlotProposerMap, error)
	SyncStatus() (*beaconTypes.SyncStatusData, error)
	GetValidatorList(uint64) ([]*beaconTypes.ValidatorData, error)
	Genesis() (*beaconTypes.GenesisData, error)
	GetWithdrawals(uint64) (*beaconTypes.Withdrawals, error)
	Randao(uint64) (*common.Hash, error)
	GetBlock(slot uint64) (*beaconTypes.SignedBeaconBlock, error)
	GetBlockHeader(slot uint64) (*beaconTypes.BlockHeaderData, error)
	GetCurrentBlockHeader() (*beaconTypes.BlockHeaderData, error)

	// post methods
	PublishBlock(context.Context, interface{}) error

	// subscription methods
	SubscribeToHeadEvents(context.Context, chan beaconTypes.HeadEventData)
	SubscribeToPayloadAttributesEvents(context.Context, chan beaconTypes.PayloadAttributesEventData)
}
