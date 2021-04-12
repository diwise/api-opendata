package main

import (
	"github.com/diwise/api-opendata/internal/pkg/application"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
)

func main() {
	serviceName := "api-opendata"

	log := logging.NewLogger()
	log.Infof("Starting up %s ...", serviceName)

	db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(), log)
	application.CreateRouterAndStartServing(log, db)
}
