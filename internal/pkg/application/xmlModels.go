package application

import "encoding/xml"

type RdfCatalog struct {
	XMLName     xml.Name `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# RDF,omitempty"`
	CatalogID   string   `xml:"http://diwise.io/catalogID"`
	Title       string   `xml:"http://diwise.io/title"`
	Description string   `xml:"http://diwise.io/description"`
	Publisher   string   `xml:"http://diwise.io/publisher"`
	License     string   `xml:"http://diwise.io/license"`
	Dataset     Dataset
}

type Dataset struct {
	Title       string `xml:"datasetTitle"`
	Description string `xml:"datasetDescription"`
	Publisher   string `xml:"datasetPublisher"`
}
