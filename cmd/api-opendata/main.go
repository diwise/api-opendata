package main

import (
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/repositories/database"
)

func main() {
	serviceName := "api-opendata"

	log := logging.NewLogger()
	log.Infof("Starting up %s ...", serviceName)

	_, err := database.NewDatabaseConnection(database.NewSQLiteConnector(), log)
	if err != nil {
		log.Error("failed to connect to datebase: %d", err)
	}
}
