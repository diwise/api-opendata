package application

import (
	"bytes"
	"compress/flate"
	"encoding/xml"
	"net/http"
	"os"

	"github.com/diwise/api-opendata/internal/pkg/application/datasets"
	"github.com/diwise/api-opendata/internal/pkg/application/services"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/rs/cors"
)

//RequestRouter needs a comment
type RequestRouter struct {
	impl *chi.Mux
}

func (router *RequestRouter) addDiwiseHandlers(log logging.Logger, db database.Datastore) {
	contextBrokerURL := os.Getenv("DIWISE_CONTEXT_BROKER_URL")
	waterQualityQueryParams := os.Getenv("WATER_QUALITY_QUERY_PARAMS")

	//router.Get("/catalogs/", NewRetrieveCatalogsHandler(log, db))
	router.Get(
		"/api/temperature/water",
		datasets.NewRetrieveWaterQualityHandler(log, contextBrokerURL, waterQualityQueryParams),
	)
	router.Get(
		"/api/beaches",
		datasets.NewRetrieveBeachesHandler(log, contextBrokerURL),
	)
	router.Get(
		"/api/temperature/air",
		datasets.NewRetrieveTemperaturesHandler(log, services.NewTempService(contextBrokerURL)),
	)
	router.Get(
		"/api/trafficflow",
		datasets.NewRetrieveTrafficFlowsHandler(log, contextBrokerURL),
	)
}

func (router *RequestRouter) addProbeHandlers() {
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

//Get accepts a pattern that should be routed to the handlerFn on a GET request
func (router *RequestRouter) Get(pattern string, handlerFn http.HandlerFunc) {
	router.impl.Get(pattern, handlerFn)
}

//Patch accepts a pattern that should be routed to the handlerFn on a PATCH request
func (router *RequestRouter) Patch(pattern string, handlerFn http.HandlerFunc) {
	router.impl.Patch(pattern, handlerFn)
}

//Post accepts a pattern that should be routed to the handlerFn on a POST request
func (router *RequestRouter) Post(pattern string, handlerFn http.HandlerFunc) {
	router.impl.Post(pattern, handlerFn)
}

//CreateRouterAndStartServing sets up the router and starts serving incoming requests
func CreateRouterAndStartServing(log logging.Logger, db database.Datastore, dcatResponse *bytes.Buffer) {

	router := &RequestRouter{impl: chi.NewRouter()}

	router.impl.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	// Enable gzip compression for our responses
	compressor := middleware.NewCompressor(
		flate.DefaultCompression,
		"text/csv", "application/json", "application/xml", "application/rdf+xml",
	)
	router.impl.Use(compressor.Handler)
	router.impl.Use(middleware.Logger)

	router.addDiwiseHandlers(log, db)
	router.addProbeHandlers()

	router.Get("/api/datasets/dcat", NewRetrieveDatasetsHandler(log, dcatResponse))

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8880"
	}

	log.Infof("Starting api-opendata on port %s.\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router.impl))
}

func NewRetrieveCatalogsHandler(log logging.Logger, db database.Datastore) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		catalogs, err := db.GetAllCatalogs()
		if err != nil {
			log.Errorf("something went wrong when trying to get all Catalogs: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, catalog := range catalogs {

			dataService, _ := db.GetDataServiceFromPrimaryKey(catalog.ID)
			dcatDataService := RdfDataService{
				Attr_rdf_about: dataService.About,
			}
			dcatDataService.Dcterms_title.XMLLang = "sv"
			dcatDataService.Dcterms_title.Title = dataService.Title
			dcatDataService.Dcat_endpointURL.Attr_rdf_resource = dataService.EndpointURL

			agent, _ := db.GetAgentFromPrimaryKey(catalog.ID)
			foafAbout := RdfAgent{
				Attr_rdf_about: agent.About,
				Foaf_name:      agent.Name,
			}

			distribution, _ := db.GetDistributionFromPrimaryKey(catalog.ID)
			dcatDist := RdfDistribution{
				Attr_rdf_about: distribution.About,
			}
			dcatDist.Dcat_accessURL.Attr_rdf_resource = dcatDataService.Dcat_endpointURL.Attr_rdf_resource
			//dcatDist.Dcat_accessService.Attr_rdf_resource = distribution.AccessService

			org, _ := db.GetOrganizationFromPrimaryKey(catalog.ID)
			rdfOrg := RdfOrganization{
				Attr_rdf_about: org.About,
				Vcard_Fn:       org.Fn,
			}
			rdfOrg.Vcard_hasEmail.Attr_rdf_resource = org.HasEmail

			dataset, _ := db.GetDatasetFromPrimaryKey(catalog.ID)
			rdfDataset := RdfDataset{
				Attr_rdf_about: dataset.About,
			}
			rdfDataset.Dcterms_title.Title = dataset.Title
			rdfDataset.Dcterms_title.XMLLang = "sv"
			rdfDataset.Dcterms_description.Description = dataset.Description
			rdfDataset.Dcterms_publisher.Attr_rdf_resource = agent.About
			rdfDataset.Dcat_distribution.Attr_rdf_resource = dcatDist.Attr_rdf_about
			rdfDataset.Dcat_contactPoint.Attr_rdf_resource = org.About

			rdfCatalog := RdfCatalog{
				Attr_rdf_about: catalog.About,
			}

			rdfCatalog.Dcterms_title.XMLLang = "sv"
			rdfCatalog.Dcterms_title.Title = catalog.Title
			rdfCatalog.Dcterms_description.XMLLang = "sv"
			rdfCatalog.Dcterms_description.Description = catalog.Description
			rdfCatalog.Dcterms_publisher.Attr_rdf_resource = agent.About
			rdfCatalog.Dcat_dataset.Attr_rdf_resource = dataset.About

			rdf := Rdf_RDF{
				Rdf_Catalog:      &rdfCatalog,
				Rdf_Dataset:      &rdfDataset,
				Rdf_Agent:        &foafAbout,
				Rdf_Distribution: &dcatDist,
				Rdf_Organization: &rdfOrg,
				Rdf_DataService:  &dcatDataService,
				Attr_rdf:         "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
				Attr_dcterms:     "http://purl.org/dc/terms/",
				Attr_vcard:       "http://www.w3.org/2006/vcard/ns#",
				Attr_dcat:        "http://www.w3.org/ns/dcat#",
				Attr_foaf:        "http://xmlns.com/foaf/0.1/",
			}

			data, _ := xml.MarshalIndent(rdf, " ", "	")
			w.Write(data)
		}

		w.WriteHeader(http.StatusOK)

	})
}

func NewRetrieveDatasetsHandler(log logging.Logger, dcatResponse *bytes.Buffer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/rdf+xml")
		w.Write(dcatResponse.Bytes())
	})
}
