package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	Conf *oauth2.Config
)

// Initialize creates the OAuth2.0 client
func Initialize() {
	log.Info("BB: Initializing auth service")
	Conf = &oauth2.Config{
		ClientID:     os.Getenv("G_ID"),
		ClientSecret: os.Getenv("G_SECRET"),
		RedirectURL:  os.Getenv("G_REDIRECT"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

// GetRedirectToGoogle returns the url for the user to authenticate with Google
func GetRedirectToGoogle(state string) string {
	return Conf.AuthCodeURL(state)
}

// GetTokenFromOAuthCode exchanges the auth code with Google for the oauth token
func GetTokenFromOAuthCode(code string) *oauth2.Token {
	// Handle the exchange code to initiate a transport.
	tok, err := Conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Error(err)
	}
	return tok
}

// ValidateToken verifies that a token is valid through Google's API
func ValidateToken(t string) (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", t))
	if err != nil {
		return nil, errors.New("sorry bro, unable to verify your token")
	}
	defer resp.Body.Close()

	rdr, err := ioutil.ReadAll(resp.Body)
	var claims map[string]interface{}
	json.Unmarshal(rdr, &claims)
	return claims, nil
}
