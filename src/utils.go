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
	Retailer     string
	PurchaseDate string
	PurchaseTime string
	Items        []Item
	Total        string
}

type ProcessResponse struct {
	Id string
}

type Item struct {
	ShortDescription string
	Price            string
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

	var processRequest ProcessRequest
	err = json.Unmarshal(body, &processRequest)
	if err != nil {
		// Something is wrong with the request body, send back correct code.
		respond(http.StatusBadRequest, []byte(InvalidBodyResponse), res)
		return
	}

	receiptID := uuid.New().String()
	// TODO: Add some logic here to calculate points.
	var receiptPoints int64 = 0

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
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(resObj)

	return
}

// Helper function for responding to the client.
func respond(code int, message []byte, res http.ResponseWriter) {
	res.WriteHeader(code)
	res.Write(message)
	return
}
