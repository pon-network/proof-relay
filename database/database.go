// Package database exposes the postgres database
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	ponPoolTypes "github.com/bsn-eng/pon-golang-types/ponPool"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func NewDatabase(url string,
	parameters databaseTypes.DatabaseOpts,
	dbDriver databaseTypes.DatabaseDriver, deleteTables bool) (*DatabaseInterface, error) {

	database, err := sql.Open(string(dbDriver), url)
	if err != nil {
		return nil, err
	}

	dbInterface := &DatabaseInterface{
		DB:     database,
		Opts:   parameters,
		Driver: dbDriver,
		URL:    url,
		Log: *logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{
			"package": "Database",
		})}

	if deleteTables {
		err := dbInterface.purgeDatabase()
		if err != nil {
			return nil, err
		}
	}

	// Apply migration
	currentDir, err := os.Getwd()
	migrationFilePath := filepath.Join(currentDir, "database", "migrations", "0001_initialize_tables.up.sql")

	if err := dbInterface.applyMigration(migrationFilePath); err != nil {
		return nil, err
	}

	dbInterface.NewDatabaseOpts()
	logrus.WithFields(logrus.Fields{
		"Max Connections":      parameters.MaxConnections,
		"Max Idle Connections": parameters.MaxIdleConnections,
		"Max Timeout":          parameters.MaxIdleTimeConnection,
	}).Info("Database Opts")

	return dbInterface, err
}

func (db *DatabaseInterface) purgeDatabase() error {
	db.Log.Info("Deleting Tables")
	currentDir, err := os.Getwd()
	migrationFilePath := filepath.Join(currentDir, "database", "migrations", "0001_remove_tables.down.sql")
	migrationSQL, err := os.ReadFile(migrationFilePath)
	if err != nil {
		return err
	}

	_, err = db.DB.Exec(string(migrationSQL))
	if err != nil {
		return err
	}

	return nil
}

func (db *DatabaseInterface) applyMigration(migrationFilePath string) error {
	migrationSQL, err := os.ReadFile(migrationFilePath)
	if err != nil {
		return err
	}

	_, err = db.DB.Exec(string(migrationSQL))
	if err != nil {
		return err
	}

	return nil
}

func (database *DatabaseInterface) PutValidatorDeliveredPayload(ctx context.Context,
	validatorPayload databaseTypes.ValidatorDeliveredPayloadDatabase) error {

	payloadJSON, err := json.Marshal(validatorPayload.Payload)
	if err != nil {
		return err
	}

	query := `INSERT INTO validator_payloads_delivered
		(slot, proposer_pubkey, block_hash, payload) VALUES
		($1, $2, $3, $4)`
	_, err = database.DB.ExecContext(
		ctx,
		query,
		validatorPayload.Slot,
		validatorPayload.ProposerPubkey,
		validatorPayload.BlockHash,
		payloadJSON,
	)

	return err
}

func (database *DatabaseInterface) PutValidatorReturnedBlock(ctx context.Context,
	returnedBlock databaseTypes.ValidatorReturnedBlockDatabase) (err error) {

	query := `INSERT INTO validator_returned_blocks
		(signature, slot, block_hash, proposer_pubkey) VALUES
		($1, $2, $3, $4)`
	_, err = database.DB.ExecContext(
		ctx,
		query,
		returnedBlock.Signature[:48],
		returnedBlock.Slot,
		returnedBlock.BlockHash,
		returnedBlock.ProposerPubkey,
	)

	return err
}

func (database *DatabaseInterface) PutValidatorDeliveredHeader(ctx context.Context,
	validatorDeliveredHeader databaseTypes.ValidatorDeliveredHeaderDatabase) error {

	query := `INSERT INTO validator_header_delivered
		(slot, proposer_pubkey, block_hash, bid_value) VALUES
		($1, $2, $3, $4)`
	_, err := database.DB.ExecContext(
		ctx,
		query,
		validatorDeliveredHeader.Slot,
		validatorDeliveredHeader.ProposerPubkey,
		validatorDeliveredHeader.BlockHash,
		validatorDeliveredHeader.BidValue,
	)

	return err
}

func (database *DatabaseInterface) PutBuilderBlockSubmission(ctx context.Context,
	builderSubmission databaseTypes.BuilderBlockDatabase) error {

	query := `INSERT INTO builder_block_submissions
		(id, slot, builder_pubkey, bid_value, builder_signature, block_hash, rpbs, rpbs_public_key, transaction_byte) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := database.DB.ExecContext(
		ctx,
		query,
		builderSubmission.Hash()[:32],
		builderSubmission.Slot,
		builderSubmission.BuilderPubkey,
		builderSubmission.BidValue.String(),
		builderSubmission.BuilderSignature,
		builderSubmission.BuilderBidHash,
		builderSubmission.RPBS,
		builderSubmission.RpbsPublicKey,
		builderSubmission.TransactionByte,
	)

	return err
}

func (database *DatabaseInterface) PutReporters(reporters []ponPoolTypes.Reporter) error {

	query := `INSERT INTO reporters
		(reporter_pubkey, active, report_count) VALUES
		($1, $2, $3) ON CONFLICT (reporter_pubkey) DO UPDATE SET
		active = $2, report_count = $3`
	for _, reporter := range reporters {
		_, err := database.DB.ExecContext(
			context.Background(),
			query,
			reporter.ReporterPubkey,
			reporter.Active,
			reporter.ReportCount,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (database *DatabaseInterface) PutBuilders(builders []ponPoolTypes.BuilderInterface) error {

	query := `INSERT INTO block_builders
		(builder_pubkey, builder_stake, status) VALUES ($1, $2, $3) 
		ON CONFLICT (builder_pubkey) DO UPDATE SET
		status = $3`
	for _, builder := range builders {
		_, err := database.DB.ExecContext(
			context.Background(),
			query,
			builder.Builder.BuilderPubkey,
			builder.Builder.BalanceStaked,
			builder.Status,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (database *DatabaseInterface) PutValidators(validators []ponPoolTypes.Validator) error {

	query := `INSERT INTO validators
		(validator_pubkey, status, report_count) VALUES
		($1, $2, $3) ON CONFLICT (validator_pubkey) DO UPDATE SET
		status = $2, report_count = $3`
	for _, validator := range validators {
		_, err := database.DB.ExecContext(
			context.Background(),
			query,
			validator.ValidatorPubkey,
			validator.Status,
			validator.ReportCount,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// Functions For Reporter To Get Bids Between Slots

func (database *DatabaseInterface) GetBuilderBlocksReporter(ctx context.Context,
	slotFrom uint64,
	slotTo uint64) (*[]databaseTypes.BuilderBlockDatabase, error) {

	query := `SELECT slot, builder_pubkey, block_hash, builder_signature, rpbs, rpbs_public_key, transaction_byte, bid_value
	FROM builder_block_submissions
	WHERE slot BETWEEN $1 AND $2
	ORDER BY slot ASC`

	rows, err := database.DB.QueryContext(ctx, query, slotFrom, slotTo)
	switch {
	case err == sql.ErrNoRows:
		database.Log.WithFields(logrus.Fields{
			"Slot From": slotFrom,
			"Slot To":   slotTo,
		}).Info("No Builder Submissions")
		return &[]databaseTypes.BuilderBlockDatabase{}, nil

	case err != nil:
		return nil, err

	default:
	}

	blockBuilders := []databaseTypes.BuilderBlockDatabase{}

	for rows.Next() {
		builder := databaseTypes.BuilderBlockDatabase{}
		var bidValueString string
		err = rows.Scan(&builder.Slot, &builder.BuilderPubkey, &builder.BuilderBidHash, &builder.BuilderSignature, &builder.RPBS, &builder.RpbsPublicKey, &builder.TransactionByte, &bidValueString)
		if err != nil {
			return nil, err
		}
		builder.BidValue.SetString(bidValueString, 10)
		blockBuilders = append(blockBuilders, builder)
	}

	return &blockBuilders, nil
}

func (database *DatabaseInterface) GetValidatorDeliveredHeaderReporter(ctx context.Context,
	slotFrom uint64,
	slotTo uint64) (*[]databaseTypes.ValidatorDeliveredHeaderDatabase, error) {

	query := `SELECT slot, proposer_pubkey, block_hash, bid_value
	FROM validator_header_delivered
	WHERE slot BETWEEN $1 AND $2
	ORDER BY slot ASC`

	rows, err := database.DB.QueryContext(ctx, query, slotFrom, slotTo)
	switch {
	case err == sql.ErrNoRows:
		database.Log.WithFields(logrus.Fields{
			"Slot From": slotFrom,
			"Slot To":   slotTo,
		}).Info("No Proposer Headers Delivered")
		return &[]databaseTypes.ValidatorDeliveredHeaderDatabase{}, nil

	case err != nil:
		return nil, err

	default:
	}

	proposerBlocks := []databaseTypes.ValidatorDeliveredHeaderDatabase{}

	for rows.Next() {
		proposer := databaseTypes.ValidatorDeliveredHeaderDatabase{}
		err = rows.Scan(&proposer.Slot, &proposer.ProposerPubkey, &proposer.BlockHash, &proposer.BidValue)
		if err != nil {
			return nil, err
		}
		proposerBlocks = append(proposerBlocks, proposer)
	}

	return &proposerBlocks, nil
}

func (database *DatabaseInterface) GetValidatorReturnedBlocksReporter(ctx context.Context,
	slotFrom uint64,
	slotTo uint64) (*[]databaseTypes.ValidatorReturnedBlockDatabase, error) {

	query := `SELECT slot, proposer_pubkey, block_hash, signature
	FROM validator_returned_blocks
	WHERE slot BETWEEN $1 AND $2
	ORDER BY slot ASC`

	rows, err := database.DB.QueryContext(ctx, query, slotFrom, slotTo)
	switch {
	case err == sql.ErrNoRows:
		database.Log.WithFields(logrus.Fields{
			"Slot From": slotFrom,
			"Slot To":   slotTo,
		}).Info("No Builder Submissions")
		return &[]databaseTypes.ValidatorReturnedBlockDatabase{}, nil

	case err != nil:
		return nil, err

	default:
	}

	returnedValidatorBlocks := []databaseTypes.ValidatorReturnedBlockDatabase{}

	for rows.Next() {
		returnedBlock := databaseTypes.ValidatorReturnedBlockDatabase{}
		err = rows.Scan(&returnedBlock.Slot, &returnedBlock.ProposerPubkey, &returnedBlock.BlockHash, &returnedBlock.Signature)
		if err != nil {
			return nil, err
		}
		returnedValidatorBlocks = append(returnedValidatorBlocks, returnedBlock)
	}

	return &returnedValidatorBlocks, nil
}

func (database *DatabaseInterface) GetValidatorDeliveredPayloadReporter(ctx context.Context,
	slotFrom uint64,
	slotTo uint64) (*[]databaseTypes.ValidatorReturnedBlockDatabase, error) {

	query := `SELECT slot, proposer_pubkey, block_hash, signature
	FROM validator_returned_blocks
	WHERE slot BETWEEN $1 AND $2
	ORDER BY slot ASC`

	rows, err := database.DB.QueryContext(ctx, query, slotFrom, slotTo)
	switch {
	case err == sql.ErrNoRows:
		database.Log.WithFields(logrus.Fields{
			"Slot From": slotFrom,
			"Slot To":   slotTo,
		}).Info("No Builder Submissions")
		return &[]databaseTypes.ValidatorReturnedBlockDatabase{}, nil

	case err != nil:
		return nil, err

	default:
	}

	returnedValidatorBlocks := []databaseTypes.ValidatorReturnedBlockDatabase{}

	for rows.Next() {
		returnedBlock := databaseTypes.ValidatorReturnedBlockDatabase{}
		err = rows.Scan(&returnedBlock.Slot, &returnedBlock.ProposerPubkey, &returnedBlock.BlockHash, &returnedBlock.Signature)
		if err != nil {
			return nil, err
		}
		returnedValidatorBlocks = append(returnedValidatorBlocks, returnedBlock)
	}

	return &returnedValidatorBlocks, nil
}
