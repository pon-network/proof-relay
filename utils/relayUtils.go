package utils

import (
	"context"
	"errors"
	"sync"
	"time"

	relayTypes "github.com/bsn-eng/pon-golang-types/relay"
	"github.com/go-redis/redis/v9"
	"github.com/sirupsen/logrus"

	beaconclient "github.com/pon-pbs/bbRelay/beaconinterface"
	"github.com/pon-pbs/bbRelay/database"
	ponpool "github.com/pon-pbs/bbRelay/ponPool"
	"github.com/pon-pbs/bbRelay/redisPackage"
)

type RelayUtils struct {
	db           *database.DatabaseInterface
	beaconClient *beaconclient.MultiBeaconClient
	ponPool      *ponpool.PonRegistrySubgraph

	proposerUtils *ProposerUtils
	builderUtils  *BuilderUtils
	reporterUtils *ReporterUtils
}

func NewRelayUtils(db *database.DatabaseInterface, beaconClient *beaconclient.MultiBeaconClient, ponPool ponpool.PonRegistrySubgraph, redisInterface redisPackage.RedisInterface) *RelayUtils {
	proposerutils := &ProposerUtils{
		ProposerStatus: ProposerUpdates{
			Mu:             sync.Mutex{},
			ValidatorsLast: make(map[string]string),
		},
		Validators: relayTypes.ValidatorIndexes{
			ValidatorPubkeyIndex: make(map[string]uint64),
			ValidatorIndexPubkey: make(map[uint64]string),
			Mu:                   sync.Mutex{},
		},
		RedisInterface: &redisInterface,
		BeaconClient:   beaconClient,
		Log: *logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
			"package": "RelayUtils",
			"utils":   "Proposer",
		}),
	}

	builderutils := &BuilderUtils{
		BuilderLast:    make(map[string]string),
		Mu:             sync.Mutex{},
		RedisInterface: &redisInterface,
		Log: *logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
			"package": "RelayUtils",
			"utils":   "Builder",
		}),
	}

	reporterutils := &ReporterUtils{
		ReporterLast:   make(map[string]bool),
		Mu:             sync.Mutex{},
		RedisInterface: &redisInterface,
		Log: *logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
			"package": "RelayUtils",
			"utils":   "Reporter",
		}),
	}

	return &RelayUtils{
		db:            db,
		beaconClient:  beaconClient,
		ponPool:       &ponPool,
		proposerUtils: proposerutils,
		builderUtils:  builderutils,
		reporterUtils: reporterutils,
	}
}

func (relayUtils *RelayUtils) StartUtils() (err error) {

	go relayUtils.ProposerUpdate()
	go relayUtils.BuilderUpdate()
	go relayUtils.ReporterUpdate()

	return nil
}

func (relay *RelayUtils) ProposerUpdate() {
	for {
		relay.proposerUtils.GetValidators(*relay.ponPool, *relay.db)
		time.Sleep(EpochDuration)
	}
}
func (relay *RelayUtils) BuilderUpdate() {
	for {
		relay.builderUtils.GetBuilders(*relay.ponPool, *relay.db)
		time.Sleep(EpochDuration)
	}
}

func (relay *RelayUtils) ReporterUpdate() {
	for {
		relay.reporterUtils.GetReporters(*relay.ponPool, *relay.db)
		time.Sleep(EpochDuration)
	}
}

func (proposer *ProposerUtils) GetValidators(ponPool ponpool.PonRegistrySubgraph, db database.DatabaseInterface) {
	validators, err := ponPool.GetValidators()
	if err != nil {
		proposer.Log.WithError(err).Error("Failed To Get Validators")
		return
	}

	proposer.ProposerStatus.Mu.Lock()
	defer proposer.ProposerStatus.Mu.Unlock()

	newProposers := []string{}
	proposer.Log.Infof("Updating %d Validators in Redis...", len(validators))
	for _, validator := range validators {
		if validator.Status == proposer.ProposerStatus.ValidatorsLast[validator.ValidatorPubkey] {
			continue
		}
		err = proposer.SetValidatorStatus(validator.ValidatorPubkey, validator.Status)
		if err != nil {
			proposer.Log.WithError(err).Error("failed to set block builder status in redis")
		}
		proposer.ProposerStatus.ValidatorsLast[validator.ValidatorPubkey] = validator.Status
		proposer.Validators.Mu.Lock()
		_, ok := proposer.Validators.ValidatorPubkeyIndex[validator.ValidatorPubkey]
		if !ok {
			newProposers = append(newProposers, validator.ValidatorPubkey)
		}
		proposer.Validators.Mu.Unlock()

	}
	if len(newProposers) != 0 {
		proposer.Log.Infof("Updating Proposer Index For %d Validators", len(newProposers))
		go proposer.ValidatorIndex(newProposers)
	}

	proposer.Log.Infof("Updating %d Validators in Database...", len(validators))
	err = db.PutValidators(validators)
	if err != nil {
		proposer.Log.WithError(err).Error("failed to save block Validators")
	}
}

func (proposer *ProposerUtils) ValidatorIndex(proposers []string) {
	ValidatorGroups := chunkSlice(proposers, 10)

	for _, validators := range ValidatorGroups {
		go proposer.BeaconClient.GetValidatorIndex(validators, &proposer.Validators)
	}
}

func (builderInterface *BuilderUtils) GetBuilders(ponPool ponpool.PonRegistrySubgraph, db database.DatabaseInterface) {
	builders, err := ponPool.GetBuilders()
	if err != nil {
		builderInterface.Log.WithError(err).Error("Failed To Get Builders")
		return
	}

	builderInterface.Mu.Lock()
	defer builderInterface.Mu.Unlock()
	builderInterface.Log.Infof("Updating %d block builders in Redis...", len(builders))
	for _, builder := range builders {
		if builder.Status == builderInterface.BuilderLast[builder.BuilderPubkey] {
			continue
		}
		err = builderInterface.SetBuilderStatus(builder.BuilderPubkey, builder.Status)
		if err != nil {
			builderInterface.Log.WithError(err).Error("failed to set block builder status in redis")
		}
		builderInterface.BuilderLast[builder.BuilderPubkey] = builder.Status
	}
	builderInterface.Log.Infof("Updating %d block builders in Database...", len(builders))
	err = db.PutBuilders(builders)
	if err != nil {
		builderInterface.Log.WithError(err).Error("failed to save block builders")
	}
}

func (reporterInterface *ReporterUtils) GetReporters(ponPool ponpool.PonRegistrySubgraph, db database.DatabaseInterface) {
	reporters, err := ponPool.GetReporters()
	if err != nil {
		reporterInterface.Log.WithError(err).Error("Failed To Get Reporters")
		return
	}

	reporterInterface.Mu.Lock()
	defer reporterInterface.Mu.Unlock()

	for _, reporter := range reporters {
		if reporter.Active == reporterInterface.ReporterLast[reporter.ReporterPubkey] {
			continue
		}
		err = reporterInterface.SetReporterStatus(reporter.ReporterPubkey, reporter.Active)
		if err != nil {
			reporterInterface.Log.WithError(err).Error("failed to set reporter status in redis")
		}
		reporterInterface.ReporterLast[reporter.ReporterPubkey] = reporter.Active
	}

	reporterInterface.Log.Infof("Updating %d Reporters in Database...", len(reporters))
	err = db.PutReporters(reporters)
	if err != nil {
		reporterInterface.Log.WithError(err).Error("failed to save reporters")
	}
}

func (proposerInterface *ProposerUtils) SetValidatorStatus(validator string, status string) error {
	return proposerInterface.RedisInterface.Client.HSet(context.Background(), keyValidatorStatus, validator, status).Err()
}

func (builderInterface *BuilderUtils) SetBuilderStatus(builder string, status string) error {
	return builderInterface.RedisInterface.Client.HSet(context.Background(), keyBuilderStatus, builder, status).Err()
}

func (reporterInterface *ReporterUtils) SetReporterStatus(reporter string, status bool) error {
	return reporterInterface.RedisInterface.Client.HSet(context.Background(), keyReporterrStatus, reporter, status).Err()
}

func (relay *RelayUtils) BuilderStatus(builder string) (BuilderStatus bool, err error) {
	res, err := relay.builderUtils.RedisInterface.Client.HGet(context.Background(), keyBuilderStatus, builder).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	status := res == "1"
	return status, err
}

func (relay *RelayUtils) ValidatorIndexToPubkey(index uint64, network uint64) (PublicKey, error) {
	relay.proposerUtils.Validators.Mu.Lock()
	defer relay.proposerUtils.Validators.Mu.Unlock()
	validator := relay.proposerUtils.Validators.ValidatorIndexPubkey[index]

	// For custom testnet
	if network == 2 {
		if index == 0 {
			validator = "0xa99a76ed7796f7be22d5b7e85deeb7c5677e88e511e0b337618f8c4eb61349b4bf2d153f649f7b53359fe8b94a38e44c"
		}
		if index == 1 {
			validator = "0xb89bebc699769726a318c8e9971bd3171297c61aea4a6578a7a4f94b547dcba5bac16a89108b6b6a1fe3695d1a874a0b"
		}
		if index == 2 {
			validator = "0xa3a32b0f8b4ddb83f1a0a853d81dd725dfe577d4f4c3db8ece52ce2b026eca84815c1a7e8e92a4de3d755733bf7e4a9b"
		}
	}

	var validatorPublicKey PublicKey
	err := validatorPublicKey.UnmarshalText([]byte(validator))
	if err != nil {
		return PublicKey{}, err
	}
	return validatorPublicKey, nil
}
