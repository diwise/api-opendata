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

type Organisation struct {
	Name string `json:"name"`
}

type Organization struct {
	About    string
	Fn       string
	HasEmail string
}

type AirQuality struct {
	ID           string   `json:"id"`
	Location     Point    `json:"location"`
	DateObserved DateTime `json:"dateObserved"`
}

type AirQualityDetails struct {
	ID                        string   `json:"id"`
	Location                  Point    `json:"location"`
	DateObserved              DateTime `json:"dateObserved"`
	AtmosphericPressure       float64  `json:"atmosphericPressure,omitempty"`
	Temperature               float64  `json:"temperature,omitempty"`
	RelativeHumidity          float64  `json:"relativeHumidity,omitempty"`
	ParticleCount             float64  `json:"particleCount,omitempty"`
	PM1                       float64  `json:"PM1,omitempty"`
	PM4                       float64  `json:"PM4,omitempty"`
	PM10                      float64  `json:"PM10,omitempty"`
	PM25                      float64  `json:"PM25,omitempty"`
	TotalSuspendedParticulate float64  `json:"totalSuspendedParticulate,omitempty"`
	CO2                       float64  `json:"CO2,omitempty"`
	NO                        float64  `json:"NO,omitempty"`
	NO2                       float64  `json:"NO2,omitempty"`
	NOx                       float64  `json:"NOx,omitempty"`
	WindDirection             float64  `json:"windDirection,omitempty"`
	WindSpeed                 float64  `json:"windSpeed,omitempty"`
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
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	Location            LineString    `json:"location"`
	Categories          []string      `json:"categories"`
	PublicAccess        string        `json:"publicAccess"`
	Length              float64       `json:"length"`
	Difficulty          float64       `json:"difficulty"`
	PaymentRequired     bool          `json:"paymentRequired"`
	Status              string        `json:"status"`
	DateLastPreparation string        `json:"dateLastPreparation,omitempty"`
	Source              string        `json:"source"`
	AreaServed          string        `json:"areaServed"`
	ManagedBy           *Organisation `json:"managedBy,omitempty"`
	Owner               *Organisation `json:"owner,omitempty"`
	SeeAlso             []string      `json:"seeAlso,omitempty"`
}

type MultiPolygon struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

type SportsField struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	Categories          []string      `json:"categories"`
	PublicAccess        string        `json:"publicAccess"`
	Location            MultiPolygon  `json:"location"`
	DateCreated         *string       `json:"dateCreated,omitempty"`
	DateModified        *string       `json:"dateModified,omitempty"`
	DateLastPreparation *string       `json:"dateLastPreparation,omitempty"`
	Source              string        `json:"source"`
	ManagedBy           *Organisation `json:"managedBy,omitempty"`
	Owner               *Organisation `json:"owner,omitempty"`
	Status              string        `json:"status,omitempty"`
}

type SportsVenue struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Categories   []string      `json:"categories"`
	PublicAccess string        `json:"publicAccess,omitempty"`
	Location     MultiPolygon  `json:"location"`
	DateCreated  *string       `json:"dateCreated,omitempty"`
	DateModified *string       `json:"dateModified,omitempty"`
	Source       string        `json:"source"`
	SeeAlso      []string      `json:"seeAlso,omitempty"`
	ManagedBy    *Organisation `json:"managedBy,omitempty"`
	Owner        *Organisation `json:"owner,omitempty"`
}

type Weather struct {
	ID           string      `json:"id"`
	Temperature  Temperature `json:"temperature"`
	DateObserved time.Time   `json:"dateObserved"`
	Source       *string     `json:"source,omitempty"`
	Location     *Point      `json:"location,omitempty"`
}

type Temperature struct {
	Average *float64       `json:"avg,omitempty"`
	Max     *float64       `json:"max,omitempty"`
	Min     *float64       `json:"min,omitempty"`
	Value   *float64       `json:"value,omitempty"`
	When    *time.Time     `json:"when,omitempty"`
	From    *time.Time     `json:"from,omitempty"`
	To      *time.Time     `json:"to,omitempty"`
	Values  *[]Temperature `json:"values,omitempty"`
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

type WaterQuality struct {
	ID           string  `json:"id"`
	Temperature  float64 `json:"temperature"`
	DateObserved string  `json:"dateObserved"`
	Source       *string `json:"source,omitempty"`
	Location     *Point  `json:"location,omitempty"`
}

type WaterQualityTemporal struct {
	ID          string  `json:"id"`
	Temperature []Value `json:"temperature"`
	Source      string  `json:"source,omitempty"`
	Location    *Point  `json:"location,omitempty"`
}

type Value struct {
	Value      float64 `json:"value"`
	ObservedAt string  `json:"observedAt"`
}

func (w WaterQuality) Age() time.Duration {
	observedAt, err := time.Parse(time.RFC3339, w.DateObserved)
	if err != nil {
		// Pretend it was almost 100 years ago
		return 100 * 365 * 24 * time.Hour
	}

	return time.Since(observedAt)
}
