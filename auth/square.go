package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

type SquareAuth struct {
	MerchantID   string `bson:"merchantId" json:"merchantId"`
	AccessToken  string `bson:"accessToken" json:"accessToken"`
	RefreshToken string `bson:"refreshToken" json:"refreshToken"`
}

func GetRedirectToSquare(email string) string {
	return fmt.Sprintf("%s/oauth2/authorize?client_id=%s&state=%s&scope=MERCHANT_PROFILE_READ PAYMENTS_WRITE ORDERS_WRITE", os.Getenv("SQ_URL"), os.Getenv("SQ_APPID"), email)
}

var OpenCheckouts = map[string]*Checkout{}

func GetTokenFromSquareAuthCode(code string) (SquareAuth, error) {
	requestData, err := json.Marshal(map[string]string{
		"client_id":     os.Getenv("SQ_APPID"),
		"client_secret": os.Getenv("SQ_SECRET"),
		"grant_type":    "authorization_code",
		"code":          code,
	})

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth2/token", os.Getenv("SQ_URL")), bytes.NewBuffer(requestData))
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Error(err)
		return SquareAuth{}, err
	}

	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(request)
	if err != nil {
		log.Error(err)
		return SquareAuth{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return SquareAuth{}, err
	}

	var rjson map[string]interface{}
	err = json.Unmarshal(body, &rjson)
	if err != nil {
		log.Error(err)
		return SquareAuth{}, err
	}

	if val, ok := rjson["access_token"]; ok {
		return SquareAuth{
			MerchantID:   rjson["merchant_id"].(string),
			AccessToken:  val.(string),
			RefreshToken: rjson["refresh_token"].(string),
		}, nil
	}

	return SquareAuth{}, errors.New("unable to find square access token")
}

type Location struct {
	Id          string `json:"id"`
	Address     string `json:"address"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Checkout struct {
	ID        string `json:"id"`
	URL       string `json:"checkout_page_url"`
	Timestamp string `json:"created_at"`
	Amount    int
	RestID    string
	UserEmail string
}

func CreateCheckout(amount int, locationID string, restName string, s *SquareAuth) (*Checkout, error) {

	requestData, err := json.Marshal(map[string]interface{}{
		"idempotency_key": GenerateUUID(),
		"redirect_url":    fmt.Sprintf(os.Getenv("SQ_REDIRECT")),
		"order": map[string]interface{}{
			"idempotency_key": GenerateUUID(),
			"order": map[string]interface{}{
				"location_id": locationID,
				"line_items": []map[string]interface{}{
					{
						"quantity": "1",
						"name":     fmt.Sprintf("%s Gift Card", restName),
						"base_price_money": map[string]interface{}{
							"amount":   amount,
							"currency": "USD",
						},
					},
				},
			},
		},
	})

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v2/locations/%s/checkouts", os.Getenv("SQ_URL"), locationID),
		bytes.NewBuffer(requestData))
	request.Header.Set("Content-Type", "application/json")

	// err = RefreshAccessToken(s)
	// if err != nil {                            ONLY FOR TESTING
	// 	return &Checkout{}, err
	// }

	authHeader := fmt.Sprintf("Bearer %s", s.AccessToken)
	request.Header.Set("Authorization", authHeader)

	if err != nil {
		return &Checkout{}, err
	}

	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(request)
	if err != nil {
		return &Checkout{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &Checkout{}, err
	}

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return &Checkout{}, err
	}

	if checkout, ok := response["checkout"]; ok {
		jCheckout, _ := json.Marshal(checkout)
		var cCheckout Checkout
		if err := json.Unmarshal(jCheckout, &cCheckout); err == nil {
			OpenCheckouts[cCheckout.ID] = &cCheckout
			return &cCheckout, nil
		}
		return &Checkout{}, fmt.Errorf("couldn't cast %s as Checkout", checkout)
	}

	return &Checkout{}, fmt.Errorf("%s", response["errors"].([]interface{})[0])
}

func GetLocations(token string) ([]Location, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v2/locations", os.Getenv("SQ_URL")), nil)
	request.Header.Set("Content-Type", "application/json")

	authHeader := fmt.Sprintf("Bearer %s", token)
	request.Header.Set("Authorization", authHeader)
	if err != nil {
		log.Error(err)
		return []Location{}, err
	}

	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(request)
	if err != nil {
		log.Error(err)
		return []Location{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return []Location{}, err
	}

	var rjson map[string][]map[string]interface{}
	err = json.Unmarshal(body, &rjson)
	if err != nil {
		log.Error(err)
		return []Location{}, err
	}

	if locations, ok := rjson["locations"]; ok {
		var output []Location
		for _, v := range locations {
			var addLine1, addState, addCity, addZip string
			if add, ok := v["address"].(map[string]interface{}); ok {
				addLine1 = add["address_line_1"].(string)
				addState = add["administrative_district_level_1"].(string)
				addCity = add["locality"].(string)
				addZip = add["postal_code"].(string)
				log.Info("burh")
			}
			var description string
			if desc, ok := v["description"]; ok {
				description = desc.(string)
			}
			output = append(output, Location{
				Id:          v["id"].(string),
				Address:     fmt.Sprintf("%s %s, %s %s", addLine1, addCity, addState, addZip),
				Name:        v["name"].(string),
				Description: description,
			})
		}
		return output, nil
	}

	return []Location{}, errors.New("unable to find locations")
}

func RefreshAccessToken(s *SquareAuth) error {
	if s.RefreshToken == "" {
		return errors.New("No refresh token")
	}
	requestData, err := json.Marshal(map[string]string{
		"client_id":     os.Getenv("SQ_APPID"),
		"client_secret": os.Getenv("SQ_SECRET"),
		"grant_type":    "refresh_token",
		"refresh_token": s.RefreshToken,
	})

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth2/token", os.Getenv("SQ_URL")), bytes.NewBuffer(requestData))
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Error(err)
		return err
	}

	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(request)
	if err != nil {
		log.Error(err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return err
	}

	var rjson map[string]interface{}
	err = json.Unmarshal(body, &rjson)
	if err != nil {
		log.Error(err)
		return err
	}

	if val, ok := rjson["access_token"]; ok {
		s.AccessToken = val.(string)
	}

	return nil
}

func ConvertSquareToMap(u SquareAuth) map[string]interface{} {
	var out map[string]interface{}
	m, _ := json.Marshal(u)
	json.Unmarshal(m, &out)
	return out
}

func ConvertMapToSquare(mIn map[string]interface{}) SquareAuth {
	var out SquareAuth
	m, _ := json.Marshal(mIn)
	json.Unmarshal(m, &out)
	return out
}

func MergeSquareAuths(uOld, uNew SquareAuth) SquareAuth {
	mOut := ConvertSquareToMap(uOld)
	nMap := ConvertSquareToMap(uNew)
	for k, v := range nMap {
		if vc, ok := v.(string); ok {
			if vc != "" {
				mOut[k] = vc
			}
		}
	}
	return ConvertMapToSquare(mOut)
}
