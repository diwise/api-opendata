package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"

	"github.com/diwise/api-opendata/internal/pkg/application"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/go-chi/chi"
)

func openDatasetsFile(ctx context.Context, path string) *os.File {
	log := logging.GetFromContext(ctx)
	datafile, err := os.Open(path)
	if err != nil {
		log.Info().Msgf("failed to open the datasets rdf file %s.", path)
		return nil
	}
	return datafile
}

func openOASFile(ctx context.Context, path string) *os.File {
	log := logging.GetFromContext(ctx)
	oasfile, err := os.Open(path)
	if err != nil {
		log.Info().Msgf("failed to open the OpenAPI specification file %s.", path)
		return nil
	}
	return oasfile
}

var datasetFileName string
var openApiSpecFileName string

func main() {
	serviceName := "api-opendata"
	serviceVersion := buildinfo.SourceVersion()

	ctx, log, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	log.Info().Msgf("Starting up %s ...", serviceName)

	flag.StringVar(&openApiSpecFileName, "oas", "/opt/diwise/openapi.json", "An OpenAPI specification to be served on /api/openapi")
	flag.StringVar(&datasetFileName, "rdffile", "/opt/diwise/datasets/dcat.rdf", "The file to serve datasets from")
	flag.Parse()

	datafile := openDatasetsFile(ctx, datasetFileName)
	oasfile := openOASFile(ctx, openApiSpecFileName)

	if datafile == nil {
		log.Fatal().Msg("Unable to open dataset file. Exiting.")
	} else {
		datasetResponseBuffer := bytes.NewBuffer(nil)
		written, err := io.Copy(datasetResponseBuffer, datafile)
		defer datafile.Close()

		if err != nil {
			log.Fatal().Err(err).Msg("unable to copy datasets file into response buffer")
		}

		log.Info().Msgf("copied %d bytes from %s into datasets response buffer.", written, datasetFileName)

		var oasResponseBuffer *bytes.Buffer
		if oasfile != nil {
			defer oasfile.Close()
			oasResponseBuffer = bytes.NewBuffer(nil)
			written, err := io.Copy(oasResponseBuffer, oasfile)
			if err != nil {
				log.Error().Err(err).Msgf("failed to copy OpenAPI specification into response buffer")
			} else {
				log.Info().Msgf("copied %d bytes from %s into openapi response buffer.", written, openApiSpecFileName)
			}
		}

		port := os.Getenv("SERVICE_PORT")
		if port == "" {
			port = "8880"
		}

		r := chi.NewRouter()

		db, err := database.NewDatabaseConnection(database.NewSQLiteConnector(), ctx)
		if err != nil {
			log.Fatal().Msgf("failed to connect to database, shutting down... %s", err.Error())
		}
		app := application.NewAPI(r, db, ctx, datasetResponseBuffer, oasResponseBuffer)
		err = app.Start(port)
		if err != nil {
			log.Fatal().Msgf("failed to start router: %s", err.Error())
		}
	}
}
