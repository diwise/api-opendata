package database_test

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"

	db "github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	log.SetFormatter(&log.JSONFormatter{})
	os.Exit(m.Run())
}

func TestDatabaseConnection(t *testing.T) {
	db, err := db.NewDatabaseConnection(db.NewSQLiteConnector(), &log.Logger{})
	if err != nil {
		t.Errorf("could not connect to database: %s", err)
	}

	_, err = db.CreateCatalog()
	if err != nil {
		t.Error("something went wrong when trying to create new Catalog")
	}
}

func TestGetAllCatalogs(t *testing.T) {
	db, err := db.NewDatabaseConnection(db.NewSQLiteConnector(), &log.Logger{})
	if err != nil {
		t.Errorf("could not connect to database: %s", err)
	}

	_, err = db.CreateCatalog()
	if err != nil {
		t.Error("something went wrong when trying to create new Catalog")
		return
	}

	catalogs, err := db.GetAllCatalogs()
	if err != nil {
		t.Error("something went wrong when trying to get all Catalogs")
		return
	}

	for _, catalog := range catalogs {
		data, _ := xml.MarshalIndent(catalog, " ", "	")

		stringData := string(data)
		if stringData != expectedXML {
			t.Errorf("data does not match expected xml output: %s != %s", stringData, expectedXML)
		}
	}

	if len(catalogs) < 1 {
		t.Error("no catalogs found.")
	}

	fmt.Print("catalogs: ", catalogs[0])
}

const expectedXML string = `<Catalog>
<ID>1</ID>
<CreatedAt>2021-04-12T17:17:45.81406+02:00</CreatedAt>
<UpdatedAt>2021-04-12T17:17:45.81406+02:00</UpdatedAt>
<DeletedAt>
	<Time>0001-01-01T00:00:00Z</Time>
	<Valid>false</Valid>
</DeletedAt>
<CatalogID>BadTemperatur01</CatalogID>
<Title>BadTemperaturer</Title>
<Description>En katalog med badtemperaturer</Description>
<Publisher>srcPublisher</Publisher>
<License>srcLicense</License>
</Catalog>`
