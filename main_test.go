package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestProcessHandler_ThrowsOnNoBody(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", nil)
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Error("Getting an incorrect status code")
	}
}

func TestProcessHandler_ThrowsOnMissingField(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "M&M Corner Market",
		"purchaseDate": "2022-03-20",
		"items": [
		  {
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  }
		],
		"total": "9.00"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Error("Getting an incorrect status code")
	}
}

func TestProcessHandler_ThrowsInvalidPrice(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "M&M Corner Market",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "14:33",
		"items": [
		  {
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  }
		],
		"total": "9.525"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Error("Getting an incorrect status code")
	}
}

func TestProcessHandler_ThrowsInvalidDate(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "M&M Corner Market",
		"purchaseDate": "2022-03-200",
		"purchaseTime": "14:33",
		"items": [
		  {
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  }
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Error("Getting an incorrect status code")
	}
}

func TestProcessHandler_ThrowsNoItems(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "M&M Corner Market",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "14:33",
		"items": [
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Error("Getting an incorrect status code")
	}
}

func TestProcessHandler_ThrowsInvalidTime(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "M&M Corner Market",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "25:33",
		"items": [
		  {
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  },{
			"shortDescription": "Gatorade",
			"price": "2.25"
		  }
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Error("Getting an incorrect status code")
	}
}

func TestProcessHandler_CorrectRetailerPoints(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "Daniel-Special-Shop   ",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "01:33",
		"items": [
		  {
			"shortDescription": "Hi",
			"price": "2.25"
		  }
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("Recieved error while reading body")
	}

	var processResponse ProcessResponse
	err = json.Unmarshal(data, &processResponse)
	if err != nil {
		t.Error("Recieved error while unmarshaling")
	}

	_, err = uuid.Parse(processResponse.Id)
	if err != nil {
		t.Error("Reponse body was not a valid UUID")
	}

	txn := testDB.Txn(false)
	raw, err := txn.First("receipt", "id", processResponse.Id)
	if err != nil || raw == nil {
		t.Error("UUID not found in the DB")
	}

	if raw.(*StoredReceipt).Points != 17 {
		t.Error("Invalid points")
	}
}

func TestProcessHandler_CorrectDatePoints(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "-",
		"purchaseDate": "2022-03-21",
		"purchaseTime": "14:00",
		"items": [
		  {
			"shortDescription": "Hi",
			"price": "2.25"
		  }
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("Recieved error while reading body")
	}

	var processResponse ProcessResponse
	err = json.Unmarshal(data, &processResponse)
	if err != nil {
		t.Error("Recieved error while unmarshaling")
	}

	_, err = uuid.Parse(processResponse.Id)
	if err != nil {
		t.Error("Reponse body was not a valid UUID")
	}

	txn := testDB.Txn(false)
	raw, err := txn.First("receipt", "id", processResponse.Id)
	if err != nil || raw == nil {
		t.Error("UUID not found in the DB")
	}

	if raw.(*StoredReceipt).Points != 6 {
		t.Error("Invalid points")
	}
}

func TestProcessHandler_CorrectTimePoints(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "-",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "14:01",
		"items": [
		  {
			"shortDescription": "Hi",
			"price": "2.25"
		  }
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("Recieved error while reading body")
	}

	var processResponse ProcessResponse
	err = json.Unmarshal(data, &processResponse)
	if err != nil {
		t.Error("Recieved error while unmarshaling")
	}

	_, err = uuid.Parse(processResponse.Id)
	if err != nil {
		t.Error("Reponse body was not a valid UUID")
	}

	txn := testDB.Txn(false)
	raw, err := txn.First("receipt", "id", processResponse.Id)
	if err != nil || raw == nil {
		t.Error("UUID not found in the DB")
	}

	if raw.(*StoredReceipt).Points != 10 {
		t.Error("Invalid points")
	}
}

func TestProcessHandler_CorrectNumberOfItemsPoints(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "-",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "16:00",
		"items": [
		  {
			"shortDescription": "Hi",
			"price": "2.25"
		  },
		  {
			"shortDescription": "Hi",
			"price": "2.25"
		  }
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("Recieved error while reading body")
	}

	var processResponse ProcessResponse
	err = json.Unmarshal(data, &processResponse)
	if err != nil {
		t.Error("Recieved error while unmarshaling")
	}

	_, err = uuid.Parse(processResponse.Id)
	if err != nil {
		t.Error("Reponse body was not a valid UUID")
	}

	txn := testDB.Txn(false)
	raw, err := txn.First("receipt", "id", processResponse.Id)
	if err != nil || raw == nil {
		t.Error("UUID not found in the DB")
	}

	if raw.(*StoredReceipt).Points != 5 {
		t.Error("Invalid points")
	}
}

func TestProcessHandler_CorrectItemsPoints(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "-",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "16:00",
		"items": [
		  {
			"shortDescription": "Thr ",
			"price": "10.00"
		  }
		],
		"total": "9.52"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("Recieved error while reading body")
	}

	var processResponse ProcessResponse
	err = json.Unmarshal(data, &processResponse)
	if err != nil {
		t.Error("Recieved error while unmarshaling")
	}

	_, err = uuid.Parse(processResponse.Id)
	if err != nil {
		t.Error("Reponse body was not a valid UUID")
	}

	txn := testDB.Txn(false)
	raw, err := txn.First("receipt", "id", processResponse.Id)
	if err != nil || raw == nil {
		t.Error("UUID not found in the DB")
	}

	if raw.(*StoredReceipt).Points != 2 {
		t.Error("Invalid points")
	}
}

func TestProcessHandler_CorrectTotalPoints(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "-",
		"purchaseDate": "2022-03-20",
		"purchaseTime": "16:00",
		"items": [
		  {
			"shortDescription": "Th",
			"price": "10.00"
		  }
		],
		"total": "10.00"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("Recieved error while reading body")
	}

	var processResponse ProcessResponse
	err = json.Unmarshal(data, &processResponse)
	if err != nil {
		t.Error("Recieved error while unmarshaling")
	}

	_, err = uuid.Parse(processResponse.Id)
	if err != nil {
		t.Error("Reponse body was not a valid UUID")
	}

	txn := testDB.Txn(false)
	raw, err := txn.First("receipt", "id", processResponse.Id)
	if err != nil || raw == nil {
		t.Error("UUID not found in the DB")
	}

	if raw.(*StoredReceipt).Points != 75 {
		t.Error("Invalid points")
	}
}

func TestProcessHandler_CorrectForRealReceipt(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "Target",
		"purchaseDate": "2022-01-01",
		"purchaseTime": "13:01",
		"items": [
		  {
			"shortDescription": "Mountain Dew 12PK",
			"price": "6.49"
		  },{
			"shortDescription": "Emils Cheese Pizza",
			"price": "12.25"
		  },{
			"shortDescription": "Knorr Creamy Chicken",
			"price": "1.26"
		  },{
			"shortDescription": "Doritos Nacho Cheese",
			"price": "3.35"
		  },{
			"shortDescription": "   Klarbrunn 12-PK 12 FL OZ  ",
			"price": "12.00"
		  }
		],
		"total": "35.35"
	  }`))
	w := httptest.NewRecorder()
	processHandler(testDB, w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("Recieved error while reading body")
	}

	var processResponse ProcessResponse
	err = json.Unmarshal(data, &processResponse)
	if err != nil {
		t.Error("Recieved error while unmarshaling")
	}

	_, err = uuid.Parse(processResponse.Id)
	if err != nil {
		t.Error("Reponse body was not a valid UUID")
	}

	txn := testDB.Txn(false)
	raw, err := txn.First("receipt", "id", processResponse.Id)
	if err != nil || raw == nil {
		t.Error("UUID not found in the DB")
	}

	if raw.(*StoredReceipt).Points != 28 {
		t.Error("Invalid points")
	}
}

func TestPointsHandler_ReturnsPoints(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodPost, "/receipts/process", strings.NewReader(`{
		"retailer": "Target",
		"purchaseDate": "2022-01-01",
		"purchaseTime": "13:01",
		"items": [
		  {
			"shortDescription": "Mountain Dew 12PK",
			"price": "6.49"
		  },{
			"shortDescription": "Emils Cheese Pizza",
			"price": "12.25"
		  },{
			"shortDescription": "Knorr Creamy Chicken",
			"price": "1.26"
		  },{
			"shortDescription": "Doritos Nacho Cheese",
			"price": "3.35"
		  },{
			"shortDescription": "   Klarbrunn 12-PK 12 FL OZ  ",
			"price": "12.00"
		  }
		],
		"total": "35.35"
	  }`))
	firstW := httptest.NewRecorder()
	processHandler(testDB, firstW, req)
	firstRes := firstW.Result()
	defer firstRes.Body.Close()
	data, _ := ioutil.ReadAll(firstRes.Body)
	var processResponse ProcessResponse
	json.Unmarshal(data, &processResponse)
	req = httptest.NewRequest(http.MethodGet, "/receipts/"+processResponse.Id+"/points", nil)
	secondW := httptest.NewRecorder()
	pointsHandler(testDB, secondW, req)
	secondRes := secondW.Result()
	if secondRes.StatusCode != 200 {
		t.Error("Getting an incorrect status code")
	}
	data, _ = ioutil.ReadAll(secondRes.Body)
	var pointsResponse PointsResponse
	json.Unmarshal(data, &pointsResponse)
	if pointsResponse.Points != 28 {
		t.Error("Invalid points")
	}
}

func TestPointsHandler_FailsOnInvalidId(t *testing.T) {
	testDB := createDB()
	req := httptest.NewRequest(http.MethodGet, "/receipts/8e55ce4d-4d0d-4765-babd-294711efc91b/points", nil)
	w := httptest.NewRecorder()
	pointsHandler(testDB, w, req)
	res := w.Result()
	if res.StatusCode != 404 {
		t.Error("Getting an incorrect status code")
	}
}
