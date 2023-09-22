package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	capellaAPI "github.com/attestantio/go-eth2-client/api/v1/capella"
	"github.com/attestantio/go-eth2-client/spec/altair"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	commonTypes "github.com/bsn-eng/pon-golang-types/common"
	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	"github.com/go-redis/redis/v9"
	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/sirupsen/logrus"
)

func (relay *Relay) handleProposerTestPayload(w http.ResponseWriter, req *http.Request) {

	payload := new(commonTypes.VersionedSignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(payload); err != nil {
		relay.log.WithError(err).Warn("Proposer payload request failed to decode")
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Proposer payload request failed to decode. %s", err.Error()))
		return
	}

	// unpack the obtained versioned signed blinded beacon block into a base signed blinded beacon block for access
	baseSignedBlindedBeaconBlock, err := payload.ToBaseSignedBlindedBeaconBlock()
	if err != nil {
		relay.log.WithError(err).Warn("could not convert versioned signed blinded beacon block to base signed blinded beacon block")
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("could not convert versioned signed blinded beacon block to base signed blinded beacon block. %s", err.Error()))
		return
	}

	slot := uint64(5861242)
	blockHash := "0x176faae852d13bef85c7a1a6eae46829f87be174727d248e4c588cd5feb089b5"
	relay.log.WithFields(logrus.Fields{
		"slot":      slot,
		"blockHash": blockHash,
	}).Info("Proposer Payload Request")

	proposerPubkey := "0x876789362d262b86dbc70f295f4e367843335eadf2c7a15268c643c7e26da01c14c8ebbec16c1fd52665ec555ffcd8b9"

	blockSubmission, err := relay.bidBoard.PayloadUtils(uint64(slot), blockHash)
	if err != nil {
		if err == redis.Nil {
			relay.log.WithError(err).Warn("No Bid Available")
			relay.RespondError(w, http.StatusBadRequest, "No Bid Available")
			return
		} else {
			relay.log.WithError(err).Error("failed getting builder API")
			relay.RespondError(w, http.StatusBadRequest, "failed getting builder API")
			return
		}
	}

	proposerBlock := &databaseTypes.ValidatorReturnedBlockDatabase{
		Signature:      baseSignedBlindedBeaconBlock.Signature.String(),
		Slot:           uint64(slot),
		BlockHash:      blockHash,
		ProposerPubkey: proposerPubkey,
	}

	err = relay.db.PutValidatorReturnedBlock(req.Context(), *proposerBlock)
	if err != nil {
		relay.log.WithError(err).Error("Error Putting In DB")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	postBody, _ := json.Marshal(payload)
	resp, err := http.Post(blockSubmission.API, "application/json", bytes.NewReader(postBody))
	if err != nil {
		relay.log.WithError(err).Error("Error Putting In DB")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		relay.log.WithError(err).Error("getPayload request failed")
		response, _ := io.ReadAll(resp.Body)
		relay.RespondError(w, http.StatusBadRequest, string(response))
		return
	}

	getPayloadResponse := new(commonTypes.VersionedExecutionPayload)
	if err := json.NewDecoder(resp.Body).Decode(&getPayloadResponse); err != nil {
		relay.log.WithError(err).Warn("getPayload request failed to decode")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	defer func() {
		errs := relay.bidBoard.PutPayloadDelivered(slot, blockSubmission.BuilderWalletAddress)
		if errs != nil {
			relay.log.WithError(errs).Error("Couldn't Set Payload Delivered Slot")
		}
	}()

	proposerBulletinBoard := bulletinBoardTypes.SlotPayloadRequest{
		Slot:     uint64(baseSignedBlindedBeaconBlock.Message.Slot),
		Proposer: proposerPubkey,
	}
	relay.bulletinBoard.Channel.SlotPayloadChannel <- proposerBulletinBoard

	relay.RespondOK(w, &getPayloadResponse)
	relay.log.WithFields(logrus.Fields{
		"Slot": slot,
	}).Info("Payload Delivered")
}

func (relay *Relay) TESThandleProposerHeader(w http.ResponseWriter, req *http.Request) {
	reqParams := mux.Vars(req)
	proposerReq, err := proposerParameters(reqParams)
	if err != nil {
		relay.log.WithError(err).Error("could not get request params")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	relay.log.WithFields(logrus.Fields{
		"slot": proposerReq.Slot,
	}).Info("Get Header Requested")

	bid, err := relay.bidBoard.WinningBid(proposerReq.Slot)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			relay.log.Warn("No Bids Available")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		relay.log.WithError(err).Error("Could't Get Winning Bid")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if (bid.Slot != proposerReq.Slot) || (bid.ProposerPubKeyHex != proposerReq.ProposerPubKeyHex) {
		relay.log.Error("Parameters Not As Per Winning Bid")
		relay.RespondError(w, http.StatusBadRequest, "Parameters Not As Per Winning Bid")
	}

	builderBidSubmission := bid.Bid.Data.Message

	bidDB := databaseTypes.ValidatorDeliveredHeaderDatabase{
		Slot:           proposerReq.Slot,
		BidValue:       0,
		BlockHash:      "",
		ProposerPubkey: proposerReq.ProposerPubKeyHex,
	}

	err = relay.db.PutValidatorDeliveredHeader(req.Context(), bidDB)
	if err != nil {
		relay.log.WithError(err).Error("could not save to Database")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	proposerBulletinBoard := bulletinBoardTypes.ProposerHeaderRequest{
		Slot:      proposerReq.Slot,
		Proposer:  proposerReq.ProposerPubKeyHex,
		Timestamp: uint64(time.Now().Unix()),
	}
	relay.bulletinBoard.Channel.ProposerHeaderChannel <- proposerBulletinBoard

	relay.log.WithFields(logrus.Fields{
		"value":     builderBidSubmission.Value.String(),
		"blockHash": "",
	}).Info("Bid Delivered To Proposer")

	go relay.testBuilderCall(bid.Slot, phase0.Hash32([32]byte{}))
	relay.RespondOK(w, &bid.Bid)
}

func (relay *Relay) testBuilderCall(slot uint64, blockHash phase0.Hash32) {
	relay.log.Info("Testing Builder Call")
	blockSubmission, err := relay.bidBoard.PayloadUtils(uint64(slot), blockHash.String())

	if err != nil {
		relay.log.WithError(err).Error("failed getting builder API")
		return
	}

	payload := capellaAPI.SignedBlindedBeaconBlock{
		Message: &capellaAPI.BlindedBeaconBlock{
			Slot:          phase0.Slot(0),
			ProposerIndex: phase0.ValidatorIndex(0),
			ParentRoot:    phase0.Root{},
			StateRoot:     phase0.Root{},
			Body: &capellaAPI.BlindedBeaconBlockBody{
				RANDAOReveal:           phase0.BLSSignature{},
				ETH1Data:               &phase0.ETH1Data{},
				SyncAggregate:          &altair.SyncAggregate{},
				ExecutionPayloadHeader: &capella.ExecutionPayloadHeader{},
				Graffiti:               [32]byte{},
				ProposerSlashings:      []*phase0.ProposerSlashing{},
				AttesterSlashings:      []*phase0.AttesterSlashing{},
				Attestations:           []*phase0.Attestation{},
				Deposits:               []*phase0.Deposit{},
				VoluntaryExits:         []*phase0.SignedVoluntaryExit{},
				BLSToExecutionChanges:  []*capella.SignedBLSToExecutionChange{},
			},
		},
	}

	payload.Message.Slot = phase0.Slot(slot)
	payload.Message.Body.ExecutionPayloadHeader.BlockHash = blockHash
	payload.Message.Body.ETH1Data.BlockHash = blockHash[:]
	payload.Message.Body.SyncAggregate.SyncCommitteeBits = bitfield.NewBitvector512()
	payload.Signature = phase0.BLSSignature{}

	resp, err := sendHTTPRequest(*relay.client, blockSubmission.API, payload)
	if err != nil {
		relay.log.WithError(err).Error("Error Getting Builder")
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		relay.log.WithError(err).Error("getPayload request failed")
		return
	}

	getPayloadResponse := new(capella.ExecutionPayload)
	if err := json.NewDecoder(resp.Body).Decode(&getPayloadResponse); err != nil {
		relay.log.WithError(err).Warn("getPayload request failed to decode")
		return
	}
}
