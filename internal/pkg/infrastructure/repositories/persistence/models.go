package persistence

import (
	"gorm.io/gorm"
)

//Catalog ...
type Catalog struct {
	gorm.Model
	About       string
	Title       string
	Description string
	AgentID     uint
	Agent       Agent `gorm:"association_foreignkey:ID"`
	License     string
	DatasetID   uint
	Dataset     Dataset `gorm:"association_foreignkey:ID"`
}

//Dataset ...
type Dataset struct {
	gorm.Model
	CatalogID      uint
	About          string
	Title          string
	Description    string
	AgentID        uint
	Agent          Agent `gorm:"association_foreignkey:ID"`
	DistributionID uint
	Distribution   string //Distribution `gorm:"association_foreignkey:ID"`
	OrganizationID uint
	Organization   string //Organization `gorm:"association_foreignkey:ID"`
}

//Distribution ...
type Distribution struct {
	gorm.Model
	About         string
	AccessUrl     string
	DataServiceID uint   `gorm:"association_foreignkey:ID"`
	DataService   string //DataService
}

//DataService ...
type DataService struct {
	gorm.Model
	About       string
	Title       string
	EndpointURL string
}

//Agent ...
type Agent struct {
	gorm.Model
	About string
	Name  string
}

type Organization struct {
	gorm.Model
	About    string
	Fn       string
	HasEmail string
}
