package application

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/datasets"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestThatRetrieveCatalogsSucceeds(t *testing.T) {
	log := logging.NewLogger()
	db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(), log)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost:8080/catalogs", nil)

	NewRetrieveCatalogsHandler(log, db).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", w.Code)
	}

}

func TestGetBeaches(t *testing.T) {

	log := logging.NewLogger()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/beaches", nil)

	datasets.NewRetrieveBeachesHandler(log, "diwise.io").ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", w.Code)
	}

	fmt.Println(w.Body.String())
}
