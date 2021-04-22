package database

import (
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//Datastore is an interface that is used to inject the database into different handlers to improve testability
type Datastore interface {
	CreateCatalog(catalog domain.Catalog) (*persistence.Catalog, error)
	CreateAgent() (*persistence.Agent, error)
	CreateDataService() (*persistence.DataService, error)
	CreateDataset(catalog persistence.Catalog) (*persistence.Dataset, error)
	CreateDistribution() (*persistence.Distribution, error)
	CreateOrganization() (*persistence.Organization, error)
	GetAllCatalogs() ([]persistence.Catalog, error)
	GetAgentFromPrimaryKey(id uint) (*persistence.Agent, error)
	GetDataServiceFromPrimaryKey(id uint) (*persistence.DataService, error)
	GetDatasetFromPrimaryKey(id uint) (*persistence.Dataset, error)
	GetDistributionFromPrimaryKey(id uint) (*persistence.Distribution, error)
	GetOrganizationFromPrimaryKey(id uint) (*persistence.Organization, error)
}

type myDB struct {
	impl *gorm.DB
}

//ConnectorFunc is used to inject a database connection method into NewDatabaseConnection
type ConnectorFunc func() (*gorm.DB, error)

//NewSQLiteConnector opens a connection to a local sqlite database
func NewSQLiteConnector() ConnectorFunc {
	return func() (*gorm.DB, error) {
		db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})

		if err == nil {
			db.Exec("PRAGMA foreign_keys = ON")
		}

		return db, err
	}
}

//NewDatabaseConnection initializes a new connection to the database and wraps it in a Datastore
func NewDatabaseConnection(connect ConnectorFunc, log logging.Logger) (Datastore, error) {
	impl, err := connect()
	if err != nil {
		return nil, err
	}

	db := &myDB{
		impl: impl.Debug(),
	}

	db.impl.AutoMigrate(
		&persistence.Catalog{},
		&persistence.Dataset{},
		&persistence.Distribution{},
		&persistence.DataService{},
		&persistence.Agent{},
		&persistence.Organization{},
	)

	err = db.impl.Model(&persistence.Catalog{}).Association("Dataset").Error
	if err != nil {
		return nil, err
	}

	err = db.impl.Model(&persistence.Catalog{}).Association("Agent").Error
	if err != nil {
		return nil, err
	}

	err = db.impl.Model(&persistence.Dataset{}).Association("Agent").Error
	if err != nil {
		return nil, err
	}

	/* err = db.impl.Model(&persistence.Dataset{}).Association("Distribution").Error
	if err != nil {
		return nil, err
	}

	err = db.impl.Model(&persistence.Dataset{}).Association("Organization").Error
	if err != nil {
		return nil, err
	} */

	return db, nil
}

func (db *myDB) CreateCatalog(catalog domain.Catalog) (*persistence.Catalog, error) {

	agent := persistence.Agent{
		About: "aboutAgent",
		Name:  "nameAgent",
	}

	dataset := persistence.Dataset{
		About: "",
	}

	newCatalog := &persistence.Catalog{
		About:       catalog.About,
		Title:       catalog.Title,
		Description: catalog.Description,
		Agent:       agent,
		License:     catalog.License,
		Dataset:     dataset,
	}

	result := db.impl.Create(newCatalog)
	if result.Error != nil {
		return nil, result.Error
	}

	return nil, nil
}

func (db *myDB) CreateAgent() (*persistence.Agent, error) {
	agent := &persistence.Agent{
		Name:  "Diwise såklart",
		About: "http://diwise.io/publisher",
	}

	result := db.impl.Create(agent)
	if result.Error != nil {
		return nil, result.Error
	}

	return agent, nil
}

func (db *myDB) CreateDataService() (*persistence.DataService, error) {
	dataservice := &persistence.DataService{
		Title:       "dataservice title",
		About:       "http://diwise.io/dataservice",
		EndpointURL: "http://diwise.io/api",
	}

	result := db.impl.Create(dataservice)
	if result.Error != nil {
		return nil, result.Error
	}

	return dataservice, nil
}

func (db *myDB) CreateDataset(catalog persistence.Catalog) (*persistence.Dataset, error) {
	dataset := &persistence.Dataset{
		CatalogID:   catalog.ID,
		About:       "http://diwise.io/dataset1",
		Title:       "Dataset",
		Description: "dataset description",
		// Publisher:   "publisher1",
	}

	result := db.impl.Create(dataset)
	if result.Error != nil {
		return nil, result.Error
	}

	return dataset, nil
}

func (db *myDB) CreateDistribution() (*persistence.Distribution, error) {
	distribution := &persistence.Distribution{
		About:       "http://diwise.io/distribution1",
		AccessUrl:   "",
		DataService: "http://diwise.io/dataservice",
	}

	result := db.impl.Create(distribution)
	if result.Error != nil {
		return nil, result.Error
	}

	return distribution, nil
}

func (db *myDB) CreateOrganization() (*persistence.Organization, error) {
	organization := &persistence.Organization{
		About:    "https://diwise.io/contactpoint1",
		Fn:       "En organization",
		HasEmail: "mailto:nomailpls@diwise.io",
	}

	result := db.impl.Create(organization)
	if result.Error != nil {
		return nil, result.Error
	}

	return organization, nil
}

func (db *myDB) GetAllCatalogs() ([]persistence.Catalog, error) {
	catalogs := []persistence.Catalog{}
	result := db.impl.Find(&catalogs)
	if result.Error != nil {
		return nil, result.Error
	}

	return catalogs, nil
}

func (db *myDB) GetAgentFromPrimaryKey(id uint) (*persistence.Agent, error) {

	agent := &persistence.Agent{}
	result := db.impl.Find(&agent, id)
	if result.Error != nil {
		return nil, result.Error
	}

	return agent, nil
}

func (db *myDB) GetDataServiceFromPrimaryKey(id uint) (*persistence.DataService, error) {

	dataservice := &persistence.DataService{}
	result := db.impl.Find(&dataservice, id)
	if result.Error != nil {
		return nil, result.Error
	}

	return dataservice, nil
}

func (db *myDB) GetDatasetFromPrimaryKey(id uint) (*persistence.Dataset, error) {

	dataset := &persistence.Dataset{}
	result := db.impl.Find(&dataset, id)
	if result.Error != nil {
		return nil, result.Error
	}

	return dataset, nil
}

func (db *myDB) GetDistributionFromPrimaryKey(id uint) (*persistence.Distribution, error) {

	distribution := &persistence.Distribution{}
	result := db.impl.Find(&distribution, id)
	if result.Error != nil {
		return nil, result.Error
	}

	return distribution, nil
}

func (db *myDB) GetOrganizationFromPrimaryKey(id uint) (*persistence.Organization, error) {

	organization := &persistence.Organization{}
	result := db.impl.Find(&organization, id)
	if result.Error != nil {
		return nil, result.Error
	}

	return organization, nil
}
