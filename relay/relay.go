package relay

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"encoding/asn1"
	"math/big"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	mevBoostAPI "github.com/attestantio/go-builder-client/api"
	capellaAPI "github.com/attestantio/go-eth2-client/api/v1/capella"
	"github.com/attestantio/go-eth2-client/spec"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	bulletinBoardTypes "github.com/bsn-eng/pon-golang-types/bulletinBoard"
	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-redis/redis/v9"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	beaconclient "github.com/pon-pbs/bbRelay/beaconinterface"
	"github.com/pon-pbs/bbRelay/bids"
	"github.com/pon-pbs/bbRelay/bls"
	"github.com/pon-pbs/bbRelay/bulletinboard"
	"github.com/pon-pbs/bbRelay/database"
	ponpool "github.com/pon-pbs/bbRelay/ponPool"
	"github.com/pon-pbs/bbRelay/redisPackage"
	reporterServer "github.com/pon-pbs/bbRelay/reporter"
	"github.com/pon-pbs/bbRelay/rpbs"
	"github.com/pon-pbs/bbRelay/signing"
	relayUtils "github.com/pon-pbs/bbRelay/utils"
)

func NewRelayAPI(params *RelayParams, log logrus.Entry) (relay *Relay, err error) {
	dataBase, err := database.NewDatabase(params.DbURL, params.DatabaseParams, params.DbDriver)
	if err != nil {
		log.WithError(err).Fatal("Failed Database")
		return nil, err
	}

	ponPool := ponpool.NewPonPool(params.PonPoolURL, params.PonPoolAPIKey)

	beaconClient, err := beaconclient.NewMultiBeaconClient(params.BeaconClientUrls)
	if err != nil {
		log.WithError(err).Fatal("Failed Beacon Client")
		return nil, err
	}
	beaconClient.Start()

	bulletinBoard, err := bulletinboard.NewMQTTClient(params.BulletinBoardParams, beaconClient)
	if err != nil {
		log.WithError(err).Fatal("Failed Bulletin Board")
		return nil, err
	}

	reporter := reporterServer.NewReporterServer(params.ReporterURL, dataBase)
	go reporter.StartServer()

	redisInterface, err := redisPackage.NewRedisInterface(params.RedisURI)
	if err != nil {
		log.WithError(err).Fatal("Failed Redis Interface")
		return nil, err
	}

	relayutils := relayUtils.NewRelayUtils(dataBase, beaconClient, *ponPool, *redisInterface)
	go relayutils.StartUtils()

	bidInterface := bids.NewBidBoard(*redisInterface, *bulletinBoard, params.BidTimeOut)

	publickey, err := bls.RelayBLSPubKey(*bls.PublicKeyFromSecretKey(params.Sk))
	if err != nil {
		log.WithError(err).Fatal("Failed Relay Public Key")
		return nil, err
	}

	networkInterface, err := NewEthNetworkDetails(params.Network, beaconClient)
	if err != nil {
		log.WithError(err).Fatal("Error Network")
	}

	fmt.Println(hexutil.Encode(publickey[:]))

	relayAPI := &Relay{
		db:             dataBase,
		ponPool:        ponPool,
		bulletinBoard:  bulletinBoard,
		beaconClient:   beaconClient,
		reporterServer: reporter,
		bidBoard:       bidInterface,
		relayutils:     relayutils,
		URL:            params.URL,
		network:        *networkInterface,

		client:    &http.Client{Timeout: time.Second},
		blsSk:     params.Sk,
		publicKey: publickey,
		log:       &log,
	}

	return relayAPI, nil
}

func (relay *Relay) Routes() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/relay", relay.handleLanding).Methods(http.MethodGet)
	r.HandleFunc("/eth/v1/builder/status", relay.handleStatus).Methods(http.MethodGet)
	r.HandleFunc("/eth/v1/builder/validators", relay.handleRegisterValidator).Methods(http.MethodPost)

	r.HandleFunc("/relay/v1/builder/blocks", relay.handleSubmitBlock).Methods(http.MethodPost)
	// r.HandleFunc("/relay/v1/builder/bounty_bids", relay.handleBountyBids).Methods(http.MethodPost)

	r.HandleFunc("/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}", relay.handleProposerHeader).Methods(http.MethodGet)
	r.HandleFunc("/eth/v1/builder/blinded_blocks", relay.handleProposerPayload).Methods(http.MethodPost)
	r.HandleFunc("/eth/v1/builder/test_blocks", relay.handleProposerTestPayload).Methods(http.MethodPost)
	r.HandleFunc("/eth/v1/builder/header/test/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}", relay.TESThandleProposerHeader).Methods(http.MethodGet)

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

func (relay *Relay) handleBountyBids(w http.ResponseWriter, req *http.Request) {
	blockTimestamp := uint64(time.Now().Unix())
	builderBlock := new(builderTypes.BuilderBlockBid)

	if err := json.NewDecoder(req.Body).Decode(&builderBlock); err != nil {
		relay.log.WithError(err).Warn("Could Not Convert Payload To Builder Submisioon")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	slot_time := relay.BlockSlotTimestamp(builderBlock.Message.Slot)
	if builderBlock.Message.ExecutionPayloadHeader.Timestamp != slot_time {
		relay.log.Warnf("incorrect timestamp. got %d, expected %d", builderBlock.Message.ExecutionPayloadHeader.Timestamp, slot_time)
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("incorrect timestamp. got %d", builderBlock.Message.ExecutionPayloadHeader.Timestamp))
		return
	}

	/// @dev Bounty Bid Time Should be [slot_time + 2, slot_time + 3]
	if blockTimestamp > (slot_time+3) || blockTimestamp < (slot_time+2) {
		relay.log.Warnf("Bounty Bid Sent Wrong Time, Got %d, Expecting between %d,%d", blockTimestamp, slot_time-10, slot_time-9)
		relay.RespondError(w, http.StatusBadRequest, "Bounty Bid Sent Wrong Time")
		return
	}

	bountyBidSlot, err := relay.bidBoard.GetBountyBidForSlot(builderBlock.Message.Slot)
	if err != nil {
		relay.log.Warn("could not get bounty bid winner")
		relay.RespondError(w, http.StatusBadRequest, "could not get bounty bid winner")
		return
	}
	if bountyBidSlot != "" {
		relay.log.Warn("Bounty Bid Already Accepted")
		relay.RespondError(w, http.StatusBadRequest, "Bounty Bid Already Accepted")
		return
	}

	openAuctionWinningBid, err := relay.bidBoard.GetOpenAuctionHighestBid(builderBlock.Message.Slot)
	if builderBlock.Message.Value.Cmp(big.NewInt(0).Mul(openAuctionWinningBid, big.NewInt(2))) == -1 {
		relay.log.Warn("Bounty Amount Not Sufficient")
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Bounty Amount Not Sufficient, Expecting %d", big.NewInt(0).Mul(openAuctionWinningBid, big.NewInt(2))))
		return
	}

	if builderBlock.Message.ExecutionPayloadHeader.WithdrawalsRoot == EmptyWithdrawalsRoot {
		relay.log.Warn("Empty Withdrawal")
		relay.RespondError(w, http.StatusBadRequest, "Empty Withdrawal")
		return
	}

	status, err := relay.relayutils.BuilderStatus(builderBlock.Message.BuilderWalletAddress.String())
	if err != nil {
		relay.log.WithError(err).Warn("Couldn' Get Builder Status")
		relay.RespondError(w, http.StatusBadRequest, "Failed To Get Builder")
		return
	}

	if !status {
		relay.log.Warnf("Builder Not Active In PON, Builder- %s", builderBlock.Message.BuilderWalletAddress.String())
		relay.RespondError(w, http.StatusBadRequest, "Builder Not Active In PON")
		return
	}

	deliveredPayloadBuilder, err := relay.bidBoard.GetPayloadDelivered(builderBlock.Message.Slot)
	if err != nil && !errors.Is(err, redis.Nil) {
		relay.log.WithError(err).Error("failed to get delivered payload slot from redis")
		relay.RespondError(w, http.StatusBadRequest, "failed to get delivered payload slot from redis")
		return
	} else if err != nil && errors.Is(err, redis.Nil) {
		relay.log.Info("No Payload Sent For Slot")
	} else {
		relay.log.WithError(err).Error("Payload Delivered For Slot")
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Payload For Slot %d Delivered For Builder %s", builderBlock.Message.Slot, deliveredPayloadBuilder))
		return
	}

	relay.beaconClient.BeaconData.Mu.Lock()
	relaySlot := relay.beaconClient.BeaconData.CurrentSlot
	relay.beaconClient.BeaconData.Mu.Unlock()
	if builderBlock.Message.Slot != relaySlot && builderBlock.Message.Slot != relaySlot+1 {
		relay.log.Warnf("submitBlock failed: submission for wrong slot, Expected %d or %d, Got %d", relaySlot, relaySlot+1, builderBlock.Message.Slot)
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("submission for wrong slot, Expected %d or %d, Got %d", relaySlot, relaySlot+1, builderBlock.Message.Slot))
		return
	}

	err = SanityBuilderBlock(*builderBlock)
	if err != nil {
		relay.log.WithError(err).Error("block submission sanity checks failed")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	builderRPBS, err := rpbs.Verify(*builderBlock)
	if err != nil {
		relay.log.WithError(err).Error("RPBS Verify Error")
		relay.RespondError(w, http.StatusInternalServerError, "RPBS Verify Error")
		return
	}
	if !builderRPBS {
		relay.log.Error("RPBS Verify Failed")
		relay.RespondError(w, http.StatusBadRequest, "RPBS Verify Failed")
		return
	}

	blockBidMsgBytes, err := builderBlock.Message.HashTreeRoot()
	if err != nil {
		relay.log.Error("could get block bid message hash tree root", "err", err)
		relay.RespondError(w, http.StatusBadRequest, "could get block bid message hash tree root")
		return
	}

	pubkey, err := crypto.Ecrecover(blockBidMsgBytes[:], builderBlock.EcdsaSignature[:])
	if err != nil {
		relay.log.Error("Could not recover ECDSA pubkey", "err", err)
		relay.RespondError(w, http.StatusInternalServerError, "Could not recover ECDSA pubkey")
		return
	}

	ecdsaPubkey, err := crypto.UnmarshalPubkey(pubkey)
	if err != nil {
		relay.log.Error("Could not recover ECDSA pubkey", "err", err)
		relay.RespondError(w, http.StatusInternalServerError, "Could not recover ECDSA pubkey")
		return
	}

	pubkeyAddress := crypto.PubkeyToAddress(*ecdsaPubkey)
	if strings.ToLower(pubkeyAddress.String()) != strings.ToLower(builderBlock.Message.BuilderWalletAddress.String()) {
		relay.log.Errorf("ECDSA pubkey does not match wallet address %s pubkeyAddress %s", pubkeyAddress.String(), builderBlock.Message.BuilderWalletAddress.String())
		relay.RespondError(w, http.StatusBadRequest, "ECDSA pubkey does not match wallet address")
		return
	}

	/* @dev 
		Once the public key is obained and verified from the signature as that
		of the builder, we can check if this public key signed the block bid message, 
		and not some other data.
	*/
	blockBidMsgBytes, err := builderBlock.Message.HashTreeRoot()
	if err != nil {
		relay.log.Error("could not marshal block bid msg", "err", err)
		relay.RespondError(w, http.StatusInternalServerError, "could not marshal block bid msg")
		return
	}

	var ecdsaSignature struct {
		R, S *big.Int
	}
	_, err = asn1.Unmarshal(builderBlock.EcdsaSignature[:], &ecdsaSignature)
	if err != nil {
		relay.log.Error("Failed to parse ECDSA signature")
		relay.RespondError(w, http.StatusBadRequest, "Failed to parse ECDSA signature")
		return
	}

	// @dev Verify the signature was created over the block bid message.
	valid := ecdsa.Verify(ecdsaPubkey, blockBidMsgBytes[:], ecdsaSignature.R, ecdsaSignature.S)
	if !valid {
		relay.log.Error("ECDSA Signature Invalid")
		relay.RespondError(w, http.StatusBadRequest, "ECDSA Signature Invalid")
		return
	}

	/// @dev Sees if builder submitted another bid while we are working with this Bid.
	lastBid, err := relay.bidBoard.BuilderBlockLast(builderBlock.Message.Slot, builderBlock.Message.BuilderWalletAddress.String())
	if err != nil {
		if err != redis.Nil {
			relay.log.WithError(err).Error("failed getting latest payload receivedAt from redis")
			relay.RespondError(w, http.StatusInternalServerError, "failed getting latest payload receivedAt from redis")
		}
	} else if blockTimestamp < uint64(lastBid) {
		relay.log.Error("Builder Submitted Another Bounty Bid, Stopping This Bid......")
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Using newer bid for Builder %s", builderBlock.Message.BuilderWalletAddress.String()))
		return
	}

	///////////////////////////////////////////////////////////////////////////
	//             SANITY CHECKS END HERE BID GOOD TO GO
	///////////////////////////////////////////////////////////////////////////

	signedBuilderBid, err := SignedBuilderBid(*builderBlock, relay.blsSk, relay.publicKey, relay.network.DomainBuilder)
	if err != nil {
		relay.log.WithError(err).Error("could not sign builder bid")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
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

	/// @dev We send builder to store that this builder won the bounty bid.
	/// @dev It sends false if some builder has already won the bounty bid while we were working with the bid
	/// @dev If thats not the case it will set this builder as winner of bounty bid
	bountyBidWon, err := relay.bidBoard.SetBountyBidForSlot(builderBlock.Message.Slot, builderBlock.Message.BuilderWalletAddress.String())
	if err != nil {
		relay.log.WithError(err).Error("Could Not Set Bounty Bid")
		relay.RespondError(w, http.StatusBadRequest, "Could Not Set Bounty Bid")
		return
	}
	if !bountyBidWon {
		relay.log.WithError(err).Error("Bounty Bid Won By Other Builder")
		relay.RespondError(w, http.StatusBadRequest, "Bounty Bid Won By Other Builder")
		return
	}

	err = relay.bidBoard.SavePayloadUtils(builderBlock.Message.Slot, builderBlock.Message.ProposerPubkey.String(), builderBlock.Message.BlockHash.String(), &getPayloadHeaderResponse)
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

	highestBidBuilder, highestBidValue, err := relay.bidBoard.AuctionBid(builderBlock.Message.Slot)
	if err != nil {
		relay.log.WithError(err).Error("could not compute top bid")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rpbsString, err := json.Marshal(*builderBlock.Message.RPBS)
	if err != nil {
		relay.log.Errorf("Couldn't Get RPBS String")
		relay.RespondError(w, http.StatusBadRequest, "Couldn't Get RPBS String")
		return
	}
	builderDbBlock := databaseTypes.BuilderBlockDatabase{
		Slot:             builderBlock.Message.Slot,
		BuilderPubkey:    builderBlock.Message.BuilderWalletAddress.String(),
		BuilderBidHash:   builderBlock.Message.ExecutionPayloadHeader.BlockHash.String(),
		BuilderSignature: builderBlock.EcdsaSignature.String(),
		RPBS:             string(rpbsString),
		RpbsPublicKey:    builderBlock.Message.RPBSPubkey,
		TransactionByte:  hexutil.Encode(builderBlock.Message.PayoutPoolTransaction),
		BidValue:         *builderBlock.Message.Value,
	}

	defer func() {
		err := relay.db.PutBuilderBlockSubmission(req.Context(), builderDbBlock)
		if err != nil {
			relay.log.WithError(err).WithField("payload", builderDbBlock).Error("saving builder block submission to database failed")
			return
		}
	}()

	builderBid := &BuilderWinningBid{
		BidID:             builderDbBlock.Hash(),
		HighestBidValue:   highestBidValue,
		HighestBidBuilder: highestBidBuilder,
	}

	relay.log.WithFields(logrus.Fields{
		"Builder": builderBlock.Message.BuilderWalletAddress.String(),
		"Value":   builderBlock.Message.Value,
	}).Info("received block from builder")

	relay.RespondOK(w, &builderBid)

}

func (relay *Relay) handleSubmitBlock(w http.ResponseWriter, req *http.Request) {

	blockTimestamp := time.Now()
	builderBlock := new(builderTypes.BuilderBlockBid)

	if err := json.NewDecoder(req.Body).Decode(&builderBlock); err != nil {
		relay.log.WithError(err).Warn("Could Not Convert Patload To Builder Submisioon")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Garbage Penalty Should be Penalised

	if builderBlock.Message.ExecutionPayloadHeader.WithdrawalsRoot == EmptyWithdrawalsRoot {
		relay.log.Warn("Empty Withdrawal")
		relay.RespondError(w, http.StatusBadRequest, "Empty Withdrawal")
		return
	}

	status, err := relay.relayutils.BuilderStatus(builderBlock.Message.BuilderWalletAddress.String())
	if err != nil {
		relay.log.WithError(err).Warn("Couldn' Get Builder Status")
		relay.RespondError(w, http.StatusBadRequest, "Failed To Get Builder")
		return
	}
	if !status {
		relay.log.Warnf("Builder Not Active In PON, Builder- %s", builderBlock.Message.BuilderWalletAddress.String())
		relay.RespondError(w, http.StatusBadRequest, "Builder Not Active In PON")
		return
	}

	slot_time := relay.BlockSlotTimestamp(builderBlock.Message.Slot)
	if builderBlock.Message.ExecutionPayloadHeader.Timestamp != slot_time {
		relay.log.Warnf("incorrect timestamp. got %d, expected %d", builderBlock.Message.ExecutionPayloadHeader.Timestamp, slot_time)
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("incorrect timestamp. got %d", builderBlock.Message.ExecutionPayloadHeader.Timestamp))
		return
	}

	deliveredPayloadBuilder, err := relay.bidBoard.GetPayloadDelivered(builderBlock.Message.Slot)
	if err != nil && !errors.Is(err, redis.Nil) {
		relay.log.WithError(err).Error("failed to get delivered payload slot from redis")
		relay.RespondError(w, http.StatusBadRequest, "failed to get delivered payload slot from redis")
		return
	} else if err != nil && errors.Is(err, redis.Nil) {
		relay.log.Info("No Payload Sent For Slot")
	} else {
		relay.log.Errorf("Payload Delivered For Slot %d, You Are Late.....", builderBlock.Message.Slot)
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Payload For Slot %d Delivered For Builder %s", builderBlock.Message.Slot, deliveredPayloadBuilder))
		return
	}

	relay.beaconClient.BeaconData.Mu.Lock()
	relaySlot := relay.beaconClient.BeaconData.CurrentSlot
	relay.beaconClient.BeaconData.Mu.Unlock()
	if builderBlock.Message.Slot != relaySlot && builderBlock.Message.Slot != relaySlot+1 {
		relay.log.Warnf("submitBlock failed: submission for wrong slot, Expected %d or %d, Got %d", relaySlot, relaySlot+1, builderBlock.Message.Slot)
		relay.RespondError(w, http.StatusBadRequest, fmt.Sprintf("submission for wrong slot, Expected %d or %d, Got %d", relaySlot, relaySlot+1, builderBlock.Message.Slot))
		return
	}

	err = SanityBuilderBlock(*builderBlock)
	if err != nil {
		relay.log.WithError(err).Error("block submission sanity checks failed")
		relay.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	builderRPBS, err := rpbs.Verify(*builderBlock)
	if err != nil {
		relay.log.WithError(err).Error("RPBS Verify Error")
		relay.RespondError(w, http.StatusInternalServerError, "RPBS Verify Error")
		return
	}
	if !builderRPBS {
		relay.log.Error("RPBS Verify Failed")
		relay.RespondError(w, http.StatusBadRequest, "RPBS Verify Failed")
		return
	}

	blockBidMsgBytes, err := builderBlock.Message.HashTreeRoot()
	if err != nil {
		relay.log.Error("could get block bid message hash tree root", "err", err)
		relay.RespondError(w, http.StatusBadRequest, "could get block bid message hash tree root")
		return
	}

	pubkey, err := crypto.Ecrecover(blockBidMsgBytes[:], builderBlock.EcdsaSignature[:])
	if err != nil {
		relay.log.Error("Could not recover ECDSA pubkey", "err", err)
		relay.RespondError(w, http.StatusInternalServerError, "Could not recover ECDSA pubkey")
		return
	}

	ecdsaPubkey, err := crypto.UnmarshalPubkey(pubkey)
	if err != nil {
		relay.log.Error("Could not recover ECDSA pubkey", "err", err)
		relay.RespondError(w, http.StatusInternalServerError, "Could not recover ECDSA pubkey")
		return
	}

	pubkeyAddress := crypto.PubkeyToAddress(*ecdsaPubkey)
	if strings.ToLower(pubkeyAddress.String()) != strings.ToLower(builderBlock.Message.BuilderWalletAddress.String()) {
		relay.log.Errorf("ECDSA pubkey does not match wallet address %s pubkeyAddress %s", pubkeyAddress.String(), builderBlock.Message.BuilderWalletAddress.String())
		relay.RespondError(w, http.StatusBadRequest, "ECDSA pubkey does not match wallet address")
		return
	}
	rpbsString, err := json.Marshal(*builderBlock.Message.RPBS)
	if err != nil {
		relay.log.Errorf("Couldn't Get RPBS String")
		relay.RespondError(w, http.StatusBadRequest, "Couldn't Get RPBS String")
		return
	}
	builderDbBlock := databaseTypes.BuilderBlockDatabase{
		Slot:             builderBlock.Message.Slot,
		BuilderPubkey:    builderBlock.Message.BuilderWalletAddress.String(),
		BuilderSignature: builderBlock.EcdsaSignature.String(),
		BuilderBidHash:   builderBlock.Message.ExecutionPayloadHeader.BlockHash.String(),
		RPBS:             string(rpbsString),
		RpbsPublicKey:    builderBlock.Message.RPBSPubkey,
		TransactionByte:  hexutil.Encode(builderBlock.Message.PayoutPoolTransaction),
		BidValue:         *builderBlock.Message.Value,
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
		if err != redis.Nil {
			relay.log.WithError(err).Error("failed getting latest payload receivedAt from redis")
			relay.RespondError(w, http.StatusInternalServerError, "failed getting latest payload receivedAt from redis")
		}
	} else if blockTimestamp.Unix() < lastBid {
		relay.log.Error("Bid After Given Bid")
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

	err = relay.bidBoard.SavePayloadUtils(builderBlock.Message.Slot, builderBlock.Message.ProposerPubkey.String(), builderBlock.Message.BlockHash.String(), &getPayloadHeaderResponse)
	if err != nil {
		relay.log.WithError(err).Error("failed saving execution payload in redis")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = relay.bidBoard.SaveBuilderBid(
		builderBlock.Message.Slot,
		builderBlock.Message.BuilderWalletAddress.String(),
		builderBlock.Message.ProposerPubkey.String(),
		uint64(blockTimestamp.Unix()),
		&getHeaderResponse,
	)
	if err != nil {
		relay.log.WithError(err).Error("could not save latest builder bid")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	highestBidBuilder, highestBidValue, err := relay.bidBoard.AuctionBid(builderBlock.Message.Slot)
	if err != nil {
		relay.log.WithError(err).Error("could not compute top bid")
		relay.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	builderBid := &BuilderWinningBid{
		BidID:             builderDbBlock.Hash(),
		HighestBidValue:   highestBidValue,
		HighestBidBuilder: highestBidBuilder,
	}

	relay.log.WithFields(logrus.Fields{
		"Builder": builderBlock.Message.BuilderWalletAddress.String(),
		"Value":   builderBlock.Message.Value,
	}).Info("received block from builder")

	relay.RespondOK(w, &builderBid)

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
	}).Info("Get Header Requested From Proposer To Relay")

	bid, err := relay.bidBoard.WinningBid(proposerReq.Slot)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			relay.log.WithFields(logrus.Fields{
				"slot": proposerReq.Slot,
			}).Warn("No Bids Available")
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
		BlockHash:      bid.Bid.Data.Message.Header.BlockHash.String(),
		ProposerPubkey: proposerReq.ProposerPubKeyHex,
		BidValue:       bid.Bid.Data.Message.Value.Uint64(),
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
		"value":     bid.Bid.Data.Message.Value.String(),
		"blockHash": bid.Bid.Data.Message.Header.BlockHash.String(),
	}).Info("Bid Delivered To Proposer")

	relay.RespondOK(w, &bid.Bid)
}

func (relay *Relay) handleProposerPayload(mevBoost http.ResponseWriter, req *http.Request) {

	payload := new(capellaAPI.SignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(payload); err != nil {
		relay.log.WithError(err).Warn("Proposer payload request failed to decode")
		relay.RespondError(mevBoost, http.StatusBadRequest, fmt.Sprintf("Proposer payload request failed to decode. %s", err.Error()))
		return
	}

	slot := payload.Message.Slot
	blockHash := payload.Message.Body.ExecutionPayloadHeader.BlockHash.String()
	relay.log.WithFields(logrus.Fields{
		"slot":      slot,
		"blockHash": blockHash,
	}).Info("Proposer GetPayload Request")

	proposerPubkey, err := relay.relayutils.ValidatorIndexToPubkey(uint64(payload.Message.ProposerIndex), relay.network.Network)
	if err != nil {
		relay.log.WithError(err).WithFields(logrus.Fields{
			"proposer": uint64(payload.Message.ProposerIndex),
		}).Error("Could Not Get Proposer Public Key For Proposer")

		relay.RespondError(mevBoost, http.StatusBadRequest, fmt.Sprintf("Could Not Get Proposer Public Key For Proposer %d", uint64(payload.Message.ProposerIndex)))
		return
	}

	if len(proposerPubkey) == 0 {
		relay.log.WithError(err).Error(fmt.Sprintf("Could Not Get Proposer Public Key For Proposer %d", uint64(payload.Message.ProposerIndex)))
		relay.RespondError(mevBoost, http.StatusBadRequest, fmt.Sprintf("Could Not Get Proposer Public Key For Proposer %d", uint64(payload.Message.ProposerIndex)))
		return
	}

	ok, err := signing.VerifySignature(payload.Message, relay.network.DomainBeaconCapella, proposerPubkey[:], payload.Signature[:])
	if !ok || err != nil {
		relay.log.WithError(err).Warn("could not verify payload signature")
		relay.RespondError(mevBoost, http.StatusBadRequest, "could not verify payload signature")
		return
	}

	blockSubmission, err := relay.bidBoard.PayloadUtils(uint64(slot), blockHash)
	if err != nil {
		relay.log.WithError(err).Warn("failed getting builder API")
		relay.RespondError(mevBoost, http.StatusBadRequest, "failed getting builder API")
		return
	}

	proposerBlock := &databaseTypes.ValidatorReturnedBlockDatabase{
		Signature:      payload.Signature.String(),
		Slot:           uint64(slot),
		BlockHash:      blockHash,
		ProposerPubkey: proposerPubkey.String(),
	}

	err = relay.db.PutValidatorReturnedBlock(req.Context(), *proposerBlock)
	if err != nil {
		relay.log.WithError(err).Error("Error Putting Proposer Returned Block In DB")
		relay.RespondError(mevBoost, http.StatusInternalServerError, fmt.Sprintf("Error Putting Proposer Returned Block In DB. %s", err.Error()))
		return
	}

	resp, err := sendHTTPRequest(*relay.client, blockSubmission.API, payload)
	if err != nil {
		relay.log.WithError(err).Error("getPayload request to builder from relay failed")
		relay.RespondError(mevBoost, http.StatusInternalServerError, fmt.Sprintf("getPayload request to builder failed. %s", err.Error()))
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		relay.log.WithError(err).Error("getPayload request to builder from relay failed")
		response, _ := io.ReadAll(resp.Body)
		relay.RespondError(mevBoost, http.StatusBadRequest, fmt.Sprintf("getPayload request to builder from relay failed. %s", string(response)))
		return
	}

	getPayloadResponse := new(capella.ExecutionPayload)
	if err := json.NewDecoder(resp.Body).Decode(&getPayloadResponse); err != nil {
		relay.log.WithError(err).Error("getPayload request from builder failed to decode")
		relay.RespondError(mevBoost, http.StatusBadRequest, fmt.Sprintf("getPayload request from builder failed to decode. %s", err.Error()))
		return
	}

	PayloadResponse := mevBoostAPI.VersionedExecutionPayload{Version: spec.DataVersionCapella, Capella: getPayloadResponse}

	defer func() {
		payloadJSON, _ := json.Marshal(getPayloadResponse)
		if err != nil {
			relay.log.WithError(err).Warn("Failed To Payload JSON")
		}
		payloadDBDelivered := &databaseTypes.ValidatorDeliveredPayloadDatabase{
			Slot:           uint64(slot),
			ProposerPubkey: proposerPubkey.String(),
			BlockHash:      PayloadResponse.Capella.BlockHash.String(),
			Payload:        payloadJSON,
		}
		err = relay.db.PutValidatorDeliveredPayload(context.Background(), *payloadDBDelivered)
		if err != nil {
			relay.log.WithError(err).Error("DB Failed")
		}
	}()

	defer func() {
		errs := relay.bidBoard.PutPayloadDelivered(uint64(slot), blockSubmission.BuilderWalletAddress)
		if errs != nil {
			relay.log.WithError(errs).Error("Couldn't Set Payload Delivered Slot")
		}
	}()

	relay.RespondOK(mevBoost, &PayloadResponse)

	proposerBulletinBoard := bulletinBoardTypes.SlotPayloadRequest{
		Slot:     uint64(payload.Message.Slot),
		Proposer: proposerPubkey.String(),
	}

	relay.bulletinBoard.Channel.SlotPayloadChannel <- proposerBulletinBoard

	relay.log.WithFields(logrus.Fields{
		"Slot": slot,
	}).Info("Payload Delivered To Proposer!")
}
