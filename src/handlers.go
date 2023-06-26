package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

const (
	InvalidBodyResponse     = "The receipt is invalid"
	ReceiptNotFoundResponse = "No receipt found for that id"
	ServerErrorResponse     = "Server error"
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

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(processReponse)
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

	var pointsResponse PointsResponse
	pointsResponse.Points = raw.(*StoredReceipt).Points

	// Send response to the client.
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(pointsResponse)
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
