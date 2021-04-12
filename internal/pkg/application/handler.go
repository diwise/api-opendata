package application

import (
	"compress/flate"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/repositories/database"

	"github.com/rs/cors"
)

//RequestRouter needs a comment
type RequestRouter struct {
	impl *chi.Mux
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

//CreateRouterAndStartServing sets up therouter and starts serving incoming requests
func CreateRouterAndStartServing(log logging.Logger, db database.Datastore) {

	router := &RequestRouter{impl: chi.NewRouter()}

	router.impl.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	// Enable gzip compression for ngsi-ld responses
	compressor := middleware.NewCompressor(flate.DefaultCompression, "application/xml", "application/rdf+xml")
	router.impl.Use(compressor.Handler)
	router.impl.Use(middleware.Logger)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8880"
	}

	log.Infof("Starting api-opendata on port %s.\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router.impl))

	router.RetrieveCatalogs(log, db)

}

func (router *RequestRouter) CreateNewCatalog() {
	//post request that should create new catalog to database.
}

func (router *RequestRouter) RetrieveCatalogs(log logging.Logger, db database.Datastore) error {
	var err error

	//get request should get all catalogs from database and return in rdf/xml format.

	// create empty array of rdfCatalogs
	// create a for loop to fill up the array - for each catalog in catalogs (decode into rdfCatalog)
	// fetch persistence.Catalogs from database to populate empty array of rdfCatalog
	// encode?
	// return xml?

	return err
}
