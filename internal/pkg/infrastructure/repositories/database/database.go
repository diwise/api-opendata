package database

import (
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/api-opendata/internal/pkg/infrastructure/repositories/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//Datastore is an interface that is used to inject the database into different handlers to improve testability
type Datastore interface {
	CreateCatalog() (*persistence.Catalog, error)
	GetAllCatalogs() ([]persistence.Catalog, error)
	GetDatasetFromPrimaryKey(id uint) (*persistence.Dataset, error)
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
	)

	db.impl.Model(&persistence.Catalog{}).Association("Dataset")

	return db, nil
}

func (db *myDB) CreateCatalog() (*persistence.Catalog, error) {

	dataset := persistence.Dataset{
		Title:       "dataset",
		Description: "description",
		Publisher:   "publisher",
	}

	catalog := &persistence.Catalog{
		CatalogID:   "BadTemperatur01",
		Title:       "BadTemperaturer",
		Description: "En katalog med badtemperaturer",
		Publisher:   "srcPublisher",
		License:     "srcLicense",
		Dataset:     []persistence.Dataset{dataset},
	}

	result := db.impl.Create(catalog)
	if result.Error != nil {
		return nil, result.Error
	}

	return nil, nil
}

func (db *myDB) GetAllCatalogs() ([]persistence.Catalog, error) {
	catalogs := []persistence.Catalog{}
	result := db.impl.Find(&catalogs)
	if result.Error != nil {
		return nil, result.Error
	}

	return catalogs, nil
}

func (db *myDB) GetDatasetFromPrimaryKey(id uint) (*persistence.Dataset, error) {

	dataset := &persistence.Dataset{}
	result := db.impl.Find(&dataset, id)
	if result.Error != nil {
		return nil, result.Error
	}

	return dataset, nil
}
