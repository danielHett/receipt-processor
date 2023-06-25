package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
)

type StoredReceipt struct {
	Id     string
	Points int64
}

type ProcessRequest struct {
	Retailer     *string
	PurchaseDate *string
	PurchaseTime *string
	Items        *[]Item
	Total        *string
}

type ProcessResponse struct {
	Id string
}

type Item struct {
	ShortDescription *string
	Price            *string
}

type PointsResponse struct {
	Points int64
}

const InvalidBodyResponse = "The receipt is invalid"
const ReceiptNotFoundResponse = "No receipt found for that id"
const ServerErrorResponse = "Server error"

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

/**
* Handler for the /receipt/process path.
 */
func processHandler(db *memdb.MemDB, res http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)

	var processRequest = new(ProcessRequest)
	err = json.Unmarshal(body, processRequest)
	if err != nil {
		fmt.Println(err)
		// Something is wrong with the request body, send back correct code.
		respond(http.StatusBadRequest, []byte(InvalidBodyResponse), res)
		return
	}

	receiptID := uuid.New().String()
	// TODO: Add some logic here to calculate points.
	receiptPoints, code, err := calculatePoints(processRequest)

	if err != nil {
		fmt.Println("HERE")
		respond(code, []byte(err.Error()), res)
		return
	}

	txn := db.Txn(true)

	if err := txn.Insert("receipt", &StoredReceipt{receiptID, receiptPoints}); err != nil {
		fmt.Println(err)
		respond(http.StatusBadRequest, []byte(ServerErrorResponse), res)
		return
	}

	txn.Commit()

	var processReponse ProcessResponse
	processReponse.Id = receiptID

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(processReponse)

	return
}

func pointsHandler(db *memdb.MemDB, res http.ResponseWriter, req *http.Request) {
	receiptId, err := uuid.Parse(chi.URLParam(req, "receiptId"))
	if err != nil {
		respond(http.StatusNotFound, []byte(ReceiptNotFoundResponse), res)
		return
	}

	txn := db.Txn(false)
	raw, err := txn.First("receipt", "id", receiptId.String())
	fmt.Println(raw)
	if err != nil || raw == nil {
		respond(http.StatusNotFound, []byte(ReceiptNotFoundResponse), res)
		return
	}

	var resObj PointsResponse
	resObj.Points = raw.(*StoredReceipt).Points

	// Send response to the client.
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(resObj)

	return
}

func calculatePoints(processRequest *ProcessRequest) (int64, int, error) {
	if processRequest == nil {
		fmt.Println("1")
		return 0, 500, errors.New(ServerErrorResponse)
	}

	// Make sure all required fields are present.
	if processRequest.Retailer == nil ||
		processRequest.PurchaseDate == nil ||
		processRequest.PurchaseTime == nil ||
		processRequest.Items == nil ||
		processRequest.Total == nil {
		fmt.Println("2")
		return 0, 400, errors.New(InvalidBodyResponse)
	}

	var total int64 = 0

	total += getRetailerPoints(processRequest.Retailer)

	r := regexp.MustCompile("^[0-9]+\\.[0-9][0-9]$")
	if !r.Match([]byte(*processRequest.Total)) {
		fmt.Println("3")
		return 0, 400, errors.New(InvalidBodyResponse)
	}

	splitTotal := strings.Split(*processRequest.Total, ".")
	if len(splitTotal) != 2 {
		fmt.Println("4")
		return 0, 500, errors.New(ServerErrorResponse)
	}

	//dollarAmount := splitTotal[0]
	centsAmount, _ := strconv.Atoi(splitTotal[1])

	if centsAmount == 0 {
		total += 50
	}

	if (centsAmount % 25) == 0 {
		total += 25
	}

	total += 5 * int64(len(*processRequest.Items)/2)

	for _, item := range *processRequest.Items {
		points, err := getItemPoints(&item)
		if err != nil {
			return 0, 400, errors.New(InvalidBodyResponse)
		}
		total += points
	}

	points, err := getPurchaseTimePoints(processRequest.PurchaseTime)
	if err != nil {
		return 0, 400, errors.New(InvalidBodyResponse)
	}
	total += points

	points, err = getPurchaseDayPoints(processRequest.PurchaseDate)
	if err != nil {
		return 0, 400, errors.New(InvalidBodyResponse)
	}
	total += points

	return total, 0, nil
}

func getRetailerPoints(s *string) int64 {
	var count int64 = 0
	for _, c := range *s {
		if unicode.IsLetter(c) || unicode.IsNumber(c) {
			count++
		}
	}

	return count
}

func getItemPoints(item *Item) (int64, error) {
	if item.ShortDescription == nil || item.Price == nil {
		return 0, errors.New(InvalidBodyResponse)
	}

	trimmedDesc := strings.TrimSpace(*item.ShortDescription)
	if len(trimmedDesc)%3 != 0 {
		return 0, nil
	}

	// TODO: Turn this logic into a function.
	r := regexp.MustCompile("^[0-9]+\\.[0-9][0-9]$")
	if !r.Match([]byte(*item.Price)) {
		return 0, errors.New(InvalidBodyResponse)
	}

	price, _ := strconv.ParseFloat(*item.Price, 64)

	return int64(math.Ceil(0.2 * price)), nil
}

func getPurchaseDayPoints(date *string) (int64, error) {
	r := regexp.MustCompile("\\d{4}\\-(0[1-9]|1[012])\\-(0[1-9]|[12][0-9]|3[01])")
	if !r.Match([]byte(*date)) {
		return 0, errors.New(InvalidBodyResponse)
	}

	day, _ := strconv.ParseInt(strings.Split(*date, "-")[2], 10, 64)

	if day%2 == 1 {
		return 6, nil
	} else {
		return 0, nil
	}
}

func getPurchaseTimePoints(time *string) (int64, error) {
	r := regexp.MustCompile("^([01]\\d|2[0-3]):([0-5]\\d)$")
	if !r.Match([]byte(*time)) {
		return 0, errors.New(InvalidBodyResponse)
	}

	splitTime := strings.Split(*time, ":")
	hour, _ := strconv.ParseInt(splitTime[0], 10, 64)
	minute, _ := strconv.ParseInt(splitTime[1], 10, 64)

	fmt.Println(hour)
	fmt.Println(minute)

	isAfter2 := hour > 14 || (hour == 14 && minute > 0)
	isBefore4 := hour < 16

	if isAfter2 && isBefore4 {
		return 10, nil
	} else {
		return 0, nil
	}
}

// Helper function for responding to the client.
func respond(code int, message []byte, res http.ResponseWriter) {
	res.WriteHeader(code)
	res.Write(message)
	return
}

// test 10
// test $10.55
// test 100.544
