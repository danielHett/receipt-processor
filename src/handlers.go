package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

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
	Id string `json:"id"`
}

type Item struct {
	ShortDescription *string
	Price            *string
}

type PointsResponse struct {
	Points int64 `json:"points"`
}

const (
	InvalidBodyResponse     = "The receipt is invalid"
	ReceiptNotFoundResponse = "No receipt found for that id"
	ServerErrorResponse     = "Server error"
)

/**
* Handler for the /receipt/process path.
 */
func processHandler(db *memdb.MemDB, res http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		respond(http.StatusBadRequest, []byte(ServerErrorResponse), res)
		return
	}

	var processRequest = new(ProcessRequest)
	err = json.Unmarshal(body, processRequest)
	if err != nil {
		respond(http.StatusBadRequest, []byte(InvalidBodyResponse), res)
		return
	}

	receiptID := uuid.New().String()
	receiptPoints, isIncorrectBody := calculatePoints(processRequest)
	if isIncorrectBody {
		respond(400, []byte(InvalidBodyResponse), res)
		return
	}

	txn := db.Txn(true)
	err = txn.Insert("receipt", &StoredReceipt{receiptID, receiptPoints})
	if err != nil {
		respond(http.StatusBadRequest, []byte(ServerErrorResponse), res)
		return
	}
	txn.Commit()

	var processReponse ProcessResponse
	processReponse.Id = receiptID
	jData, err := json.Marshal(processReponse)
	if err != nil {
		respond(http.StatusBadRequest, []byte(ServerErrorResponse), res)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(jData)
}

func pointsHandler(db *memdb.MemDB, res http.ResponseWriter, req *http.Request) {
	urlParts := strings.Split(req.URL.String(), "/")
	receiptId := urlParts[2]
	_, err := uuid.Parse(receiptId)
	if err != nil {
		respond(http.StatusNotFound, []byte(ReceiptNotFoundResponse), res)
		return
	}

	txn := db.Txn(false)
	raw, err := txn.First("receipt", "id", receiptId)
	if err != nil || raw == nil {
		respond(http.StatusNotFound, []byte(ReceiptNotFoundResponse), res)
		return
	}

	var pointsResponse PointsResponse
	pointsResponse.Points = raw.(*StoredReceipt).Points
	jData, err := json.Marshal(pointsResponse)
	if err != nil {
		respond(http.StatusBadRequest, []byte(ServerErrorResponse), res)
		return
	}

	// Send response to the client.
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(jData)
}

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

	// Get the points for the number of items on the receipt.
	if len(*processRequest.Items) == 0 {
		return 0, true
	}
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
