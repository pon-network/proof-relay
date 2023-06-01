package cmd

import (
	"github.com/bsn-eng/pon-wtfpl-relay/bls"
	"github.com/bsn-eng/pon-wtfpl-relay/constants"
	"github.com/bsn-eng/pon-wtfpl-relay/relay"
	"github.com/bsn-eng/pon-wtfpl-relay/signing"
)

var (
	beaconNodeURIs        []string
	redisURI              string
	postgresURL           string
	ponSubgraph           string
	network               string
	relayURL              string
	apiSecretKey          string
	maxDBConnections      string
	maxIdleConnections    string
	maxTimeConnection     string
	dbDriver              string
	ponPoolURL            string
	ponPoolAPIKey         string
	bulletinBoardBroker   string
	bulletinBoardPort     string
	bulletinBoardClient   string
	bulletinBoardUserName string
	bulletinBoardPassword string
	reporterURL           string
	bidTimeout            string
	secretKey             *bls.SecretKey
	readTimeout           string
	readHeaderTimeout     string
	writeTimeout          string
	idleTimeout           string
	newRelicApp           string
	newRelicLicense       string
	newRelicForwarding    bool
)

var (
	relayDefaultURL              = "localhost:9000"
	apiDefaultSecretKey          = ""
	defaultNetwork               = "Ethereum"
	defaultPostgresURL           = ""
	defaultRedisURI              = ""
	defaultBeaconURIs            = []string{"localhost:3500"}
	maxDBConnectionsDefault      = "100"
	maxIdleConnectionsDefault    = "100"
	maxTimeConnectionDefault     = "100s"
	dbDriverDefault              = "postgres"
	ponPoolURLDefault            = ""
	ponPoolAPIKeyDefault         = ""
	bulletinBoardBrokerDefault   = ""
	bulletinBoardPortDefault     = ""
	bulletinBoardClientDefault   = ""
	bulletinBoardUserNameDefault = ""
	bulletinBoardPasswordDefault = ""
	reporterURLDefault           = "localhost:9001"
	bidTimeoutDefault            = "15s"
	readTimeoutDefault           = "10s"
	readHeaderTimeoutDefault     = "10s"
	writeTimeoutDefault          = "10s"
	idleTimeoutDefault           = "10s"
	newRelicAppDefault           = ""
	newRelicLicenseDefault       = ""
	newRelicForwardingDefault    = true
)

func NewEthNetworkDetails(network string) (*relay.EthNetwork, error) {
	if network == "Ethereum" {
		domainBuilder, err := signing.ComputeDomain(signing.DomainTypeAppBuilder, constants.GenesisForkVersionMainnet, signing.Root{}.String())
		if err != nil {
			return nil, err
		}
		domainBeaconCapella, err := signing.ComputeDomain(signing.DomainTypeBeaconProposer, constants.CapellaForkVersionMainnet, constants.GenesisValidatorsRootMainnet)
		if err != nil {
			return nil, err
		}
		return &relay.EthNetwork{
			Network:             0,
			GenesisTime:         uint64(constants.GenesisTimeMainnet),
			DomainBuilder:       domainBuilder,
			DomainBeaconCapella: domainBeaconCapella,
		}, nil
	}

	if network == "Goerli" {
		domainBuilder, err := signing.ComputeDomain(signing.DomainTypeAppBuilder, constants.GenesisForkVersionGoerli, signing.Root{}.String())
		if err != nil {
			return nil, err
		}
		domainBeaconCapella, err := signing.ComputeDomain(signing.DomainTypeBeaconProposer, constants.CapellaForkVersionGoerli, constants.GenesisValidatorsRootGoerli)
		if err != nil {
			return nil, err
		}
		return &relay.EthNetwork{
			Network:             1,
			GenesisTime:         uint64(constants.GenesisTimeGoerli),
			DomainBuilder:       domainBuilder,
			DomainBeaconCapella: domainBeaconCapella,
		}, nil
	}
	return &relay.EthNetwork{}, nil
}
