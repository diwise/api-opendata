package database_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/domain"
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

	agent := domain.Agent{
		About: "aboutAgent",
		Name:  "nameAgent",
	}

	/* dataservice := domain.DataService{}

	distribution := domain.Distribution{
		About:       "distibutionAbout",
		AccessUrl:   "accessUrl",
		DataService: dataservice,
	}

	organization := domain.Organization{
		About:    "organizationAbout",
		Fn:       "organizationFn",
		HasEmail: "organizationEmail",
	} */

	dataset := domain.Dataset{
		About:        "aboutDataset",
		Title:        "titleDataset",
		Description:  "descriptionString",
		Publisher:    agent,
		Distribution: "distribution",
		Organization: "organization",
	}

	catalog := domain.Catalog{
		About:       "about",
		Title:       "title",
		Description: "desc",
		Agent:       agent,
		License:     "license",
		Dataset:     dataset,
	}

	_, err = db.CreateCatalog(catalog)
	if err != nil {
		t.Error("something went wrong when trying to create new Catalog")
	}
}

func TestThatAllThingsCanBeRetrievedFromDatabase(t *testing.T) {
	db, err := db.NewDatabaseConnection(db.NewSQLiteConnector(), &log.Logger{})
	if err != nil {
		t.Errorf("could not connect to database: %s", err)
	}

	agent := domain.Agent{
		About: "aboutAgent",
		Name:  "nameAgent",
	}

	/* dataservice := domain.DataService{}

	distribution := domain.Distribution{
		About:       "distibutionAbout",
		AccessUrl:   "accessUrl",
		DataService: dataservice,
	}

	organization := domain.Organization{
		About:    "organizationAbout",
		Fn:       "organizationFn",
		HasEmail: "organizationEmail",
	}
	*/

	dataset := domain.Dataset{
		About:        "aboutDataset",
		Title:        "titleDataset",
		Description:  "descriptionString",
		Publisher:    agent,
		Distribution: "distribution",
		Organization: "organization",
	}

	catalog := domain.Catalog{
		About:       "about",
		Title:       "title",
		Description: "desc",
		Agent:       agent,
		License:     "license",
		Dataset:     dataset,
	}

	_, err = db.CreateCatalog(catalog)
	if err != nil {
		t.Errorf("something went wrong when trying to create new Catalog: %s", err.Error())
		return
	}

	catalogs, err := db.GetAllCatalogs()
	if err != nil {
		t.Error("something went wrong when trying to get all Catalogs")
		return
	}

	if len(catalogs) < 1 {
		t.Error("no catalogs found.")
	}

	fmt.Print("catalogs: ", catalogs[0])
}
