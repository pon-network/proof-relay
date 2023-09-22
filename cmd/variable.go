package cmd

import (
	"github.com/pon-pbs/bbRelay/bls"
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
	deleteTables          bool
	discordWebhook        string
)

var (
	relayDefaultURL              = "localhost:9062"
	apiDefaultSecretKey          = ""
	defaultNetwork               = "Ethereum"
	defaultPostgresURL           = ""
	defaultRedisURI              = "redis://localhost:6379"
	defaultBeaconURIs            = []string{"http://localhost:3500"}
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
	bulletinBoardUsernameDefault = ""
	reporterURLDefault           = "localhost:9001"
	bidTimeoutDefault            = "15s"
	readTimeoutDefault           = "10s"
	readHeaderTimeoutDefault     = "10s"
	writeTimeoutDefault          = "10s"
	idleTimeoutDefault           = "10s"
	deleteTablesDefault          = false
	discordWebhookDefault        = ""
)

var RelayVersion = "dev"
