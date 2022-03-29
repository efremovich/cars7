package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	http.HandleFunc("/getCompence", getCompence)
	http.HandleFunc("/getBonus", getBonus)

	http.HandleFunc("/fort/mileage", getMileAge)
	http.HandleFunc("/fort/cars", getFortCars)

	fmt.Println(http.ListenAndServe(":49200", nil))
}
func login(params *Params) {
	client := getClient(false, "cars7")
	queryLoqin := url.QueryEscape(params.Login)
	loginURL := "http://lk.cars7.ru/Account/LoginApp?login=" + queryLoqin + "&password=" + params.Password
	fmt.Println(loginURL)
	resp, err := client.Get(loginURL)

	if err != nil {
		fmt.Println(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	params.Category = "2"
	body = getDataCSV(&params)
	w.Write(body)
}

func getCars(w http.ResponseWriter, r *http.Request) {

	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	params.Category = "4"
	body = getDataCSV(&params)
	w.Write(body)
}

func getFine(w http.ResponseWriter, r *http.Request) {

	var params Params

	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	params.Category = "8"
	body = getDataCSV(&params)
	w.Write(body)
}

func getRefills(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	params.Category = "12"
	body = getDataCSV(&params)
	w.Write(body)
}

func getCompence(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	params.Category = "9"
	body = getDataCSV(&params)
	w.Write(body)
}

func getBonus(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	params.Category = "10"
	body = getDataCSV(&params)
	w.Write(body)
}

func getDataCSV(params *Params) []byte {
	login(params)
	client := getClient(false, "cars7")
	startDate := params.formatTime(params.StartDate, "2006-01-02T15:04:05Z")
	endDate := params.formatTime(params.EndDate, "2006-01-02T15:04:05Z")
	fmt.Printf("Запрос транзакций с %s по %s", startDate, endDate)
	formData := url.Values{}
	formData.Set("DateStart", startDate)
	formData.Set("DateEnd", endDate)
	formData.Set("Car", "")

	resp, err := client.PostForm(
		"http://lk.cars7.ru/Export/ExportToCsv?type="+params.Category+"&category=0&timezone=-3",
		formData,
	)
	if err != nil {
		fmt.Println(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	type fileStruct struct {
		FileName string `json:"FileName"`
		File     string `json:"File"`
	}

	var file fileStruct
	err = json.Unmarshal(body, &file)
	if err != nil {
		fmt.Printf("Ошибка преобразования json: %v", err)
	}
	resp, err = client.Get("http://lk.cars7.ru/Export/GetFileCsv?url=" + url.QueryEscape(file.File))
	if err != nil {
		fmt.Println(err)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	return body
}
