package domain

//Catalog ..
type Catalog struct {
	About       string
	Title       string
	Description string
	Agent       Agent
	License     string
	Dataset     Dataset
}

//Dataset ...
type Dataset struct {
	About        string
	Title        string
	Description  string
	Publisher    Agent
	Distribution string //Distribution
	Organization string //Organization
}

//Distribution ...
type Distribution struct {
	About       string
	AccessUrl   string
	DataService string //DataService
}

//DataService ...
type DataService struct {
	About       string
	Title       string
	EndpointURL string
}

//Agent ...
type Agent struct {
	About string
	Name  string
}

type Organization struct {
	About    string
	Fn       string
	HasEmail string
}

type Temperature struct {
	Id    string
	Value float64
	When  string
}
