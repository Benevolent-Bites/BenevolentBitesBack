package twilio

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/go-resty/resty/v2"
)

// Codes stores all of the currently issued codes in RAM
// map[email]code
var Codes map[string]string

var (
	TwilioNumber string
	RestyClient  *resty.Client
)

func Initialize() {
	TwilioNumber = os.Getenv("TW_NUM")
	RestyClient = resty.New()
	RestyClient.SetBasicAuth("AC8755a2368f6e69bd997a1c0d2e5f40e3", os.Getenv("TW_TKN"))
}

// MakeConfirmationCall calls a restaurant's phone number to verify them
func MakeConfirmationCall(recipient string, email string) error {
	code := generateConfirmationCode()
	_, err := RestyClient.R().SetFormData(
		map[string]string{
			"To":    recipient,
			"From":  TwilioNumber,
			"Twiml": fmt.Sprintf("<Response><Say>Hello! Your restaurant verification code is %s</Say></Response>", code)}).
		Post("https://api.twilio.com/2010-04-01/Accounts/AC8755a2368f6e69bd997a1c0d2e5f40e3/Calls.json")

	if err != nil {
		return err
	}

	Codes[email] = code

	return nil
}

func VerifyCode(email string, code string) bool {
	if c, ok := Codes[email]; ok {
		if code == c {
			return true
		}
	}
	return false
}

// 4 digit random pin
func generateConfirmationCode() string {
	return fmt.Sprintf("%s %s %s %s",
		rand.Intn(10),
		rand.Intn(10),
		rand.Intn(10),
		rand.Intn(10),
	)
}
