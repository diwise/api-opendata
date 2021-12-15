package application

import (
	"bytes"
	"compress/flate"
	"encoding/xml"
	"net/http"
	"os"

	"github.com/diwise/api-opendata/internal/pkg/application/datasets"
	"github.com/diwise/api-opendata/internal/pkg/application/services/temperature"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/rs/cors"
)

type Application interface {
	Start(port string) error
}

type opendataApp struct {
	router chi.Router
	db     database.Datastore
	log    logging.Logger
}

func NewApplication(r chi.Router, db database.Datastore, log logging.Logger, dcatResponse *bytes.Buffer, openapiResponse *bytes.Buffer) Application {
	return newOpendataApp(r, db, log, dcatResponse, openapiResponse)
}

func newOpendataApp(r chi.Router, db database.Datastore, log logging.Logger, dcatResponse *bytes.Buffer, openapiResponse *bytes.Buffer) *opendataApp {
	o := &opendataApp{
		router: r,
		db:     db,
		log:    log,
	}

	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	// Enable gzip compression for our responses
	compressor := middleware.NewCompressor(
		flate.DefaultCompression,
		"text/csv", "application/json", "application/xml", "application/rdf+xml",
	)
	r.Use(compressor.Handler)
	r.Use(middleware.Logger)

	o.addDiwiseHandlers(r, log, db)
	o.addProbeHandlers(r)

	r.Get("/api/datasets/dcat", o.newRetrieveDatasetsHandler(log, dcatResponse))
	r.Get("/api/openapi", o.newRetrieveOpenAPIHandler(log, openapiResponse))

	return o
}

func (a *opendataApp) Start(port string) error {
	a.log.Infof("Starting api-opendata on port:%s", port)
	return http.ListenAndServe(":"+port, a.router)
}

func (o *opendataApp) addDiwiseHandlers(r chi.Router, log logging.Logger, db database.Datastore) {
	contextBrokerURL := os.Getenv("DIWISE_CONTEXT_BROKER_URL")
	waterQualityQueryParams := os.Getenv("WATER_QUALITY_QUERY_PARAMS")
	stratsysCompanyCode := os.Getenv("STRATSYS_COMPANY_CODE")
	stratsysClientId := os.Getenv("STRATSYS_CLIENT_ID")
	stratsysScope := os.Getenv("STRATSYS_SCOPE")
	stratsysLoginUrl := os.Getenv("STRATSYS_LOGIN_URL")
	stratsysDefaultUrl := os.Getenv("STRATSYS_DEFAULT_URL")

	//r.Get("/catalogs", o.catalogsHandler())
	r.Get(
		"/api/temperature/water",
		datasets.NewRetrieveWaterQualityHandler(log, contextBrokerURL, waterQualityQueryParams),
	)
	r.Get(
		"/api/beaches",
		datasets.NewRetrieveBeachesHandler(log, contextBrokerURL),
	)
	r.Get(
		"/api/temperature/air",
		datasets.NewRetrieveTemperaturesHandler(log, temperature.NewTempService(contextBrokerURL)),
	)
	r.Get(
		"/api/temperature/air/sensors",
		datasets.NewRetrieveTemperatureSensorsHandler(log, contextBrokerURL),
	)
	r.Get(
		"/api/trafficflow",
		datasets.NewRetrieveTrafficFlowsHandler(log, contextBrokerURL),
	)
	r.Get(
		"/api/stratsys/publishedreports",
		datasets.NewRetrieveStratsysReportsHandler(log, stratsysCompanyCode, stratsysClientId, stratsysScope, stratsysLoginUrl, stratsysDefaultUrl))
	r.Get(
		"/api/stratsys/publishedreports/{id}",
		datasets.NewRetrieveStratsysReportsHandler(log, stratsysCompanyCode, stratsysClientId, stratsysScope, stratsysLoginUrl, stratsysDefaultUrl))
}

func (o *opendataApp) addProbeHandlers(r chi.Router) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (o *opendataApp) catalogsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		catalogs, err := o.db.GetAllCatalogs()
		if err != nil {
			o.log.Errorf("something went wrong when trying to get all Catalogs: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, catalog := range catalogs {

			dataService, _ := o.db.GetDataServiceFromPrimaryKey(catalog.ID)
			dcatDataService := RdfDataService{
				Attr_rdf_about: dataService.About,
			}
			dcatDataService.Dcterms_title.XMLLang = "sv"
			dcatDataService.Dcterms_title.Title = dataService.Title
			dcatDataService.Dcat_endpointURL.Attr_rdf_resource = dataService.EndpointURL

			agent, _ := o.db.GetAgentFromPrimaryKey(catalog.ID)
			foafAbout := RdfAgent{
				Attr_rdf_about: agent.About,
				Foaf_name:      agent.Name,
			}

			distribution, _ := o.db.GetDistributionFromPrimaryKey(catalog.ID)
			dcatDist := RdfDistribution{
				Attr_rdf_about: distribution.About,
			}
			dcatDist.Dcat_accessURL.Attr_rdf_resource = dcatDataService.Dcat_endpointURL.Attr_rdf_resource
			//dcatDist.Dcat_accessService.Attr_rdf_resource = distribution.AccessService

			org, _ := o.db.GetOrganizationFromPrimaryKey(catalog.ID)
			rdfOrg := RdfOrganization{
				Attr_rdf_about: org.About,
				Vcard_Fn:       org.Fn,
			}
			rdfOrg.Vcard_hasEmail.Attr_rdf_resource = org.HasEmail

			dataset, _ := o.db.GetDatasetFromPrimaryKey(catalog.ID)
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

func (o *opendataApp) newRetrieveDatasetsHandler(log logging.Logger, dcatResponse *bytes.Buffer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/rdf+xml")
		w.Write(dcatResponse.Bytes())
	})
}

func (o *opendataApp) newRetrieveOpenAPIHandler(log logging.Logger, openapiResponse *bytes.Buffer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if openapiResponse == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(openapiResponse.Bytes())
	})
}
