package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	db := createDB()
	router := chi.NewRouter()

	router.Post("/receipts/process", func(res http.ResponseWriter, req *http.Request) {
		processHandler(db, res, req)
	})

	router.Get("/receipts/{receiptId}/points", func(res http.ResponseWriter, req *http.Request) {
		pointsHandler(db, res, req)
	})

	//Use the default DefaultServeMux.
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal(err)
	}
}
