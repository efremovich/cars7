package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
)

var coockie = make(map[string]*cookiejar.Jar)

type Params struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Category  string `json:"category"`
	Car       string `json:"car"`
	Password  string `json:"password"`
	Login     string `json:"login"`
	URLstring string `json:"url_string"`
}

func main() {
	createJar("cars7")
	http.HandleFunc("/getOrders", getOrders)
	http.HandleFunc("/getCars", getCars)
	http.HandleFunc("/getFine", getFine)
	http.HandleFunc("/getRefills", getRefills)
	http.HandleFunc("/getCompence", cars7Compence)
	http.HandleFunc("/getBonus", getBonus)

	http.HandleFunc("/fort/mileage", getMileAge)
	http.HandleFunc("/fort/cars", getFortCars)

	http.HandleFunc("/zont/mileage", getZontMileAge)
	http.HandleFunc("/zont/cars", getZontCars)

	http.HandleFunc("/ccars/mileage", getMileAgeCcar)
	http.HandleFunc("/ccars/cars", getCCars)

	fmt.Println(http.ListenAndServe(":49200", nil))
}
