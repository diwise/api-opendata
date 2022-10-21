package domain

import (
	"time"
)

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

type AirQuality struct {
	ID       string `json:"id"`
	Location Point  `json:"location"`
}

type AirQualityDetails struct {
	ID                        string `json:"id"`
	Location                  Point  `json:"location"`
	AtmosphericPressure       Number `json:"atmosphericPressure,omitempty"`
	Temperature               Number `json:"temperature,omitempty"`
	RelativeHumidity          Number `json:"relativeHumidity,omitempty"`
	ParticleCount             Number `json:"particleCount,omitempty"`
	PM1                       Number `json:"PM1,omitempty"`
	PM4                       Number `json:"PM4,omitempty"`
	PM10                      Number `json:"PM10,omitempty"`
	PM25                      Number `json:"PM25,omitempty"`
	TotalSuspendedParticulate Number `json:"totalSuspendedParticulate,omitempty"`
	CO2                       Number `json:"CO2,omitempty"`
	NO                        Number `json:"NO,omitempty"`
	NO2                       Number `json:"NO2,omitempty"`
	NOx                       Number `json:"NOx,omitempty"`
	Voltage                   Number `json:"voltage,omitempty"`
}

type Number struct {
	Type     string  `json:"type"`
	Value    float64 `json:"value"`
	UnitCode string  `json:"unitCode"`
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
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	Location            LineString `json:"location"`
	Categories          []string   `json:"categories"`
	Length              float64    `json:"length"`
	Difficulty          float64    `json:"difficulty"`
	PaymentRequired     bool       `json:"paymentRequired"`
	Status              string     `json:"status"`
	DateLastPreparation string     `json:"dateLastPreparation,omitempty"`
	Source              string     `json:"source"`
	AreaServed          string     `json:"areaServed"`
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
	ID        string `json:"id"`
	Location  Point  `json:"location"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type CityworksDetails struct {
	ID           string `json:"id"`
	Location     Point  `json:"location"`
	Description  string `json:"description"`
	DateModified string `json:"dateModified,omitempty"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
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

type RoadAccident struct {
	ID           string `json:"id"`
	AccidentDate string `json:"accidentDate"`
	Location     Point  `json:"location"`
}

type RoadAccidentDetails struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	Location     Point  `json:"location"`
	AccidentDate string `json:"accidentDate"`
	DateCreated  string `json:"dateCreated"`
	DateModified string `json:"dateModified,omitempty"`
	Status       string `json:"status"`
}
