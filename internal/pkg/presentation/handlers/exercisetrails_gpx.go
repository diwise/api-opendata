package handlers

import (
	"fmt"

	"github.com/diwise/api-opendata/internal/pkg/domain"
)

const xmlHeader string = `<?xml version="1.0" encoding="UTF-8"?>
<gpx creator="diwise cip" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd" version="1.1" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>%s</name>
    <trkseg>`

const trkPtFmt string = `
      <trkpt lat="%0.6f" lon="%0.6f">%s
      </trkpt>`

const eleFmt string = `
        <ele>%0.1f</ele>`

const xmlFooter string = `
    </trkseg>
  </trk>
</gpx>`

func convertTrailToGPX(t *domain.ExerciseTrail) ([]byte, error) {

	gpx := fmt.Sprintf(xmlHeader, t.Name)

	for index := range t.Location.Coordinates {
		elevation := ""
		if len(t.Location.Coordinates[index]) > 2 {
			elevation = fmt.Sprintf(eleFmt, t.Location.Coordinates[index][2])
		}
		gpx = gpx + fmt.Sprintf(trkPtFmt, t.Location.Coordinates[index][1], t.Location.Coordinates[index][0], elevation)
	}

	gpx = gpx + xmlFooter

	return []byte(gpx), nil
}
