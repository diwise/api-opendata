package sportsvenues

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func TestExpectedOutputOfGetByID(t *testing.T) {
	is := is.New(t)

	ms := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(response.Code(http.StatusOK), response.Body([]byte(testData))),
	)
	defer ms.Close()

	svci := NewSportsVenueService(context.Background(), zerolog.Logger{}, ms.URL(), "ignored")
	svc, ok := svci.(*sportsvenueSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.NoErr(err)

	sportsvenue, err := svc.GetByID("urn:ngsi-ld:SportsVenue:se:sundsvall:facilities:641")
	is.NoErr(err)

	sportsvenueJSON, err := json.Marshal(sportsvenue)
	is.NoErr(err)

	is.Equal(expectedOutput, string(sportsvenueJSON))
}

func TestExpectedOutputOfGetAll(t *testing.T) {
	is := is.New(t)
	ms := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(response.Code(http.StatusOK), response.Body([]byte(testData))),
	)
	defer ms.Close()

	svci := NewSportsVenueService(context.Background(), zerolog.Logger{}, ms.URL(), "ignored")
	svc, ok := svci.(*sportsvenueSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.NoErr(err)

	sportsfields := svc.GetAll([]string{})

	is.Equal(len(sportsfields), 1)
}

const testData string = `[{"@context":["https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld"],"category":["ice-rink"],"dateCreated":{"@type":"DateTime","@value":"2018-10-11T13:31:18Z"},"dateModified":{"@type":"DateTime","@value":"2021-02-19T14:42:28Z"},"description":"en bra beskrivning","id":"urn:ngsi-ld:SportsVenue:se:sundsvall:facilities:641","location":{"type":"MultiPolygon","coordinates":[[[[17.34617972962255,62.412574010033595],[17.347279929404046,62.41262480545839],[17.34723895085804,62.41267442979932],[17.34677784825609,62.41265203299056],[17.3467384430584,62.412700741615886],[17.346158134543554,62.41267312155208],[17.34617972962255,62.412574010033595]]],[[[17.34761399162344,62.41237392319236],[17.34855035006534,62.41242025816159],[17.34855487518801,62.41240807636365],[17.348697258604084,62.41241502451384],[17.348694859337424,62.4124266585969],[17.3491824164106,62.41245060556396],[17.349186451844567,62.412439096546166],[17.34925259028114,62.412442947393956],[17.349242665669887,62.412488258989285],[17.34951570696293,62.41250324055892],[17.349450338564544,62.41279074442415],[17.34917837549993,62.41277769325007],[17.349174084398584,62.41278964661511],[17.34910750991809,62.41278567734849],[17.34911035207147,62.41277426029809],[17.348622494412524,62.41275099160957],[17.348618421809622,62.412762276725374],[17.34847512574829,62.41275531691869],[17.348477443178098,62.41274436672738],[17.347990639200116,62.412720198001196],[17.347987258641506,62.4127321629188],[17.347920447429903,62.4127286281349],[17.347926048373157,62.41270323609897],[17.347804278351486,62.41269643367649],[17.347780967699386,62.41264408493952],[17.34755669799009,62.412629526396316],[17.34761399162344,62.41237392319236]]],[[[17.345975603255734,62.412564444047085],[17.346071624928104,62.41211577666336],[17.34712002235073,62.412164188763754],[17.34713915061031,62.41208236135621],[17.347485282876978,62.412099515178376],[17.34746669392995,62.41218019533143],[17.347655476475186,62.41218891145522],[17.347554686956936,62.41263845980769],[17.345975603255734,62.412564444047085]]]]},"name":"Stora ishallen","seeAlso":["https://sundsvall.se/kontakter/uthyrningsbyran-2/"],"source":"https://api.sundsvall.se/facilities/2.1/get/641","type":"SportsVenue"}]`

const expectedOutput string = `{"id":"urn:ngsi-ld:SportsVenue:se:sundsvall:facilities:641","name":"Stora ishallen","description":"en bra beskrivning","categories":["ice-rink"],"location":{"type":"MultiPolygon","coordinates":[[[[17.34617972962255,62.412574010033595],[17.347279929404046,62.41262480545839],[17.34723895085804,62.41267442979932],[17.34677784825609,62.41265203299056],[17.3467384430584,62.412700741615886],[17.346158134543554,62.41267312155208],[17.34617972962255,62.412574010033595]]],[[[17.34761399162344,62.41237392319236],[17.34855035006534,62.41242025816159],[17.34855487518801,62.41240807636365],[17.348697258604084,62.41241502451384],[17.348694859337424,62.4124266585969],[17.3491824164106,62.41245060556396],[17.349186451844567,62.412439096546166],[17.34925259028114,62.412442947393956],[17.349242665669887,62.412488258989285],[17.34951570696293,62.41250324055892],[17.349450338564544,62.41279074442415],[17.34917837549993,62.41277769325007],[17.349174084398584,62.41278964661511],[17.34910750991809,62.41278567734849],[17.34911035207147,62.41277426029809],[17.348622494412524,62.41275099160957],[17.348618421809622,62.412762276725374],[17.34847512574829,62.41275531691869],[17.348477443178098,62.41274436672738],[17.347990639200116,62.412720198001196],[17.347987258641506,62.4127321629188],[17.347920447429903,62.4127286281349],[17.347926048373157,62.41270323609897],[17.347804278351486,62.41269643367649],[17.347780967699386,62.41264408493952],[17.34755669799009,62.412629526396316],[17.34761399162344,62.41237392319236]]],[[[17.345975603255734,62.412564444047085],[17.346071624928104,62.41211577666336],[17.34712002235073,62.412164188763754],[17.34713915061031,62.41208236135621],[17.347485282876978,62.412099515178376],[17.34746669392995,62.41218019533143],[17.347655476475186,62.41218891145522],[17.347554686956936,62.41263845980769],[17.345975603255734,62.412564444047085]]]]},"dateCreated":"2018-10-11T13:31:18Z","dateModified":"2021-02-19T14:42:28Z","source":"https://api.sundsvall.se/facilities/2.1/get/641","seeAlso":["https://sundsvall.se/kontakter/uthyrningsbyran-2/"]}`