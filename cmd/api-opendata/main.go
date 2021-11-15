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
		log.Infof("failed to open the datasets rdf file %s.", path)
		return nil
	}
	return datafile
}

func openOASFile(log logging.Logger, path string) *os.File {
	oasfile, err := os.Open(path)
	if err != nil {
		log.Infof("failed to open the OpenAPI specification file %s.", path)
		return nil
	}
	return oasfile
}

var datasetFileName string
var openApiSpecFileName string

func main() {
	serviceName := "api-opendata"

	log := logging.NewLogger()
	log.Infof("Starting up %s ...", serviceName)

	flag.StringVar(&openApiSpecFileName, "oas", "", "An OpenAPI specification to be served on /api/openapi")
	flag.StringVar(&datasetFileName, "rdffile", "", "The file to serve datasets from")
	flag.Parse()

	datafile := openDatasetsFile(log, datasetFileName)
	oasfile := openOASFile(log, openApiSpecFileName)

	if datafile == nil {
		log.Fatal("Unable to open dataset file. Exiting.")
	} else {
		datasetResponseBuffer := bytes.NewBuffer(nil)
		written, err := io.Copy(datasetResponseBuffer, datafile)
		defer datafile.Close()

		if err != nil {
			log.Fatal("unable to copy datasets file into response buffer: %s", err.Error())
		}

		log.Infof("copied %d bytes from %s into datasets response buffer.", written, datasetFileName)

		var oasResponseBuffer *bytes.Buffer
		if oasfile != nil {
			defer oasfile.Close()
			oasResponseBuffer = bytes.NewBuffer(nil)
			written, err := io.Copy(oasResponseBuffer, oasfile)
			if err != nil {
				log.Errorf("failed to copy OpenAPI specification into response buffer: %s", err.Error())
			} else {
				log.Infof("copied %d bytes from %s into openapi response buffer.", written, openApiSpecFileName)
			}
		}

		db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(), log)
		application.CreateRouterAndStartServing(log, db, datasetResponseBuffer, oasResponseBuffer)
	}
}
