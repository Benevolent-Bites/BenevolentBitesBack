package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rishabh-bector/BenevolentBitesBack/auth"
	"github.com/rishabh-bector/BenevolentBitesBack/database"
	"github.com/rishabh-bector/BenevolentBitesBack/email"

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
	startTime := FindStartOf(c.Query("range"))

	// Compute statistics //

	restStats := CalcStats(startTime, cards)
	restTrans := CalcTrans(startTime, cards)

	employees := int(float32(restStats.Total) * 0.25)
	restaurant := int(float32(restStats.Total) * 0.75)

	c.JSON(200, RestaurantReport{
		Total:        restStats.Total,
		Employees:    employees,
		Restaurant:   restaurant,
		Transactions: restTrans.Redeems,
		Sales:        restTrans.Sales,
		Outstanding:  restStats.Outstanding,
		Redeemed:     restStats.Redeemed,
	})
}

type EmployeeReport struct {
	Week   time.Time
	Amount int
}

func StartEmployeeReportLoop() {
	ticker := time.NewTicker(6 * time.Hour)
	go employeeReportLoop(ticker)
}

func employeeReportLoop(ticker *time.Ticker) {
	for {
		select {
		case _ = <-ticker.C:
			if time.Now().Weekday() == time.Saturday {
				SendEmployeeReports()
				time.Sleep(48 * time.Hour)
			}
		}
	}
}

func SendEmployeeReports() {
	rests := database.GetAllPublishedRestaurants()

	for r := range rests {
		rest := rests[r]
		report := CreateEmployeeReport(&rest)

		// Don't send email if amount is 0
		if report.Amount == 0 {
			continue
		}

		for a := range rest.Employees {
			email.SendEmail(
				[]string{rest.Employees[a]["email"]},
				fmt.Sprintf("Benevolent Bites Weekly Report: %s", report.Week.Format("01-02-2006")),
				fmt.Sprintf(email.ReportFormat, rest.Employees[a]["name"], float32(report.Amount)/100, rest.Name),
			)
		}
	}
}

func CreateEmployeeReport(rest *database.Restaurant) EmployeeReport {
	// Get all restaurant cards
	cards, err := database.GetRestaurantCards(rest.UUID)
	if err != nil {
		return EmployeeReport{}
	}

	startTime := FindStartOf("week")
	restStats := CalcStats(startTime, cards)

	individualAmnt := int(float32(restStats.Total)*0.25) / int(len(rest.Employees))

	return EmployeeReport{
		Week:   startTime,
		Amount: individualAmnt,
	}
}

type RestStats struct {
	Total         int // Total credit purchased in the given time range
	Redeemed      int // Total credit redeemed in the given time range
	TotalRedeemed int // Total credit redeemed since restaurant signup
	Outstanding   int // Total outstanding credit since restaurant signup
}

func CalcStats(startTime time.Time, cards []database.Card) RestStats {
	total := 0
	redeemed := 0
	totalRedeemed := 0
	outstanding := 0

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
					redeemed += trans.Amount
				}
			}
		}
	}

	outstanding -= totalRedeemed

	return RestStats{
		Total:         total,
		Redeemed:      redeemed,
		TotalRedeemed: totalRedeemed,
		Outstanding:   outstanding,
	}
}

type RestTransactions struct {
	Redeems []ReportTransaction // All redemptions in the given time range
	Sales   []ReportTransaction // All credit purchases in the given time range
}

func CalcTrans(startTime time.Time, cards []database.Card) RestTransactions {
	reportTrans := []ReportTransaction{} // All redemptions in the given time range
	salesTrans := []ReportTransaction{}  // All credit purchases in the given time range

	for c := range cards {
		for t := range cards[c].Transactions {
			trans := cards[c].Transactions[t]
			transTime, err := time.Parse(time.RFC3339, trans.Timestamp)
			if err != nil {
				log.Info(err)
				continue
			}

			if !transTime.Before(startTime) {
				if t > 0 {
					reportTrans = append(reportTrans, ReportTransaction{
						Timestamp: trans.Timestamp,
						Amount:    trans.Amount,
					})
				} else if t == 0 {
					salesTrans = append(salesTrans, ReportTransaction{
						Timestamp: trans.Timestamp,
						Amount:    trans.Amount,
					})
				}
			}
		}
	}

	return RestTransactions{
		Redeems: reportTrans,
		Sales:   salesTrans,
	}
}

func FindStartOf(timeRange string) time.Time {
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
	return startTime
}
