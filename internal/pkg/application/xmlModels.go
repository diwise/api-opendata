package application

import "encoding/xml"

type Rdf_RDF struct {
	Attr_rdf         string           `xml:"xmlns:rdf,attr"`
	Attr_dcterms     string           `xml:"xmlns:dcterms,attr"`
	Attr_vcard       string           `xml:"xmlns:vcard,attr"`
	Attr_dcat        string           `xml:"xmlns:dcat,attr"`
	Attr_foaf        string           `xml:"xmlns:foaf,attr"`
	XMLName          xml.Name         `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# rdf:RDF,omitempty"`
	Rdf_Catalog      *RdfCatalog      `xml:"dcat:Catalog,omitempty"`
	Rdf_Agent        *RdfAgent        `xml:"foaf:Agent,omitempty"`
	Rdf_Dataset      *RdfDataset      `xml:"dcat:Dataset,omitempty"`
	Rdf_Distribution *RdfDistribution `xml:"dcat:Distribution,omitempty"`
	Rdf_Organization *RdfOrganization `xml:"dcat:Organization,omitempty"`
	Rdf_DataService  *RdfDataService  `xml:"dcat:DataService,omitempty"`
}

type RdfCatalog struct {
	XMLName        xml.Name `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# dcat:Catalog,omitempty"`
	Attr_rdf_about string   `xml:"rdf:about,attr"`
	Dcterms_title  struct {
		XMLLang string   `xml:"xml:lang,attr"`
		XMLName xml.Name `xml:"dcterms:title"`
		Title   string   `xml:",chardata"`
	}
	Dcterms_description struct {
		XMLLang     string   `xml:"xml:lang,attr"`
		XMLName     xml.Name `xml:"dcterms:description"`
		Description string   `xml:",chardata"`
	}
	Dcterms_publisher struct {
		XMLName           xml.Name `xml:"dcterms:publisher"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
	Dcat_dataset struct {
		XMLName           xml.Name `xml:"dcat:dataset"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
	Dcterms_license struct {
		XMLName           xml.Name `xml:"dcterms:license"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
}

type RdfDataset struct {
	XMLName        xml.Name `xml:"dcat:Dataset"`
	Attr_rdf_about string   `xml:"rdf:about,attr"`
	Dcterms_title  struct {
		XMLLang string   `xml:"xml:lang,attr"`
		XMLName xml.Name `xml:"dcterms:title"`
		Title   string   `xml:",chardata"`
	}
	Dcterms_description struct {
		XMLLang     string   `xml:"xml:lang,attr"`
		XMLName     xml.Name `xml:"dcterms:description"`
		Description string   `xml:",chardata"`
	}
	Dcterms_publisher struct {
		XMLName           xml.Name `xml:"dcterms:publisher"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
	Dcat_distribution struct {
		XMLName           xml.Name `xml:"dcat:distribution"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
	Dcat_contactPoint struct {
		XMLName           xml.Name `xml:"dcat:contactPoint"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
}
type RdfAgent struct {
	XMLName        xml.Name `xml:"foaf:Agent"`
	Attr_rdf_about string   `xml:"rdf:about,attr"`
	Foaf_name      string   `xml:"foaf:name"`
}

type RdfDistribution struct {
	XMLName        xml.Name `xml:"dcat:Distribution"`
	Attr_rdf_about string   `xml:"rdf:about,attr"`
	Dcat_accessURL struct {
		XMLName           xml.Name `xml:"dcat:accessURL"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
	Dcat_accessService struct {
		XMLName           xml.Name `xml:"dcat:accessService"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
}

type RdfOrganization struct {
	XMLName        xml.Name `xml:"dcat:Organization"`
	Attr_rdf_about string   `xml:"rdf:about,attr"`
	Vcard_Fn       string   `xml:"vcard:fn"`
	Vcard_hasEmail struct {
		XMLName           xml.Name `xml:"vcard:hasEmail"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
}

type RdfDataService struct {
	XMLName        xml.Name `xml:"dcat:DataService"`
	Attr_rdf_about string   `xml:"rdf:about,attr"`
	Dcterms_title  struct {
		XMLLang string   `xml:"xml:lang,attr"`
		XMLName xml.Name `xml:"dcterms:title"`
		Title   string   `xml:",chardata"`
	}
	Dcat_endpointURL struct {
		XMLName           xml.Name `xml:"dcat:endpointURL"`
		Attr_rdf_resource string   `xml:"rdf:resource,attr"`
	}
}
