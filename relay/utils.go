package relay

import (
	"errors"
	"net/http"
	"strconv"

	capellaAPI "github.com/attestantio/go-builder-client/api/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	relayTypes "github.com/bsn-eng/pon-golang-types/relay"
	"github.com/bsn-eng/pon-wtfpl-relay/bls"
	"github.com/bsn-eng/pon-wtfpl-relay/signing"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	lenParentHash     = 66
	lenProposerPubKey = 98
)

func loggingMiddleware(next http.Handler, logger logrus.Entry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func SanityBuilderBlock(payload relayTypes.BuilderSubmitBlockRequest) error {
	if payload.Message.BlockHash.String() != payload.Message.ExecutionPayloadHeader.BlockHash.String() {
		return errors.New("Block Hash Wrong")
	}

	if payload.Message.ParentHash.String() != payload.Message.ExecutionPayloadHeader.ParentHash.String() {
		return errors.New("Block Hash Wrong")
	}

	return nil
}

func proposerParameters(payload map[string]string) (ProposerReqParams, error) {

	slotStr := payload["slot"]
	parentHash := payload["parent_hash"]
	proposerPubkey := payload["pubkey"]

	slot, err := strconv.ParseInt(slotStr, 10, 64)
	if err != nil {
		return ProposerReqParams{}, err
	}

	if len(proposerPubkey) != lenProposerPubKey {
		return ProposerReqParams{}, errors.New("Proposer Pubkey Wrong Length")
	}

	if len(parentHash) != lenParentHash {
		return ProposerReqParams{}, errors.New("Parent Hash Wrong Length")
	}
	return ProposerReqParams{Slot: uint64(slot), ProposerPubKeyHex: proposerPubkey, ParentHashHex: parentHash}, nil
}

func SignedBuilderBid(builderBid relayTypes.BuilderSubmitBlockRequest, sk *bls.SecretKey, publicKey phase0.BLSPubKey, domain signing.Domain) (*capellaAPI.SignedBuilderBid, error) {

	header := builderBid.Message.ExecutionPayloadHeader

	builderBidSubmission := capellaAPI.BuilderBid{
		Value:  uint256.NewInt(builderBid.Message.Value),
		Header: header,
		Pubkey: publicKey,
	}

	sig, err := signing.SignMessage(&builderBidSubmission, domain, sk)
	if err != nil {
		return nil, err
	}

	return &capellaAPI.SignedBuilderBid{
		Message:   &builderBidSubmission,
		Signature: sig,
	}, nil
}
