package persistence

import (
	"gorm.io/gorm"
)

//Catalog ...
type Catalog struct {
	gorm.Model
	CatalogID   string
	Title       string
	Description string
	Publisher   string
	License     string
	Dataset     []Dataset
}

//Dataset ...
type Dataset struct {
	gorm.Model
	CatalogID   uint
	Title       string
	Description string
	Publisher   string
}

//Distribution ...
type Distribution struct {
	gorm.Model
	WebAddress string
}

//DataService ...
type DataService struct {
	gorm.Model
	Title       string
	EndpointURL string
}

//Agent ...
type Agent struct {
	gorm.Model
	Name string
}

//ContactPoint ...
type ContactPoint struct {
	gorm.Model
	Kind          string
	FormattedName string
	Email         string
}
