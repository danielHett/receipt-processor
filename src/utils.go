package main

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func getRetailerPoints(s *string) int64 {
	var count int64 = 0
	for _, c := range *s {
		if unicode.IsLetter(c) || unicode.IsNumber(c) {
			count++
		}
	}

	return count
}

func getTotalPoints(total *string) (int64, bool) {
	var points int64 = 0
	r := regexp.MustCompile("^[0-9]+\\.[0-9][0-9]$")
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

func getItemPoints(item *Item) (int64, bool) {
	if item.ShortDescription == nil || item.Price == nil {
		return 0, true
	}

	trimmedDesc := strings.TrimSpace(*item.ShortDescription)
	if len(trimmedDesc)%3 != 0 {
		return 0, false
	}

	// TODO: Turn this logic into a function.
	r := regexp.MustCompile("^[0-9]+\\.[0-9][0-9]$")
	if !r.Match([]byte(*item.Price)) {
		return 0, true
	}

	price, _ := strconv.ParseFloat(*item.Price, 64)

	return int64(math.Ceil(0.2 * price)), false
}

func getPurchaseDayPoints(date *string) (int64, bool) {
	r := regexp.MustCompile("\\d{4}\\-(0[1-9]|1[012])\\-(0[1-9]|[12][0-9]|3[01])")
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

func getPurchaseTimePoints(time *string) (int64, bool) {
	r := regexp.MustCompile("^([01]\\d|2[0-3]):([0-5]\\d)$")
	if !r.Match([]byte(*time)) {
		return 0, true
	}

	splitTime := strings.Split(*time, ":")
	hour, _ := strconv.ParseInt(splitTime[0], 10, 64)
	minute, _ := strconv.ParseInt(splitTime[1], 10, 64)

	fmt.Println(hour)
	fmt.Println(minute)

	isAfter2 := hour > 14 || (hour == 14 && minute > 0)
	isBefore4 := hour < 16

	if isAfter2 && isBefore4 {
		return 10, false
	} else {
		return 0, false
	}
}

func respond(code int, message []byte, res http.ResponseWriter) {
	res.WriteHeader(code)
	res.Write(message)
}
