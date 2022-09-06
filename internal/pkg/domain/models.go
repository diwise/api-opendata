package domain

import "time"

// Catalog ..
type Catalog struct {
	About       string
	Title       string
	Description string
	Agent       Agent
	License     string
	Dataset     Dataset
}

// Dataset ...
type Dataset struct {
	About        string
	Title        string
	Description  string
	Publisher    Agent
	Distribution string //Distribution
	Organization string //Organization
}

// Distribution ...
type Distribution struct {
	About       string
	AccessUrl   string
	DataService string //DataService
}

// DataService ...
type DataService struct {
	About       string
	Title       string
	EndpointURL string
}

// Agent ...
type Agent struct {
	About string
	Name  string
}

type Organization struct {
	About    string
	Fn       string
	HasEmail string
}

type Beach struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Location     Point         `json:"location"`
	WaterQuality *WaterQuality `json:"waterquality,omitempty"`
}

type BeachDetails struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	Location     Point           `json:"location"`
	WaterQuality *[]WaterQuality `json:"waterquality,omitempty"`
	SeeAlso      *[]string       `json:"seeAlso,omitempty"`
}

type ExerciseTrail struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Location   Point    `json:"location"`
	Categories []string `json:"categories"`
	Length     float64  `json:"length"`
	Status     string   `json:"status"`
}

type ExerciseTrailDetails struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Location    LineString `json:"location"`
	Categories  []string   `json:"categories"`
	Length      float64    `json:"length"`
	Status      string     `json:"status"`
	Source      string     `json:"source"`
	AreaServed  string     `json:"areaServed"`
}

type Sensor struct {
	Id           string
	Temperatures []Temperature
}

type Temperature struct {
	Id      string
	Average *float64
	Max     *float64
	Min     *float64
	Value   *float64
	When    *time.Time
	From    *time.Time
	To      *time.Time
}

type WaterQuality struct {
	Temperature  float64 `json:"temperature"`
	DateObserved string  `json:"dateObserved"`
	Source       *string `json:"source,omitempty"`
}

func (wq WaterQuality) Age() time.Duration {
	observedAt, err := time.Parse(time.RFC3339, wq.DateObserved)
	if err != nil {
		// Pretend it was almost 100 years ago
		return 100 * 365 * 24 * time.Hour
	}

	return time.Since(observedAt)
}

type Cityworks struct {
	ID          string   `json:"id"`
	Location    Point    `json:"location"`
	DateCreated DateTime `json:"dateCreated"`
}

type CityworksDetails struct {
	ID           string   `json:"id"`
	Location     Point    `json:"location"`
	Description  string   `json:"description"`
	DateCreated  DateTime `json:"dateCreated"`
	DateModified DateTime `json:"dateModified"`
	StartDate    DateTime `json:"startDate"`
	EndDate      DateTime `json:"endDate"`
}

type DateTime struct {
	Type  string `json:"@type"`
	Value string `json:"@value"`
}

type Point struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

func NewPoint(latitude, longitude float64) *Point {
	return &Point{"Point", []float64{longitude, latitude}}
}

type LineString struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

func NewLineString(coordinates [][]float64) *LineString {
	return &LineString{"LineString", coordinates}
}
