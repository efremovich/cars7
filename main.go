package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
)

var coockie = make(map[string]*cookiejar.Jar)

type Params struct {
	StartDate  string `json:"start_date,omitempty"`
	EndDate    string `json:"end_date,omitempty"`
	Category   string `json:"category,omitempty"`
	Car        string `json:"car,omitempty"`
	Password   string `json:"password,omitempty"`
	Login      string `json:"login,omitempty"`
	URLstring  string `json:"url_string,omitempty"`
	CompenceID string `json:"compence_id,omitempty"`
}

func main() {
	createJar("cars7")
	http.HandleFunc("/getOrders", getOrders)
	http.HandleFunc("/getCars", getCars)
	http.HandleFunc("/getFine", getFine)
	http.HandleFunc("/getRefills", getRefills)
	http.HandleFunc("/getCompence", cars7Compence)
	http.HandleFunc("/getCompenceFile", cars7GetDocement)
	http.HandleFunc("/createOrUpdateCompence", cars7CreateOrUpdateDocement)
	http.HandleFunc("/getBonus", getBonus)

	http.HandleFunc("/fort/mileage", getMileAge)
	http.HandleFunc("/fort/cars", getFortCars)

	http.HandleFunc("/zont/mileage", getZontMileAge)
	http.HandleFunc("/zont/cars", getZontCars)

	http.HandleFunc("/ccars/mileage", getMileAgeCcar)
	http.HandleFunc("/ccars/cars", getCCars)

	fmt.Println(http.ListenAndServe(":49200", nil))
}
