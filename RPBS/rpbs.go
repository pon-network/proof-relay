package rpbs

import (
	"fmt"

	relayTypes "github.com/bsn-eng/pon-golang-types/relay"
)

func Verify(builderBid relayTypes.BuilderSubmitBlockRequest) (bool, error) {
	info := fmt.Sprintf("BuilderWalletAddress: %s, Slot: %d, Amount: %d, Transaction: %s", builderBid.Message.BuilderPubkey.String(), builderBid.Message.Slot, builderBid.Message.Value, builderBid.Message.PayoutPoolTransaction.String())
	bid, err := VerifySignatureWithStringInput(builderBid.Message.RPBSPubkey, info, builderBid.Message.RPBS)
	if err != nil {
		return false, err
	}
	return bid, nil
}
