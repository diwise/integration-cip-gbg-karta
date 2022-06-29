package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestBeachUnmarshal(t *testing.T) {
	is := is.New(t)
	b := &Beach{}
	json.Unmarshal([]byte(beachJson), b)

	is.Equal("urn:ngsi-ld:Beach:SE0A21480000004452", b.Id)
	is.Equal("61e0a246cfc4d247cca9604c", b.Source)
	lon, lat := b.AsPoint()
	is.Equal(11.837656, lon)
	is.Equal(57.658862, lat)
}

func TestWaterQualityObservedUnmarshal(t *testing.T) {
	is := is.New(t)
	wqo := &WaterQualityObserved{}
	json.Unmarshal([]byte(wqoString), wqo)

	is.Equal("urn:ngsi-ld:WaterQualityObserved:SE0A21480000000532:2022-06-27T17:00:00+02:00", wqo.Id)
    is.Equal(18.8, wqo.Temp)
    is.Equal("https://www.smhi.se/", wqo.Source)
    is.True(wqo.Time() != time.Time{})
}

const wqoString = `
    {
        "@context": "https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld",
        "id": "urn:ngsi-ld:WaterQualityObserved:SE0A21480000000532:2022-06-27T17:00:00+02:00",
        "type": "WaterQualityObserved",
        "dateObserved": {
            "@type": "DateTime",
            "@value": "2022-06-27T17:00:00+02:00"
        },
        "source": "https://www.smhi.se/",
        "temperature": 18.8,
        "location": {
            "type": "Point",
            "coordinates": [
                11.924098,
                57.623416
            ]
        }
    }
`

const beachJson string = `
   {
        "@context": "https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld",
        "id": "urn:ngsi-ld:Beach:SE0A21480000004452",
        "type": "Beach",
        "areaServed": "Långedrag",
        "dataProvider": "ServiceGuiden",
        "dateCreated": {
            "@type": "DateTime",
            "@value": "2022-06-27T15:41:25Z"
        },
        "facilities": [
            "Hund tillåtet",
            "Hav"
        ],
        "source": "61e0a246cfc4d247cca9604c",
        "description": "<p>En liten klippholme, inte långt från Saltholmens spårvagnshållplats. Badplatsen är en av de mest populära i Göteborg och här badar du antingen från klippor eller tar hjälp av badstegarna. Kiosk finns vid hållplatsen.</p>\r\n<p><strong>Utrustning:</strong> Badstegar. Friluftstoaletter öppna under <a href=\"https://goteborg.se/wps/portal?uri=gbglnk%3a20201219207511\" target=\"_blank\" rel=\"noopener\">Badsäsongen</a>.</p>\r\n<p><strong>Närmaste hållplats: </strong>Saltholmen. Från hållplatsen är det 500 meter till badplatsen.</p>\r\n<p><strong>Närmaste flexlinjemötesplats:</strong> 1160 Skärgårdsterminalen (Flexlinjen Älvsborg)</p>\r\n<p><a href=\"https://www.parkeringgoteborg.se/hitta-parkering/?searchtext=Aspholmen+%28Saltholmen%29&amp;VisitOrRent=Visit&amp;parkingtype=1&amp;vehicletype=1&amp;SubmitSearchParking_ParkingPage_Visit=Visa\" target=\"_blank\" rel=\"noopener\">Hitta närmsta parkering hos Parkering Göteborg</a></p>\r\n<p>Om soptunnan eller containern är full, var snäll och ta med dig ditt skräp hem. Tack för att du tar ditt ansvar! </p>",
        "name": "Aspholmen (Saltholmen)",
        "seeAlso": [
            "https://goteborg.se/wps/portal/start/kultur-och-fritid/fritid-och-natur/friluftsliv-natur-och/badplatser--utomhusbad/badplatser-utomhusbad/?id=3710",
            "https://badplatsen.havochvatten.se/badplatsen/api/testlocationprofile/SE0A21480000004452",
            "https://www.t-d.se/sv/TD2/Avtal/Goteborgs-stad/Aspholmen-Saltholmen/"
        ],
        "location": {
            "type": "MultiPolygon",
            "coordinates": [
                [
                    [
                        [
                            11.837656,
                            57.658862
                        ],
                        [
                            11.837656,
                            57.658962
                        ],
                        [
                            11.837756,
                            57.658962
                        ],
                        [
                            11.837656,
                            57.658862
                        ]
                    ]
                ]
            ]
        }
    }
`
