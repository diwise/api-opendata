package main

import (
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/application"
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/repositories/database"
)

func main() {
	serviceName := "api-opendata"

	log := logging.NewLogger()
	log.Infof("Starting up %s ...", serviceName)

	db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(), log)
	application.CreateRouterAndStartServing(log, db)
}
