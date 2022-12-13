package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"

	presentation "github.com/diwise/api-opendata/internal/pkg/presentation"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/go-chi/chi/v5"
)

func openFile(ctx context.Context, description, path string) *os.File {
	file, err := os.Open(path)
	if err != nil {
		log := logging.GetFromContext(ctx)
		log.Error().Err(err).Msgf("failed to open the %s file %s.", description, path)
		return nil
	}
	return file
}

func openDatasetsFile(ctx context.Context, path string) *os.File {
	return openFile(ctx, "datasets rdf", path)
}

func openOASFile(ctx context.Context, path string) *os.File {
	return openFile(ctx, "OpenAPI specification", path)
}

func openOrganisationsFile(ctx context.Context, path string) *os.File {
	if path == "" {
		return nil
	}

	return openFile(ctx, "organisations registry", path)
}

var datasetFileName string
var openApiSpecFileName string
var organisationRegistryFile string

func main() {
	serviceName := "api-opendata"
	serviceVersion := buildinfo.SourceVersion()

	ctx, log, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	flag.StringVar(&openApiSpecFileName, "oas", "/opt/diwise/openapi.json", "An OpenAPI specification to be served on /api/openapi")
	flag.StringVar(&organisationRegistryFile, "orgreg", "", "A yaml file containing known organisations")
	flag.StringVar(&datasetFileName, "rdffile", "/opt/diwise/datasets/dcat.rdf", "The file to serve datasets from")
	flag.Parse()

	datafile := openDatasetsFile(ctx, datasetFileName)
	oasfile := openOASFile(ctx, openApiSpecFileName)
	orgFile := openOrganisationsFile(ctx, organisationRegistryFile)

	if datafile == nil {
		log.Fatal().Msg("Unable to open dataset file. Exiting.")
	} else {
		defer datafile.Close()

		datasetResponseBuffer := bytes.NewBuffer(nil)
		written, err := io.Copy(datasetResponseBuffer, datafile)

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
				log.Error().Err(err).Msg("failed to copy OpenAPI specification into response buffer")
			} else {
				log.Info().Msgf("copied %d bytes from %s into openapi response buffer.", written, openApiSpecFileName)
			}
		}

		port := env.GetVariableOrDefault(log, "SERVICE_PORT", "8080")

		r := chi.NewRouter()

		api := presentation.NewAPI(r, ctx, datasetResponseBuffer, oasResponseBuffer, orgFile)
		err = api.Start(port)
		if err != nil {
			log.Fatal().Msgf("failed to start router: %s", err.Error())
		}
	}
}
