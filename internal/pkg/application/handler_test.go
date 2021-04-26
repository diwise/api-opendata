package application

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	csvData := bytes.NewBufferString("place_id;name;latitude;longitude;description;address;postalcode;city;facilities;wc;shower;changing_room;lifeguard;lifebuoy;trash;firstaid;grilling_area;bathing_jetty;bathing_ladder;diving_tower;td_url;accessibility;wheelchair;public_transit;public_transit_distance;cycle_track;parking;parking_cost;water;beach_sand;beach_stone;beach_rock;beach_concrete;beach_grass;pet_bath;camping;temp_url;extra_url;visit_url;owner;phone;email\nSE0441273000000001;Vesljungasjön;56.4212500165633;13.7674095026078;Vesljungasjön, mellan Visseltofta och Emmaljunga, är en riktigt pärla. Här finns en vacker sandstrand på över hundra meter och en lång, fin brygga. Även här är det långgrunt.;;28022;Osby;;N;;;;Y;Y;;Y;Y;Y;N;;Gångavstånd från parkering cirka 50 meter.;;;;;Y;N;Lake;Y;N;N;N;N;N;N;;https://www.havochvatten.se/badplatser-och-badvatten/kommuner-och-badplatser/kommuner/badplatser-i-osby-kommun.html;https://www.osby.se/se--gora/upplev-osby-kommun/bada.html;K;0479123456;samhallsbyggnad@osby.se")

	log := logging.NewLogger()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/beaches", nil)

	GetBeaches(log, csvData).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", w.Code)
	}
}
