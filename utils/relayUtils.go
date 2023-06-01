package utils

import (
	"context"
	"errors"
	"sync"
	"time"

	beaconclient "github.com/bsn-eng/pon-wtfpl-relay/beaconinterface"
	"github.com/bsn-eng/pon-wtfpl-relay/database"
	ponpool "github.com/bsn-eng/pon-wtfpl-relay/ponPool"
	"github.com/bsn-eng/pon-wtfpl-relay/redisPackage"
	"github.com/go-redis/redis/v9"
	"github.com/sirupsen/logrus"
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
		ValidatorsLast: make(map[string]string),
		Mu:             sync.Mutex{},
		RedisInterface: &redisInterface,
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

	proposer.Mu.Lock()
	defer proposer.Mu.Unlock()

	proposer.Log.Infof("Updating %d Validators in Redis...", len(validators))
	for _, validator := range validators {
		if validator.Status == proposer.ValidatorsLast[validator.ValidatorPubkey] {
			continue
		}
		err = proposer.SetValidatorStatus(validator.ValidatorPubkey, validator.Status)
		if err != nil {
			proposer.Log.WithError(err).Error("failed to set block builder status in redis")
		}
		proposer.ValidatorsLast[validator.ValidatorPubkey] = validator.Status
	}
	proposer.Log.Infof("Updating %d Validators in Database...", len(validators))
	err = db.PutValidators(validators)
	if err != nil {
		proposer.Log.WithError(err).Error("failed to save block Validators")
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

func (relay *RelayUtils) ValidatorIndexToPubkey(index uint64) (PublicKey, error) {
	validator, err := relay.beaconClient.RetrieveValidatorByIndex(index)
	if err != nil {
		return PublicKey{}, err
	}
	var validatorPublicKey PublicKey
	err = validatorPublicKey.UnmarshalText([]byte(validator.Validator.Pubkey))
	if err != nil {
		return PublicKey{}, err
	}
	return validatorPublicKey, nil
}
