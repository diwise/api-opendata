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
	Publisher   string
	License     string
	Dataset     string
}

//Dataset ...
type Dataset struct {
	gorm.Model
	About        string
	Title        string
	Description  string
	Publisher    string
	Distribution string
	ContactPoint string
}

//Distribution ...
type Distribution struct {
	gorm.Model
	About         string
	AccessUrl     string
	AccessService string
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
