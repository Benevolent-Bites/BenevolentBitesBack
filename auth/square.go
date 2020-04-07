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
	return fmt.Sprintf("https://squareup.com/oauth2/authorize?client_id=%s&state=%s&scope=MERCHANT_PROFILE_READ PAYMENTS_READ SETTLEMENTS_READ BANK_ACCOUNTS_READ PAYMENTS_WRITE", os.Getenv("SQ_APPID"), email)
}

func GetTokenFromSquareAuthCode(code string) (SquareAuth, error) {
	requestData, err := json.Marshal(map[string]string{
		"client_id":     os.Getenv("SQ_APPID"),
		"client_secret": os.Getenv("SQ_SECRET"),
		"grant_type":    "authorization_code",
		"code":          code,
	})

	request, err := http.NewRequest("POST", "https://connect.squareup.com/oauth2/token", bytes.NewBuffer(requestData))
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
	id          string `json:"id"`
	address     string `json:"address"`
	name        string `json:"name"`
	description string `json:"description"`
}

func GetLocations(token string) ([]Location, error) {
	request, err := http.NewRequest("GET", "https://connect.squareup.com/v2/locations")
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
		output := make([]Location, len(locations))
		for _, v := range locations {
			add := v["address"].(map[string]string)
			addLine1 := add["address_line_1"]
			addState := add["administrative_district_level_1"]
			addCity := add["locality"]
			addZip := add["postal_code"]
			var description string
			if desc, ok := v["description"]; ok {
				description = desc
			}
			output = append(output, Location{
				id:          v["id"].(string),
				address:     fmt.Sprintf("%s %s, %s %s", addLine1, addCity, addState, addZip),
				name:        v["name"].(string),
				description: description,
			})
		}
		return output, nil
	}

	return []Location{}, errors.New("unable to find locations")
}

func RefreshAccessToken(s *SquareAuth) error {
	requestData, err := json.Marshal(map[string]string{
		"client_id":     os.Getenv("SQ_APPID"),
		"client_secret": os.Getenv("SQ_SECRET"),
		"grant_type":    "refresh_token",
		"refresh_token": s.RefreshToken,
	})

	request, err := http.NewRequest("POST", "https://connect.squareup.com/oauth2/token", bytes.NewBuffer(requestData))
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
