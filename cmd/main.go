package main

import (
	//"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/rishabh-bector/BenevolentBitesBack/auth"
	"github.com/rishabh-bector/BenevolentBitesBack/database"
	"github.com/rishabh-bector/BenevolentBitesBack/places"
	"github.com/rishabh-bector/BenevolentBitesBack/twilio"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

// --------------------
// Api Endpoints
// --------------------
//
// General:
//
// / - returns PROD or DEV environment
// /oauth - redirected to by Google, exchanges auth code
// /verify - allows frontend to validate user
//
// Search:
//
// /search/coords - allows frontend to search for restaurants around coords, given a query string
//
// Restaurants:
//
// /rest/signup - creates a new restaurant
// /rest/getinfo - returns all info
// /rest/setinfo - sets all info
// /rest/getlocations - gets all associated locations from square API
// /rest/setlocation - sets location for a restaurant
// /rest/getdetails - returns detailed info about a restaurant using Google's API
// /rest/redeemcard - allows restaurant to subtract credit from their issued cards
// /rest/setpassword - allows restaurant owner to set a password for staff to redeem customer cards
// /rest/getphoto - returns photo of restaurant from Google Places API
// /rest/verifycall - calls the restaurants number from Google to verify them
// /rest/verifycode - verifies the call code to that which the user entered
// /rest/publish - makes sure that the new restaurant has: information, square, employees, and phone verification
//
// Square:
//
// /square/signup - redirect user to square login
// /square/oauth - redirected to by Square, exchanges auth code
// /square/processcheckout - creates card after a square checkout callback
//
// Users:
//
// /user/signup - creates a new user
// /user/getinfo - returns all info
// /user/setinfo - sets all info
// /user/getavatar - gets user's google avatar
// /user/buy - allows user to purchase credit - see BeginPaymentFlow()
// /user/getcards - returns all of a user's cards and their balances
//
// --------------------
// Environment variables
// --------------------
//
// S_PORT: backend server port
// S_ENV: DEV or PROD
// S_FRONT: frontend redirct URL
//
// G_ID: google client id
// G_SECRET: google client secret
// G_REDIRECT: google redirect url
//
// M_URL: mongo db url
// M_DB: mongo db name
//

var Router *gin.Engine

func main() {
	gin.SetMode("debug")

	log.SetLevel(log.InfoLevel)
	log.Info("BB: S T A R T I N G !")
	log.Info("ENVIRONMENT: ", os.Getenv("S_ENV"))

	auth.Initialize()
	database.Initialize()
	places.Initialize()
	twilio.Initialize()

	Router = gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{os.Getenv("S_CORS")}
	config.AllowCredentials = true
	config.AllowMethods = []string{"POST", "GET", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	config.AllowOriginFunc = func(origin string) bool {
		return origin == os.Getenv("S_CORS")
	}

	Router.Use(cors.New(config))
	Router.LoadHTMLGlob("../templates/*")
	Router.Static("/assets", "../assets")

	Router.GET("/", Healthcheck)
	Router.GET("/oauth", HandleOAuthCode)
	Router.GET("/verify", VerifyToken)

	Router.GET("/search/coords", SearchCoords)

	Router.GET("/rest/signup", StartRESTOAuth2Flow)
	Router.GET("/rest/getinfo", GetRestaurantInfo)
	Router.GET("/rest/getdetails", GetRestaurantDetails)
	Router.GET("/rest/getphoto", GetRestaurantPhoto)
	Router.GET("/rest/verifycall", MakeVerifyCall)
	Router.POST("/rest/verifycode", VerifyCode)
	Router.POST("/rest/setinfo", SetRestaurantInfo)
	Router.POST("/rest/setpassword", SetRestaurantPassword)
	Router.POST("/rest/redeemcard", RedeemCard)
	Router.GET("/rest/getlocations", GetLocations)
	Router.GET("/rest/setlocation", SetLocation)
	Router.GET("/rest/publish", PublishRestaurant)

	Router.GET("/user/signup", StartUSEROAuth2Flow)
	Router.GET("/user/getinfo", GetUserInfo)
	Router.GET("/user/getavatar", GetUserAvatar)
	Router.GET("/user/getcards", GetUserCards)
	Router.GET("/user/buy", BeginPaymentFlow)
	Router.POST("/user/setinfo", SetUserInfo)

	Router.GET("/square/signup", StartSquareOAuth2Flow)
	Router.GET("/square/oauth", HandleSquareOAuthCode)
	Router.GET("/square/processcheckout", ProcessCheckout)

	Router.Run(os.Getenv("S_PORT")) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

// Healthcheck displays DEV or PROD
func Healthcheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "online", "env": os.Getenv("S_ENV")})
}

// StartRESTOAuth2Flow redirects the user to google to begin the OAuth2.0 process
//	- for restaurants
func StartRESTOAuth2Flow(c *gin.Context) {
	r := c.Query("redirect")
	if r == "" {
		r = fmt.Sprintf("%s/restaurants?a=b", os.Getenv("S_FRONT"))
	}
	c.Redirect(307, auth.GetRedirectToGoogle(r))
}

// StartUSEROAuth2Flow redirects the user to google to begin the OAuth2.0 process
//	- for normal users
func StartUSEROAuth2Flow(c *gin.Context) {
	r := c.Query("redirect")
	if r == "" {
		r = fmt.Sprintf("%s/users?a=b", os.Getenv("S_FRONT"))
	}
	c.Redirect(307, auth.GetRedirectToGoogle(r))
}

// HandleOAuthCode is called by Google, and exchanges the auth code for the main access token
func HandleOAuthCode(c *gin.Context) {
	t := auth.GetTokenFromOAuthCode(c.Query("code")).Extra("id_token").(string)

	redirect := c.Query("state")

	u := database.ValidateUser(t)
	if u.Email == "nil" {
		log.Error("BB: Unable to validate token")
		c.Data(200, "text/html", []byte(
			fmt.Sprintf("<html><body onload=\"window.location.replace('%s/restaurants&login=%s&error=%s')\"/></html>",
				redirect,
				"fail",
				"unable to validate token",
			)))
		return
	}

	secure := true
	if os.Getenv("S_ENV") == "LOCAL" {
		secure = false
	}
	c.SetCookie("bb-access", t, 3600, "/", "", secure, false)

	c.Data(200, "text/html", []byte(
		fmt.Sprintf("<html><body onload=\"window.location.replace('%s&login=%s&error=%s');\"/></html>",
			redirect,
			"success",
			"none",
		)))
}

// SetRestaurantInfo allows the frontend to update restaurant info
func SetRestaurantInfo(c *gin.Context) {
	// Obtain and validate token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	// Unmarshal frontend data
	var r database.Restaurant
	if err := c.ShouldBindJSON(&r); err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, invalid json"})
		return
	}

	r.Owner = email

	// Determine PlaceID just in case address changed or new restaurant
	placeID, err := places.GetPlaceID(r.Name, fmt.Sprintf("%s %s %s %s", r.Address, r.City, r.State, r.Zip))
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	r.PlaceID = placeID

	err = database.UpdateRestaurant(email, r)
	if err != nil {
		log.Info(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{})
}

// GetRestaurantInfo retrieves restaurant info for the frontend
func GetRestaurantInfo(c *gin.Context) {
	// Obtain and validate token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	r := database.DoesRestaurantExist(email)
	if r.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, that restaurant doesn't exist"})
		return
	}

	u := database.DoesUserExist(email)
	if u.Email == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to locate owner's account"})
	}

	resp := map[string]interface{}{
		"owner":       r.Owner,
		"contact":     r.ContactEmail,
		"name":        r.Name,
		"address":     r.Address,
		"city":        r.City,
		"state":       r.State,
		"website":     r.Website,
		"yelp":        r.Yelp,
		"description": r.Description,
		"employees":   r.Employees,
		"published":   r.Published,
	}

	if u.Square.MerchantID != "" {
		resp["hasSquare"] = true
	} else {
		resp["hasSquare"] = false
	}

	c.JSON(200, resp)
}

// VerifyToken allows the frontend to authenticate a token
func VerifyToken(c *gin.Context) {
	// Obtain and validate token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"email": verify["email"].(string)})
}

// SetUserInfo allows the frontend to update user info
func SetUserInfo(c *gin.Context) {
	// Obtain and validate token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	// Unmarshal frontend data
	var u database.User
	if err := c.ShouldBindJSON(&u); err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, invalid json"})
		return
	}

	u.Email = email

	err = database.UpdateUser(email, u)
	if err != nil {
		log.Info(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{})
}

// GetUserInfo retrieves user info for the frontend
func GetUserInfo(c *gin.Context) {
	// Obtain and validate token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	u := database.DoesUserExist(email)
	if u.Email == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, that user doesn't exist"})
		return
	}

	// Remove sensitive info
	u.Square = auth.SquareAuth{}
	u.Token = ""

	c.JSON(200, u)
}

// GetUserAvatar retrieves the user's google avatar image
func GetUserAvatar(c *gin.Context) {
	// Obtain and validate token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"avatar": verify["picture"].(string)})
}

// --------------------
// Square API
// --------------------

// StartSquareOAuth2Flow redirects the user to Square to begin the OAuth2.0 process
func StartSquareOAuth2Flow(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	c.Redirect(307, auth.GetRedirectToSquare(email))
}

// HandleSquareOAuthCode is called by Square to deliver the authentication code,
// which is then exchanging for an access_token and a refresh_token
//
// These tokens are then updated in the database
func HandleSquareOAuthCode(c *gin.Context) {
	owner := c.Query("state")
	square, err := auth.GetTokenFromSquareAuthCode(c.Query("code"))
	if err != nil {
		c.Data(200, "text/html", []byte(
			fmt.Sprintf("<html><body onload=\"window.location.replace('%s/restaurants?square=%s&error=%s')\"/></html>",
				os.Getenv("S_FRONT"),
				"fail",
				err.Error(),
			)))
		return
	}

	err = database.UpdateRestaurantSquareAuth(owner, square)
	if err != nil {
		c.Data(200, "text/html", []byte(
			fmt.Sprintf("<html><body onload=\"window.location.replace('%s/restaurants?square=%s&error=%s')\"/></html>",
				os.Getenv("S_FRONT"),
				"fail",
				err.Error(),
			)))
		return
	}

	c.Data(200, "text/html", []byte(
		fmt.Sprintf("<html><body onload=\"window.location.replace('%s/restaurants/chooselocation')\"/></html>",
			os.Getenv("S_FRONT"),
		)))
}

// Get Locations from a Square Merchant ID
func GetLocations(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	// Make sure restaurant exists
	r := database.DoesRestaurantExist(email)
	if r.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find your restaurant"})
		return
	}

	// Find restaurant owner user (square payment recipient)
	u := database.DoesUserExist(r.Owner)
	if u.Email == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find your user"})
		return
	}

	// Refresh their access token
	auth.RefreshAccessToken(&u.Square)

	locations, err := auth.GetLocations(u.Square.AccessToken)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, locations)
}

// Get Locations from a Square Merchant ID
func SetLocation(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	// Make sure restaurant exists
	r := database.DoesRestaurantExist(email)
	if r.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find your restaurant"})
		return
	}

	r.LocationID = c.Query("id")
	err = database.UpdateRestaurant(email, r)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{})
}

// BeginPaymentFlow starts the payment process with the user by serving them the Square checkout page
func BeginPaymentFlow(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	// Make sure restaurant exists
	r := database.DoesRestaurantExistUUID(c.Query("restId"))
	if r.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find that restaurant"})
		return
	}

	// Find restaurant owner user (square payment recipient)
	ro := database.DoesUserExist(r.Owner)
	if ro.Email == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find restaurant auth"})
		return
	}

	// Find restaurant owner user (square payment recipient)
	u := database.DoesUserExist(email)
	if u.Email == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find your user"})
		return
	}

	amount, err := strconv.Atoi(c.Query("amount"))
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, invalid amount"})
		return
	}

	checkout, err := auth.CreateCheckout(amount, r.LocationID, r.Name, &ro.Square)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, could not create a checkout"})
		return
	}

	checkout.RestID = r.UUID
	checkout.UserEmail = u.Email
	checkout.Amount = amount

	c.Redirect(303, checkout.URL)
}

// ProcessCheckout is called by the Checkout page with the checkout ID
func ProcessCheckout(c *gin.Context) {
	checkoutID := c.Query("checkoutId")
	checkout := *auth.OpenCheckouts[checkoutID]

	// Find restaurant
	r := database.DoesRestaurantExistUUID(checkout.RestID)
	if r.Owner == "nil" {
		processCardError(c, "sorry bro, unable to find that restaurant")
		return
	}

	// Add new card in database
	trans := database.Transaction{
		Timestamp: checkout.Timestamp,
		Amount:    checkout.Amount,
		ID:        checkout.ID,
	}
	_, err := database.CreateCard(checkout.UserEmail, checkout.RestID, trans)
	if err != nil {
		processCardError(c, err.Error())
		return
	}

	c.Redirect(303, fmt.Sprintf("%s/users/cards", os.Getenv("S_FRONT")))
}

// Helper to return ProcessCard errors
func processCardError(c *gin.Context, err string) {
	c.Data(403, "text/html", []byte(
		fmt.Sprintf("%s/users/cards?&error=%s",
			os.Getenv("S_FRONT"),
			err,
		)))
}

// GetUserCards returns all of a user's cards
func GetUserCards(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	cards, err := database.GetUserCards(email)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
	}

	c.JSON(200, cards)
}

func SearchCoords(c *gin.Context) {
	i, err := strconv.Atoi(c.Query("range"))
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	s, err := places.SearchCoords(c.Query("query"), c.Query("lat"), c.Query("lng"), i)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, s)
}

func GetRestaurantDetails(c *gin.Context) {
	rest := c.Query("restId")

	dbRest := database.DoesRestaurantExistUUID(rest)
	if dbRest.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find that restaurant"})
		return
	}

	c.JSON(200, gin.H{"name": dbRest.Name})
}

func GetRestaurantPhoto(c *gin.Context) {
	photoReference := c.Query("photoreference")

	res, contentLength, err := places.GetPlacePhoto(photoReference)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
	}

	c.DataFromReader(200, contentLength, res.ContentType, res.Data, map[string]string{})
}

func SetRestaurantPassword(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	r := database.DoesRestaurantExist(email)
	if r.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find that restaurant"})
		return
	}

	var data map[string]interface{}
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	hash, err := HashPassword(data["password"].(string))
	if err != nil {
		c.JSON(403, gin.H{"error": "sorry bro, error code: 98234"})
		return
	}

	r.PassHash = hash
	err = database.UpdateRestaurant(email, r)
	if err != nil {
		c.JSON(403, gin.H{"error": "sorry bro, error code: 98235"})
		return
	}

	c.JSON(200, gin.H{})
}

type RedeemCardData struct {
	CardID   string `json:"cardId"`
	Password string `json:"password"`
	Amount   int    `json:"amount"`
}

// RedeemCard verifies that the password entered by the staff
// member into the frontend is correct and then updates the card balance accordingly
func RedeemCard(c *gin.Context) {
	var data RedeemCardData

	err := c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	cardDb := database.DoesCardExist(data.CardID)
	if cardDb.UUID == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find that card"})
		return
	}

	restDb := database.DoesRestaurantExistUUID(cardDb.RestUUID)
	if restDb.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find the restaurant for that card"})
		return
	}

	// Verify password
	verified := CheckPasswordHash(data.Password, restDb.PassHash)
	if !verified {
		c.JSON(403, gin.H{"error": "sorry bro, invalid password"})
		return
	}

	// Redeem card
	err = database.SubtractCredit(data.CardID, data.Amount)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{})
}

func MakeVerifyCall(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	// Look up restaurant in DB
	restDb := database.DoesRestaurantExist(email)
	if restDb.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find that restaurant"})
		return
	}

	// Get place details
	details, err := places.GetPlaceDetails(restDb.PlaceID)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	recipient := details.InternationalPhoneNumber
	err = twilio.MakeConfirmationCall(recipient, email)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"phone": recipient})
}

func VerifyCode(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "sorry bro, unable to find cookie token"})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	var data map[string]string
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}

	// Verify code
	verified := twilio.VerifyCode(email, data["code"])
	if !verified {
		c.JSON(403, gin.H{"error": "sorry bro, wrong code"})
		return
	}

	// Look up restaurant in DB
	restDb := database.DoesRestaurantExist(email)
	if restDb.Owner == "nil" {
		c.JSON(403, gin.H{"error": "sorry bro, unable to find that restaurant"})
		return
	}

	// Update verified status
	restDb.Verified = true
	database.UpdateRestaurant(email, restDb)

	c.JSON(200, gin.H{})
}

func PublishRestaurant(c *gin.Context) {
	// Obtain and validate google token
	token, err := c.Cookie("bb-access")
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": "Unable to find cookie token. Please login again."})
		return
	}

	verify, err := auth.ValidateToken(token)
	if err != nil {
		log.Error(err)
		c.JSON(403, gin.H{"error": err.Error()})
		return
	}
	email := verify["email"].(string)

	// Look up restaurant in DB
	restDb := database.DoesRestaurantExist(email)
	if restDb.Owner == "nil" {
		c.JSON(403, gin.H{"error": "Unable to find that restaurant"})
		return
	}

	// Look up user
	userDb := database.DoesUserExist(email)
	if userDb.Name == "nil" {
		c.JSON(403, gin.H{"error": "Unable to find that restaurant."})
		return
	}

	// Check restaurant details
	if restDb.Name == "" || restDb.Description == "" {
		c.JSON(403, gin.H{"error": "Unable to find restaurant name or description."})
		return
	}

	// Check square
	if userDb.Square.AccessToken == "" {
		c.JSON(403, gin.H{"error": "Unable to find restaurant square integration."})
		return
	}

	// Check employees
	if len(restDb.Employees) > 0 {
		c.JSON(403, gin.H{"error": "Unable to find restaurant employees."})
		return
	}

	// Check phone verification
	if restDb.Verified == false {
		c.JSON(403, gin.H{"error": "Please authenticate your restaurant by phone."})
		return
	}

	// Publish restaurant
	restDb.Published = true
	database.UpdateRestaurant(email, restDb)

	c.JSON(200, gin.H{})
}
