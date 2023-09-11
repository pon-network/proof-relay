package rpbs

import (
	"fmt"
	"strings"

	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func Verify(builderBid builderTypes.BuilderBlockBid) (bool, error) {
	info := strings.ToLower(fmt.Sprintf("BuilderWalletAddress:%s,Slot:%d,Amount:%d,Transaction:%s", builderBid.Message.BuilderWalletAddress.String(), builderBid.Message.Slot, builderBid.Message.Value, hexutil.Encode(builderBid.Message.PayoutPoolTransaction)))
	bid, err := VerifySignatureWithStringInput(builderBid.Message.RPBSPubkey, info, builderBid.Message.RPBS)
	if err != nil {
		return false, err
	}
	return bid, nil
}
