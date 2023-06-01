package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	capellaAPI "github.com/attestantio/go-eth2-client/api/v1/capella"
	"github.com/attestantio/go-eth2-client/spec/capella"
	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	relayTypes "github.com/bsn-eng/pon-golang-types/relay"
	beaconclient "github.com/bsn-eng/pon-wtfpl-relay/beaconinterface"
	"github.com/bsn-eng/pon-wtfpl-relay/bids"
	"github.com/bsn-eng/pon-wtfpl-relay/bls"
	"github.com/bsn-eng/pon-wtfpl-relay/bulletinboard"
	"github.com/bsn-eng/pon-wtfpl-relay/database"
	ponpool "github.com/bsn-eng/pon-wtfpl-relay/ponPool"
	"github.com/bsn-eng/pon-wtfpl-relay/redisPackage"
	reporterServer "github.com/bsn-eng/pon-wtfpl-relay/reporter"
	"github.com/bsn-eng/pon-wtfpl-relay/rpbs"
	"github.com/bsn-eng/pon-wtfpl-relay/signing"
	relayUtils "github.com/bsn-eng/pon-wtfpl-relay/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-redis/redis/v9"
	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

func NewRelayAPI(params *RelayParams, log logrus.Entry) (relay *Relay, err error) {
	dataBase, err := database.NewDatabase(params.DbURL, params.DatabaseParams, params.DbDriver)
	if err != nil {
		log.WithError(err).Fatal("Failed Database")
		return nil, err
	}

	ponPool := ponpool.NewPonPool(params.PonPoolURL, params.PonPoolAPIKey)

	bulletinBoard, err := bulletinboard.NewMQTTClient(params.BulletinBoardParams)
	if err != nil {
		log.WithError(err).Fatal("Failed Bulletin Board")
		return nil, err
	}

	beaconClient, err := beaconclient.NewMultiBeaconClient(params.BeaconClientUrls)
	if err != nil {
		log.WithError(err).Fatal("Failed Beacon Client")
		return nil, err
	}

	reporter := reporterServer.NewReporterServer(params.ReporterURL, dataBase)
	reporter.StartServer()

	redisInterface, err := redisPackage.NewRedisInterface(params.RedisURI)
	if err != nil {
		log.WithError(err).Fatal("Failed Redis Interface")
		return nil, err
	}

	relayutils := relayUtils.NewRelayUtils(dataBase, beaconClient, *ponPool, *redisInterface)
	relayutils.StartUtils()

	bidInterface := bids.NewBidBoard(*redisInterface, *bulletinBoard, params.BidTimeOut)

	publickey, err := bls.RelayBLSPubKey(*bls.PublicKeyFromSecretKey(params.Sk))
	if err != nil {
		log.WithError(err).Fatal("Failed Relay Public Key")
		return nil, err
	}

	relayAPI := &Relay{
		db:             dataBase,
		ponPool:        ponPool,
		bulletinBoard:  bulletinBoard,
		beaconClient:   beaconClient,
		reporterServer: reporter,
		bidBoard:       bidInterface,
		relayutils:     relayutils,

		client:    &http.Client{},
		blsSk:     params.Sk,
		publicKey: publickey,
		log:       &log,
	}

	if params.NewRelicApp != "" {
		app, err := newrelic.NewApplication(
			newrelic.ConfigAppName(params.NewRelicApp),
			newrelic.ConfigLicense(params.NewRelicLicense),
			newrelic.ConfigAppLogForwardingEnabled(params.NewRelicForwarding),
		)
		if err != nil {
			log.WithError(err).Error("New Relic Couldn't Setup")
		}
		relayAPI.newRelicApp = app
		relayAPI.newRelicEnabled = true
	} else {
		log.Warn("New Relic Closed")
		relayAPI.newRelicEnabled = false
	}

	return relayAPI, nil
}

func (relay *Relay) Routes() http.Handler {
	r := mux.NewRouter()

	if relay.newRelicEnabled {

		r.HandleFunc(newrelic.WrapHandleFunc(relay.newRelicApp, "/relay", relay.handleLanding)).Methods(http.MethodGet)
		r.HandleFunc(newrelic.WrapHandleFunc(relay.newRelicApp, "/eth/v1/builder/status", relay.handleStatus)).Methods(http.MethodGet)
		r.HandleFunc(newrelic.WrapHandleFunc(relay.newRelicApp, "/eth/v1/builder/validators", relay.handleRegisterValidator)).Methods(http.MethodPost)

		r.HandleFunc(newrelic.WrapHandleFunc(relay.newRelicApp, "/relay/builder/submitBlock", relay.handleSubmitBlock)).Methods(http.MethodPost)

		r.HandleFunc(newrelic.WrapHandleFunc(relay.newRelicApp, "/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}", relay.handleProposerHeader)).Methods(http.MethodGet)
		r.HandleFunc(newrelic.WrapHandleFunc(relay.newRelicApp, "/eth/v1/builder/blinded_blocks", relay.handleProposerPayload)).Methods(http.MethodPost)

	} else {

		r.HandleFunc("/relay", relay.handleLanding).Methods(http.MethodGet)
		r.HandleFunc("/eth/v1/builder/status", relay.handleStatus).Methods(http.MethodGet)
		r.HandleFunc("/eth/v1/builder/validators", relay.handleRegisterValidator).Methods(http.MethodPost)

		r.HandleFunc("/relay/builder/submitBlock", relay.handleSubmitBlock).Methods(http.MethodPost)

		r.HandleFunc("/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}", relay.handleProposerHeader).Methods(http.MethodGet)
		r.HandleFunc("/eth/v1/builder/blinded_blocks", relay.handleProposerPayload).Methods(http.MethodPost)

	}

	return loggingMiddleware(r, *relay.log)
}

func (relay *Relay) StartServer(ServerParams *RelayServerParams) (err error) {
	relay.log.Info("Relay Server")
	relay.server = &http.Server{
		Addr:              relay.URL,
		Handler:           relay.Routes(),
		ReadTimeout:       ServerParams.ReadTimeout,
		ReadHeaderTimeout: ServerParams.ReadHeaderTimeout,
		WriteTimeout:      ServerParams.WriteTimeout,
		IdleTimeout:       ServerParams.IdleTimeout,
	}

	err = relay.server.ListenAndServe()
	return err
}

func (relay *Relay) RespondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := HTTPError{Code: code, Message: message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		relay.log.WithField("response", resp).WithError(err).Error("Couldn't write error response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (relay *Relay) RespondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		relay.log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (relay *Relay) BlockSlotTimestamp(slot uint64) uint64 {
	return relay.network.GenesisTime + (slot * 12)
}

func (relay *Relay) handleLanding(w http.ResponseWriter, req *http.Request) {
	relay.RespondOK(w, "PON Relay")
}

func (relay *Relay) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (relay *Relay) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (relay *Relay) handleSubmitBlock(w http.ResponseWriter, req *http.Request) {

	blockTimestamp := time.Now()
	builderBlock := new(relayTypes.BuilderSubmitBlockRequest)

	if err := json.NewDecoder(req.Body).Decode(&builderBlock); err != nil {
		relay.log.WithError(err).Warn("Could Not Convert Patload To Builder Submisioon")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	status, err := relay.relayutils.BuilderStatus(builderBlock.Message.BuilderWalletAddress.String())
	if err != nil {
		relay.log.WithError(err).Warn("Couldn' Get Builder Status")
		relay.RespondError(w, http.StatusBadRequest, "Failed To Get Builder")
		return
	}
	if !status {
		relay.log.Warn("Builder Not Active In PON")
		relay.RespondError(w, http.StatusBadRequest, "Builder Not Active In PON")
		return
	}

	if builderBlock.Message.ExecutionPayloadHeader.Timestamp != relay.BlockSlotTimestamp(builderBlock.Message.Slot) {
		relay.log.Warnf("incorrect timestamp. got %d, expected %d", builderBlock.Message.ExecutionPayloadHeader.Timestamp)
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("incorrect timestamp. got %d", builderBlock.Message.ExecutionPayloadHeader.Timestamp))
		return
	}

	deliveredPayloadBuilder, err := relay.bidBoard.GetPayloadDelivered(builderBlock.Message.Slot)
	if err != nil && !errors.Is(err, redis.Nil) {
		relay.log.WithError(err).Error("failed to get delivered payload slot from redis")
		relay.RespondError(w, http.StatusBadRequest, "failed to get delivered payload slot from redis")
		return
	} else if err != nil && errors.Is(err, redis.Nil) {
		relay.log.Info("No Payload Sent For Slot, Bid Submitted")
	} else {
		relay.log.WithError(err).Error("Payload Delivered For Slot")
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Payload For Slot %d Delivered For Builder %s", builderBlock.Message.Slot, deliveredPayloadBuilder))
		return
	}

	builderRPBS, err := rpbs.Verify(*builderBlock)
	if err != nil {
		relay.log.WithError(err).Error("RPBS Verify Error")
		relay.RespondError(w, http.StatusBadRequest, "RPBS Verify Error")
		return
	}
	if !builderRPBS {
		relay.log.Error("RPBS Verify Failed")
		relay.RespondError(w, http.StatusBadRequest, "RPBS Verify Failed")
		return
	}

	relay.beaconClient.BeaconData.Mu.Lock()
	relaySlot := relay.beaconClient.BeaconData.CurrentSlot
	relay.beaconClient.BeaconData.Mu.Unlock()

	if builderBlock.Message.Slot <= relaySlot {
		relay.log.Info("submitBlock failed: submission for past slot")
		relay.RespondError(w, http.StatusBadRequest, "submission for past slot")
		return
	}

	err = SanityBuilderBlock(*builderBlock)
	if err != nil {
		relay.log.WithError(err).Info("block submission sanity checks failed")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	pubkey, err := crypto.Ecrecover(crypto.Keccak256Hash(builderBlock.EcdsaSignature[:]).Bytes(), builderBlock.EcdsaSignature[:])
	if err != nil {
		relay.log.Error("Could not recover ECDSA pubkey", "err", err)
		relay.RespondError(w, http.StatusServiceUnavailable, "Could not recover ECDSA pubkey")
		return
	}
	ecdsaPubkey, err := crypto.UnmarshalPubkey(pubkey)
	if err != nil {
		relay.log.Error("Could not recover ECDSA pubkey", "err", err)
		relay.RespondError(w, http.StatusServiceUnavailable, "Could not recover ECDSA pubkey")
		return
	}

	pubkeyAddress := crypto.PubkeyToAddress(*ecdsaPubkey)
	if strings.ToLower(pubkeyAddress.String()) != strings.ToLower(builderBlock.Message.BuilderWalletAddress.String()) {
		relay.log.Error("ECDSA pubkey does not match wallet address", "err", err, "pubkeyAddress", pubkeyAddress.String(), "walletAddress", builderBlock.Message.BuilderWalletAddress.String())
		relay.RespondError(w, http.StatusServiceUnavailable, "ECDSA pubkey does not match wallet address")
		return
	}

	builderDbBlock := databaseTypes.BuilderBlockDatabase{
		Slot:             builderBlock.Message.Slot,
		BuilderPubkey:    builderBlock.Message.BuilderWalletAddress.String(),
		BuilderSignature: builderBlock.EcdsaSignature.String(),
		TransactionByte:  builderBlock.Message.PayoutPoolTransaction.String(),
		Value:            builderBlock.Message.Value,
	}

	defer func() {
		err := relay.db.PutBuilderBlockSubmission(req.Context(), builderDbBlock)
		if err != nil {
			relay.log.WithError(err).WithField("payload", builderDbBlock).Error("saving builder block submission to database failed")
			return
		}
	}()

	lastBid, err := relay.bidBoard.BuilderBlockLast(builderBlock.Message.Slot, builderBlock.Message.BuilderWalletAddress.String())
	if err != nil {
		relay.log.WithError(err).Error("failed getting latest payload receivedAt from redis")
	} else if blockTimestamp.Unix() < lastBid {
		relay.log.Info("Bid After Given Bid")
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Using newer bid for Builder %s", builderBlock.Message.BuilderWalletAddress.String()))
		return
	}

	signedBuilderBid, err := SignedBuilderBid(*builderBlock, relay.blsSk, relay.publicKey, relay.network.DomainBuilder)
	if err != nil {
		relay.log.WithError(err).Error("could not sign builder bid")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	getHeaderResponse := relayUtils.GetHeaderResponse{
		Version: VersionCapella,
		Data:    signedBuilderBid,
	}

	getPayloadHeaderResponse := relayUtils.GetPayloadUtils{
		Version:              VersionCapella,
		Data:                 builderBlock.Message.ExecutionPayloadHeader,
		API:                  builderBlock.Message.Endpoint,
		BuilderWalletAddress: builderBlock.Message.BuilderWalletAddress.String(),
	}

	err = relay.bidBoard.SavePayloadUtils(builderBlock.Message.Slot, builderBlock.Message.BuilderWalletAddress.String(), builderBlock.Message.BlockHash.String(), &getPayloadHeaderResponse)
	if err != nil {
		relay.log.WithError(err).Error("failed saving execution payload in redis")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = relay.bidBoard.SaveBuilderBid(
		builderBlock.Message.Slot,
		builderBlock.Message.BuilderWalletAddress.String(),
		builderBlock.Message.ProposerPubkey.String(),
		blockTimestamp,
		&getHeaderResponse,
	)
	if err != nil {
		relay.log.WithError(err).Error("could not save latest builder bid")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	highestBidBuilder, err := relay.bidBoard.AuctionBid(builderBlock.Message.Slot)
	if err != nil {
		relay.log.WithError(err).Error("could not compute top bid")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	relay.log.WithFields(logrus.Fields{
		"Builder": builderBlock.Message.BuilderWalletAddress.String(),
		"Value":   builderBlock.Message.Value,
	}).Info("received block from builder")

	relay.RespondOK(w, &highestBidBuilder)

}

func (relay *Relay) handleProposerHeader(w http.ResponseWriter, req *http.Request) {
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

	bidDB := databaseTypes.ValidatorDeliveredHeaderDatabase{
		Slot:           proposerReq.Slot,
		Value:          *bid.Bid.Data.Message.Value.ToBig(),
		BlockHash:      bid.Bid.Data.Message.Header.BlockHash.String(),
		ProposerPubkey: proposerReq.ProposerPubKeyHex,
	}

	err = relay.db.PutValidatorDeliveredHeader(req.Context(), bidDB)
	if err != nil {
		relay.log.WithError(err).Error("could not save to Database")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	relay.log.WithFields(logrus.Fields{
		"value":     bid.Bid.Data.Message.Value.String(),
		"blockHash": bid.Bid.Data.Message.Header.BlockHash.String(),
	}).Info("Bid Delivered To Proposer")

	relay.RespondOK(w, &bid.Bid)
}

func (relay *Relay) handleProposerPayload(w http.ResponseWriter, req *http.Request) {

	payload := new(capellaAPI.SignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(payload); err != nil {
		relay.log.WithError(err).Warn("Payload request failed to decode")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	slot := payload.Message.Slot
	blockHash := payload.Message.Body.ExecutionPayloadHeader.BlockHash.String()
	relay.log.WithFields(logrus.Fields{
		"slot":      slot,
		"blockHash": blockHash,
	}).Info("Proposer Payload Request")

	proposerPubkey, err := relay.relayutils.ValidatorIndexToPubkey(uint64(payload.Message.ProposerIndex))
	if err != nil {
		relay.log.WithError(err).Error("Error Proposer Public Key")
		relay.RespondError(w, http.StatusBadRequest, "Could Not Get Proposer Public Key")
		return
	}

	if len(proposerPubkey) == 0 {
		relay.log.WithError(err).Error("Could Not Get Proposer Public Key")
		relay.RespondError(w, http.StatusBadRequest, "Could Not Get Proposer Public Key")
		return
	}

	ok, err := signing.VerifySignature(payload.Message, relay.network.DomainBeaconCapella, proposerPubkey[:], payload.Signature[:])
	if !ok || err != nil {
		relay.log.WithError(err).Warn("could not verify payload signature")
		relay.RespondError(w, http.StatusBadRequest, "could not verify payload signature")
		return
	}

	blockSubmission, err := relay.bidBoard.PayloadUtils(uint64(slot), proposerPubkey.String(), blockHash)

	if err != nil {
		relay.log.WithError(err).Warn("failed getting builder API")
		relay.RespondError(w, http.StatusBadRequest, "failed getting builder API")
	}

	proposerBlock := &databaseTypes.ValidatorReturnedBlockDatabase{
		Signature:      payload.Signature.String(),
		Slot:           uint64(slot),
		BlockHash:      blockHash,
		ProposerPubkey: proposerPubkey.String(),
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

	getPayloadResponse := new(capella.ExecutionPayload)
	if err := json.NewDecoder(resp.Body).Decode(&getPayloadResponse); err != nil {
		relay.log.WithError(err).Warn("getPayload request failed to decode")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	PayloadResponse := ProposerPayload{Bellatrix: nil, Capella: getPayloadResponse}

	defer func() {
		payloadJSON, err := getPayloadResponse.MarshalJSON()
		if err != nil {
			relay.log.WithError(err).Warn("Failed To Store Payload Delivered")
		}
		payloadDBDelivered := &databaseTypes.ValidatorDeliveredPayloadDatabase{
			Slot:           uint64(slot),
			ProposerPubkey: proposerPubkey.String(),
			BlockHash:      PayloadResponse.Capella.BlockHash.String(),
			Payload:        payloadJSON,
		}
		err = relay.db.PutValidatorDeliveredPayload(req.Context(), *payloadDBDelivered)
		if err != nil {
			fmt.Println(err)
		}
	}()

	defer func() {
		errs := relay.bidBoard.PutPayloadDelivered(slot, blockSubmission.BuilderWalletAddress)
		if errs != nil {
			relay.log.WithError(errs).Error("Couldn't Set Payload Delivered Slot")
		}
	}()

	relay.RespondOK(w, &PayloadResponse)
	relay.log.WithFields(logrus.Fields{
		"Slot": slot,
	}).Info("Payload Delivered")
}
