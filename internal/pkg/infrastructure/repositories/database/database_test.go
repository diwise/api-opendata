package database_test

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application"
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

func TestThatAllThingsCanBeRetrievedFromDatabase(t *testing.T) {
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

		dataService, _ := db.GetDataServiceFromPrimaryKey(catalog.ID)
		dcatDataService := application.RdfDataService{
			Attr_rdf_about: dataService.About,
		}
		dcatDataService.Dcterms_title.XMLLang = "sv"
		dcatDataService.Dcterms_title.Title = dataService.Title
		dcatDataService.Dcat_endpointURL.Attr_rdf_resource = dataService.EndpointURL

		agent, _ := db.GetAgentFromPrimaryKey(catalog.ID)
		foafAbout := application.RdfAgent{
			Attr_rdf_about: agent.About,
			Foaf_name:      agent.Name,
		}

		distribution, _ := db.GetDistributionFromPrimaryKey(catalog.ID)
		dcatDist := application.RdfDistribution{
			Attr_rdf_about: distribution.About,
		}
		dcatDist.Dcat_accessURL.Attr_rdf_resource = dcatDataService.Dcat_endpointURL.Attr_rdf_resource
		dcatDist.Dcat_accessService.Attr_rdf_resource = distribution.AccessService

		org, _ := db.GetOrganizationFromPrimaryKey(catalog.ID)
		rdfOrg := application.RdfOrganization{
			Attr_rdf_about: org.About,
			Vcard_Fn:       org.Fn,
		}
		rdfOrg.Vcard_hasEmail.Attr_rdf_resource = org.HasEmail

		dataset, _ := db.GetDatasetFromPrimaryKey(catalog.ID)
		rdfDataset := application.RdfDataset{
			Attr_rdf_about: dataset.About,
		}
		rdfDataset.Dcterms_title.Title = dataset.Title
		rdfDataset.Dcterms_title.XMLLang = "sv"
		rdfDataset.Dcterms_description.Description = dataset.Description
		rdfDataset.Dcterms_publisher.Attr_rdf_resource = agent.About
		rdfDataset.Dcat_distribution.Attr_rdf_resource = dcatDist.Attr_rdf_about
		rdfDataset.Dcat_contactPoint.Attr_rdf_resource = org.About

		rdfCatalog := application.RdfCatalog{
			Attr_rdf_about: catalog.About,
		}

		rdfCatalog.Dcterms_title.XMLLang = "sv"
		rdfCatalog.Dcterms_title.Title = catalog.Title
		rdfCatalog.Dcterms_description.XMLLang = "sv"
		rdfCatalog.Dcterms_description.Description = catalog.Description
		rdfCatalog.Dcterms_publisher.Attr_rdf_resource = agent.About
		rdfCatalog.Dcat_dataset.Attr_rdf_resource = dataset.About

		rdf := application.Rdf_RDF{
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
		xmlString := string(data)
		fmt.Println(xmlString)

		/* if stringData != expectedXML {
			t.Errorf("data does not match expected xml output: %s != %s", stringData, expectedXML)
		} */
	}

	if len(catalogs) < 1 {
		t.Error("no catalogs found.")
	}

	fmt.Print("catalogs: ", catalogs[0])
}
