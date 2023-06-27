/**
* This file contains the handlers for the two server endpoints. All helper code is
* defined seperately in the utils.go file.
 */

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
)

// Key-Value pair that is stored in the DB.
type StoredReceipt struct {
	Id     string
	Points int64
}

// Below two model incoming requests to the receipts/process endpoint. It represents a receipt.
type ProcessRequest struct {
	Retailer     *string
	PurchaseDate *string
	PurchaseTime *string
	Items        *[]Item
	Total        *string
}

type Item struct {
	ShortDescription *string
	Price            *string
}

// Models a response to the receipts/process endpoint.
type ProcessResponse struct {
	Id string `json:"id"`
}

// Models a response to the receipts/{id}/points endpoint.
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

	// Create a random id for the receipt, get the points from the body.
	receiptID := uuid.New().String()
	receiptPoints, isErr := calculatePoints(processRequest)
	if isErr {
		respond(400, []byte(InvalidBodyResponse), res)
		return
	}

	// Set up a transaction and store the id, points as a key-val pair in the DB.
	txn := db.Txn(true)
	err = txn.Insert("receipt", &StoredReceipt{receiptID, receiptPoints})
	if err != nil {
		respond(http.StatusBadRequest, []byte(ServerErrorResponse), res)
		return
	}
	txn.Commit()

	// Send the id back to the client.
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

/**
* Handler for the /receipt/{id}/points path.
 */
func pointsHandler(db *memdb.MemDB, res http.ResponseWriter, req *http.Request) {
	// Get the id from the path.
	urlParts := strings.Split(req.URL.String(), "/")
	receiptId := urlParts[2]
	_, err := uuid.Parse(receiptId)
	if err != nil {
		// There wasn't a valid uuid.
		respond(http.StatusNotFound, []byte(ReceiptNotFoundResponse), res)
		return
	}

	// Try retrieving the points from the db.
	txn := db.Txn(false)
	raw, err := txn.First("receipt", "id", receiptId)
	if err != nil || raw == nil {
		// Couldn't find the id.
		respond(http.StatusNotFound, []byte(ReceiptNotFoundResponse), res)
		return
	}

	// Put the retrieved points in a response.
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
