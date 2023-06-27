package main

import (
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/hashicorp/go-memdb"
)

/**
* Creates a new instance of the database for the receipt processor.
 */
func createDB() *memdb.MemDB {
	var schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"receipt": &memdb.TableSchema{
				Name: "receipt",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.UUIDFieldIndex{Field: "Id"},
					},
					"points": &memdb.IndexSchema{
						Name:    "points",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "Points"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}

	return db
}

/*
* Given a receipt (ProcessRequest), use the rules provided in the challenge to
* compute the total points.
 */
func calculatePoints(processRequest *ProcessRequest) (int64, bool) {
	// Make sure all required fields are present.
	if processRequest.Retailer == nil ||
		processRequest.PurchaseDate == nil ||
		processRequest.PurchaseTime == nil ||
		processRequest.Items == nil ||
		processRequest.Total == nil {

		return 0, true
	}

	var total int64 = 0

	total += getRetailerPoints(processRequest.Retailer)

	// Get the points for the number of items on the receipt. If there are no
	// items return an error to the client.
	if len(*processRequest.Items) == 0 {
		return 0, true
	}

	// Rule: 5 points for every two items on the receipt.
	total += 5 * int64(len(*processRequest.Items)/2)

	points, err := getTotalPoints(processRequest.Total)
	if err {
		return 0, true
	}
	total += points

	for _, item := range *processRequest.Items {
		points, err := getItemPoints(&item)
		if err {
			return 0, true
		}
		total += points
	}

	points, err = getPurchaseTimePoints(processRequest.PurchaseTime)
	if err {
		return 0, true
	}
	total += points

	points, err = getPurchaseDayPoints(processRequest.PurchaseDate)
	if err {
		return 0, true
	}
	total += points

	return total, false
}

// Rule: One point for every alphanumeric character in the retailer name.
func getRetailerPoints(s *string) int64 {
	var count int64 = 0
	for _, c := range *s {
		if unicode.IsLetter(c) || unicode.IsNumber(c) {
			count++
		}
	}

	return count
}

// Rule: 50 points if the total is a round dollar amount with no cents.
// AND
// Rule: 25 points if the total is a multiple of 0.25.
func getTotalPoints(total *string) (int64, bool) {
	var points int64 = 0
	r := regexp.MustCompile(`^[0-9]+\.[0-9][0-9]$`)
	if !r.Match([]byte(*total)) {
		return 0, true
	}

	centsAmount, _ := strconv.Atoi(strings.Split(*total, ".")[1])
	if centsAmount == 0 {
		points += 50
	}

	if (centsAmount % 25) == 0 {
		points += 25
	}

	return points, false
}

// Rule: If the trimmed length of the item description is a multiple of 3, multiply the price
// by 0.2 and round up to the nearest integer. The result is the number of points earned.
func getItemPoints(item *Item) (int64, bool) {
	if item.ShortDescription == nil || item.Price == nil {
		return 0, true
	}

	trimmedDesc := strings.TrimSpace(*item.ShortDescription)
	if len(trimmedDesc)%3 != 0 {
		return 0, false
	}

	r := regexp.MustCompile(`^[0-9]+\.[0-9][0-9]$`)
	if !r.Match([]byte(*item.Price)) {
		return 0, true
	}

	price, _ := strconv.ParseFloat(*item.Price, 64)

	return int64(math.Ceil(0.2 * price)), false
}

// Rule: 6 points if the day in the purchase date is odd.
func getPurchaseDayPoints(date *string) (int64, bool) {
	r := regexp.MustCompile(`^\d{4}\-(0[1-9]|1[012])\-(0[1-9]|[12][0-9]|3[01])$`)
	if !r.Match([]byte(*date)) {
		return 0, true
	}

	day, _ := strconv.ParseInt(strings.Split(*date, "-")[2], 10, 64)
	if day%2 == 1 {
		return 6, false
	} else {
		return 0, false
	}
}

// Rule: 10 points if the time of purchase is after 2:00pm and before 4:00pm.
func getPurchaseTimePoints(time *string) (int64, bool) {
	r := regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)
	if !r.Match([]byte(*time)) {
		return 0, true
	}

	splitTime := strings.Split(*time, ":")
	hour, _ := strconv.ParseInt(splitTime[0], 10, 64)
	minute, _ := strconv.ParseInt(splitTime[1], 10, 64)

	isAfter2 := hour > 14 || (hour == 14 && minute > 0)
	isBefore4 := hour < 16

	if isAfter2 && isBefore4 {
		return 10, false
	} else {
		return 0, false
	}
}

// Sends a message back to the client.
func respond(code int, message []byte, res http.ResponseWriter) {
	res.WriteHeader(code)
	res.Write(message)
}
