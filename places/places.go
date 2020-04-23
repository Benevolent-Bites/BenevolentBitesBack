package places

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/rishabh-bector/BenevolentBitesBack/database"
	log "github.com/sirupsen/logrus"

	"googlemaps.github.io/maps"
)

var (
	GKey string
)

// Initialize creates the Google Maps API client
func Initialize() {
	GKey = os.Getenv("G_API")
}

// SearchResponse contains the info returned by a restaurant search query.
// For any given address/radius, both restaurants supported by Benevolent Bites
// and those that are not supported will be returned.
type SearchResponse struct {
	On  []APIDetails `json:"on"`
	Off []APIDetails `json:"off"`
}

type APIDetails struct {
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Image       string  `json:"image"`
	Rating      float32 `json:"rating"`
	PriceLevel  int     `json:"priceLevel"`
	Description string  `json:"description"`
	RestID      string  `json:"restID"`
}

// SearchCoords searches for restaurants around the Coords of the origin, based
// on the query provided by the frontend
func SearchCoords(query, lat, lng string, rngMiles int) (SearchResponse, error) {
	params := map[string]string{
		"key":      GKey,
		"location": fmt.Sprintf("%s,%s", lat, lng),
		"radius":   fmt.Sprintf("%v", rngMiles*1600),
		"keyword":  query,
		"type":     "restaurant",
	}

	var res maps.PlacesSearchResponse
	body, err := SendGAPIRequest("https://maps.googleapis.com/maps/api/place/nearbysearch/json", params)
	if err != nil {
		log.Error(err)
		return SearchResponse{}, err
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Error(err)
		file, err := os.Create("log.txt")
		file.Write(body)
		return SearchResponse{}, err
	}

	places := res.Results

	// Gather more search results from page tokens
	depth := 0
	nToken := res.NextPageToken

	for depth < 5 {
		if nToken != "" {
			nextRes, err := ResolvePageToken(params["key"], params["location"], params["radius"], res.NextPageToken)
			if err != nil {
				log.Info(err)
			}
			places = append(places, nextRes.Results...)

			nToken = nextRes.NextPageToken
			depth += 1
		} else {
			break
		}
	}

	sr := SearchResponse{
		On:  []APIDetails{},
		Off: []APIDetails{},
	}

	for p := range places {
		pid := places[p].PlaceID
		r := database.DoesRestaurantExistPlaceID(pid)

		d := APIDetails{
			Name:       places[p].Name,
			Address:    places[p].Vicinity,
			Latitude:   places[p].Geometry.Location.Lat,
			Longitude:  places[p].Geometry.Location.Lng,
			Rating:     places[p].Rating,
			PriceLevel: places[p].PriceLevel,
		}

		if len(places[p].Photos) > 0 {
			d.Image = places[p].Photos[0].PhotoReference
		}

		if r.Owner == "nil" {
			d.RestID = pid

			sr.Off = append(sr.Off, d)
		} else {
			d.Description = r.Description
			d.RestID = r.UUID
			d.Name = r.Name

			if r.Published {
				sr.On = append(sr.On, d)
			}
		}
	}

	return sr, nil
}

func ResolvePageToken(key, location, radius, tok string) (maps.PlacesSearchResponse, error) {
	params := map[string]string{
		"key":       key,
		"location":  location,
		"radius":    radius,
		"pagetoken": tok,
	}

	var res maps.PlacesSearchResponse
	body, err := SendGAPIRequest("https://maps.googleapis.com/maps/api/place/nearbysearch/json", params)
	if err != nil {
		log.Error(err)
		return maps.PlacesSearchResponse{}, err
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Error(err)
		file, err := os.Create("log.txt")
		file.Write(body)
		return maps.PlacesSearchResponse{}, errors.New(err.Error() + " " + string(body))
	}

	return res, nil
}

func GetPlacePhoto(pr string) (maps.PlacePhotoResponse, int64, error) {
	response := maps.PlacePhotoResponse{}
	res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/place/photo?key=%s&photoreference=%s&maxwidth=%v", GKey, pr, 400))
	if err != nil {
		return response, 0, err
	}

	response.Data = res.Body
	response.ContentType = res.Header.Get("Content-Type")

	return response, res.ContentLength, nil
}

// GetPlaceID uses the Google Places API to search for the "place id"
// of a particular address
func GetPlaceID(restName string, address string) (string, error) {
	params := map[string]string{
		"key":       GKey,
		"input":     fmt.Sprintf("%s %s", restName, address),
		"inputtype": "textquery",
		"fields":    "place_id",
	}

	var res maps.FindPlaceFromTextResponse
	body, err := SendGAPIRequest("https://maps.googleapis.com/maps/api/place/findplacefromtext/json", params)
	if err != nil {
		return "", err
	}
	json.Unmarshal(body, &res)

	if len(res.Candidates) == 0 {
		return "", errors.New("sorry bro, no restaurant found at that address")
	}

	return res.Candidates[0].PlaceID, nil
}

// GetPlaceDetails uses the Google Places API to find details
// about a particular establishment
//
func GetPlaceDetails(placeID string) (maps.PlaceDetailsResult, error) {
	params := map[string]string{
		"key":      GKey,
		"place_id": placeID,
	}

	var res map[string]interface{}
	body, err := SendGAPIRequest("https://maps.googleapis.com/maps/api/place/details/json", params)
	if err != nil {
		log.Error(err.Error())
		return maps.PlaceDetailsResult{}, err
	}

	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Errorf("error unmarshaling google response, %s", err.Error())
		file, err := os.Create("log.txt")
		file.Write(body)
		return maps.PlaceDetailsResult{}, err
	}

	if _, ok := res["result"]; !ok {
		return maps.PlaceDetailsResult{}, fmt.Errorf("%s", res)
	}

	var resMain maps.PlaceDetailsResult
	body2, err := json.Marshal(res["result"].(map[string]interface{}))
	if err != nil {
		log.Errorf("error marshaling, %s", err.Error())
		return maps.PlaceDetailsResult{}, err
	}

	err = json.Unmarshal(body2, &resMain)
	if err != nil {
		log.Errorf("error unmarshaling google response [result]", err.Error())
		return maps.PlaceDetailsResult{}, err
	}

	return resMain, nil
}

// SendGAPIRequest uses api.py to send a GET to any url, with the given params
// This is because, for some mysterious reason, it doesn't work in Go
func SendGAPIRequest(url string, params map[string]string) ([]byte, error) {
	b, _ := json.Marshal(params)
	cmd := exec.Command(
		"python3",
		"../places/api.py",
		"--url", url,
		"--params", string(b),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(err.Error() + " " + string(out))
	}

	out2 := strings.Replace(string(out), "True", "true", -1)
	out3 := strings.Replace(string(out2), "False", "false", -1)
	out4 := strings.Replace(string(out3), "\n", "", -1)
	out5 := strings.Replace(string(out4), "\\", "", -1)

	onlydouble := []rune(strings.Replace(string(out5), "'", "\"", -1))

	for i := 0; i < len(onlydouble); i++ {
		if onlydouble[i] == rune('"') {
			if onlydouble[i+1] != rune(',') && onlydouble[i+1] != rune('}') && onlydouble[i+1] != rune(']') && onlydouble[i+1] != rune(':') {
				if onlydouble[i-1] != rune('{') && onlydouble[i-1] != rune('[') {
					if i > 1 && !(onlydouble[i-1] == rune(' ') && (onlydouble[i-2] == rune(':') || onlydouble[i-2] == rune(','))) {
						onlydouble[i] = '\''
					}
				}
			}
		}
	}

	return []byte(string(onlydouble)), nil
}
