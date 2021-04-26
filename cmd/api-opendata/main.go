package main

import (
	"bytes"
	"flag"
	"io"
	"os"

	"github.com/diwise/api-opendata/internal/pkg/application"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
)

func openDatasetsFile(log logging.Logger, path string) *os.File {
	datafile, err := os.Open(path)
	if err != nil {
		log.Infof("Failed to open the datasets rdf file %s.", path)
		return nil
	}
	return datafile
}

var datasetFileName string

func main() {
	serviceName := "api-opendata"

	log := logging.NewLogger()
	log.Infof("Starting up %s ...", serviceName)

	flag.StringVar(&datasetFileName, "rdffile", "", "The file to serve datasets from")
	flag.Parse()

	datafile := openDatasetsFile(log, datasetFileName)
	if datafile == nil {
		log.Fatal("Unable to open dataset file. Exiting.")
	} else {
		datasetResponseBuffer := bytes.NewBuffer(nil)
		written, err := io.Copy(datasetResponseBuffer, datafile)
		defer datafile.Close()

		if err == nil {
			log.Infof("Copied %d bytes from %s into response buffer.", written, datasetFileName)

			db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(), log)
			application.CreateRouterAndStartServing(log, db, datasetResponseBuffer)
		}
	}
}
