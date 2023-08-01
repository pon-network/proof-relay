package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	capellaAPI "github.com/attestantio/go-builder-client/api/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"

	beaconclient "github.com/pon-pbs/bbRelay/beaconinterface"
	"github.com/pon-pbs/bbRelay/bls"
	"github.com/pon-pbs/bbRelay/constants"
	"github.com/pon-pbs/bbRelay/signing"
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

func SanityBuilderBlock(payload builderTypes.BuilderBlockBid) error {
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

func SignedBuilderBid(builderBid builderTypes.BuilderBlockBid, sk *bls.SecretKey, publicKey phase0.BLSPubKey, domain signing.Domain) (*capellaAPI.SignedBuilderBid, error) {

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
func NewEthNetworkDetails(network string, beaconClient *beaconclient.MultiBeaconClient) (*EthNetwork, error) {

	genesisNetwork, err := beaconClient.Genesis()
	if err != nil {
		return nil, err
	}
	domainBuilder, err := signing.ComputeDomain(signing.DomainTypeAppBuilder, genesisNetwork.GenesisForkVersion, signing.Root{}.String())
	if err != nil {
		return nil, err
	}

	if network == "Ethereum" {

		domainBeaconCapella, err := signing.ComputeDomain(signing.DomainTypeBeaconProposer, constants.CapellaForkVersionMainnet, genesisNetwork.GenesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return &EthNetwork{
			Network:             0,
			GenesisTime:         genesisNetwork.GenesisTime,
			DomainBuilder:       domainBuilder,
			DomainBeaconCapella: domainBeaconCapella,
		}, nil
	}

	if network == "Goerli" {

		domainBeaconCapella, err := signing.ComputeDomain(signing.DomainTypeBeaconProposer, constants.CapellaForkVersionGoerli, genesisNetwork.GenesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return &EthNetwork{
			Network:             1,
			GenesisTime:         genesisNetwork.GenesisTime,
			DomainBuilder:       domainBuilder,
			DomainBeaconCapella: domainBeaconCapella,
		}, nil
	}

	if network == "Custom-Testnet" {
		domainBeaconCapella, err := signing.ComputeDomain(signing.DomainTypeBeaconProposer, constants.CapellaForkVersionCustomTestnet, genesisNetwork.GenesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return &EthNetwork{
			Network:             2,
			GenesisTime:         genesisNetwork.GenesisTime,
			DomainBuilder:       domainBuilder,
			DomainBeaconCapella: domainBeaconCapella,
		}, nil
	}
	return &EthNetwork{}, nil
}

func sendHTTPRequest(client http.Client, url string, msg any) (http.Response, error) {
	msgbytes, err := json.Marshal(msg)
	if err != nil {
		return http.Response{}, err
	}
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(msgbytes))
	if err != nil {
		return http.Response{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return http.Response{}, err
	}

	if resp.StatusCode != 200 {
		return http.Response{}, fmt.Errorf("invalid response code: %d", resp.StatusCode)
	}

	return *resp, nil

}
