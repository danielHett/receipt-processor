package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func main() {
	var port string
	if len(os.Args) > 1 {
		if convPort, err := strconv.Atoi(os.Args[1]); err != nil || convPort <= 0 {
			panic("The port argument must be a nonnegative integer")
		}
		port = os.Args[1]
	} else {
		port = "8080"
	}

	db := createDB()
	router := chi.NewRouter()

	router.Post("/receipts/process", func(res http.ResponseWriter, req *http.Request) {
		processHandler(db, res, req)
	})

	router.Get("/receipts/{receiptId}/points", func(res http.ResponseWriter, req *http.Request) {
		pointsHandler(db, res, req)
	})

	fmt.Println("Starting on port " + port)
	err := http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatal(err)
	}
}
