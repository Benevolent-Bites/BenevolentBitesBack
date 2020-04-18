package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rishabh-bector/BenevolentBitesBack/auth"
	"github.com/rishabh-bector/BenevolentBitesBack/database"
	log "github.com/sirupsen/logrus"
)

type RestaurantReport struct {
	Total      int `json:"total"`
	Restaurant int `json:"restaurant"`
	Employees  int `json:"employees"`

	Transactions []ReportTransaction `json:"transactions"`
	Sales        []ReportTransaction `json:"sales"`

	Redeemed    int `json:"redeemed"`
	Outstanding int `json:"outstanding"`
}

type ReportTransaction struct {
	Timestamp string `json:"timestamp"`
	Amount    int    `json:"amount"`
}

// CreateRestaurantReport returns an informative report to the restaurant owner
func CreateRestaurantReport(c *gin.Context) {
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

	// Get all restaurant cards
	cards, err := database.GetRestaurantCards(restDb.UUID)
	if err != nil {
		c.JSON(403, gin.H{"error": "Unable to find any cards for your restaurant"})
		return
	}

	// How far back to go
	timeRange := c.Query("range")
	startTime := time.Now()
	if timeRange == "week" {
		for startTime.Weekday() != time.Sunday {
			startTime = startTime.AddDate(0, 0, -1)
		}
	}
	if timeRange == "month" {
		for startTime.Day() != 1 {
			startTime = startTime.AddDate(0, 0, -1)
		}
	}

	// Compute statistics //

	total := 0         // Total credit purchased in the given time range
	redeemed := 0      // Total credit redeemed in the given time range
	totalRedeemed := 0 // Total credit redeemed since restaurant signup
	outstanding := 0   // Total outstanding credit since restaurant signup

	reportTrans := []ReportTransaction{} // All redemptions in the given time range
	salesTrans := []ReportTransaction{}  // All credit purchases in the given time range

	for c := range cards {
		created, err := time.Parse(time.RFC3339, cards[c].Transactions[0].Timestamp)
		if err != nil {
			log.Info(err)
		}

		outstanding += cards[c].Transactions[0].Amount

		if !created.Before(startTime) {
			total += cards[c].Transactions[0].Amount
		}

		for t := range cards[c].Transactions {
			trans := cards[c].Transactions[t]
			transTime, err := time.Parse(time.RFC3339, trans.Timestamp)
			if err != nil {
				log.Info(err)
				continue
			}

			if t > 0 {
				totalRedeemed += (trans.Amount * -1)
			}

			if !transTime.Before(startTime) {
				if t > 0 {
					reportTrans = append(reportTrans, ReportTransaction{
						Timestamp: trans.Timestamp,
						Amount:    trans.Amount,
					})
					redeemed += trans.Amount
				} else if t == 0 {
					salesTrans = append(salesTrans, ReportTransaction{
						Timestamp: trans.Timestamp,
						Amount:    trans.Amount,
					})
				}
			}
		}
	}

	outstanding -= totalRedeemed
	employees := int(float32(total) * 0.25)
	restaurant := int(float32(total) * 0.75)

	c.JSON(200, RestaurantReport{
		Total:        total,
		Employees:    employees,
		Restaurant:   restaurant,
		Transactions: reportTrans,
		Sales:        salesTrans,
		Outstanding:  outstanding,
		Redeemed:     redeemed,
	})
}
