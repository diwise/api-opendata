package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
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
		log.Error("failed to open file", slog.String("description", description), slog.String("path", path), slog.String("error", err.Error()))
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

const serviceName string = "api-opendata"

var datasetFileName string
var openApiSpecFileName string
var organisationRegistryFile string

func main() {
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
		log.Error("Unable to open dataset file. Exiting.")
		os.Exit(1)
	} else {
		defer datafile.Close()

		datasetResponseBuffer := bytes.NewBuffer(nil)
		written, err := io.Copy(datasetResponseBuffer, datafile)

		if err != nil {
			log.Error("unable to copy datasets file into response buffer", slog.String("error", err.Error()))
			os.Exit(1)
		}

		log.Info(fmt.Sprintf("copied %d bytes from %s into datasets response buffer.", written, datasetFileName))

		var oasResponseBuffer *bytes.Buffer
		if oasfile != nil {
			defer oasfile.Close()

			oasResponseBuffer = bytes.NewBuffer(nil)
			written, err := io.Copy(oasResponseBuffer, oasfile)

			if err != nil {
				log.Error("failed to copy OpenAPI specification into response buffer", slog.String("error", err.Error()))
			} else {
				log.Info(fmt.Sprintf("copied %d bytes from %s into openapi response buffer.", written, openApiSpecFileName))
			}
		}

		r := chi.NewRouter()

		var reader io.Reader = orgFile
		if orgFile == nil {
			reader = bytes.NewBufferString("")
		}

		api := presentation.NewAPI(ctx, r, datasetResponseBuffer, oasResponseBuffer, reader)

		port := env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")
		err = api.Start(ctx, port)
		if err != nil {
			log.Error("failed to start router", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
}
