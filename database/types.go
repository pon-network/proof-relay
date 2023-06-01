package database

import (
	"database/sql"

	databaseTypes "github.com/bsn-eng/pon-golang-types/database"
	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sirupsen/logrus"
)

type DatabaseInterface struct {
	DB     *sql.DB // Function so we have functions on top of it
	Opts   databaseTypes.DatabaseOpts
	Driver databaseTypes.DatabaseDriver
	Log    logrus.Entry
	URL    string
}

func (database *DatabaseInterface) NewDatabaseOpts() {

	database.DB.SetMaxOpenConns(database.Opts.MaxConnections)

	database.DB.SetMaxIdleConns(database.Opts.MaxIdleConnections)

	database.DB.SetConnMaxIdleTime(database.Opts.MaxIdleTimeConnection)
}

func (database *DatabaseInterface) DBMigrate() error {
	migrationOpts, err := iofs.New(databaseTypes.Content, "migrations/")
	if err != nil {
		database.Log.Fatal(err)
		return err
	}

	migration, err := migrate.NewWithSourceInstance("iofs", migrationOpts, database.URL)
	if err != nil {
		database.Log.Fatal(err)
		return err
	}

	defer migration.Close()

	err = migration.Up()
	if err != nil {
		database.Log.Fatal("Database Migrate Error")
		return err
	}

	database.Log.Info("Database Migrate Succesful")
	return nil
}
