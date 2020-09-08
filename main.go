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
}

func main() {
	createJar("cars7")

	http.HandleFunc("/getOrders", getOrders)
	http.HandleFunc("/getCars", getCars)
	http.HandleFunc("/getFine", getFine)
	//http.HandleFunc("/getOrders", getOrders)
	//http.HandleFunc("/getOrders", getOrders)
	//http.HandleFunc("/getOrders", getOrders)
	fmt.Println(http.ListenAndServe(":49200", nil))
}

func login() {
	client := getClient(false)
	resp, err := client.Get("http://lk.cars7.ru/Account/LoginApp?login=%D0%A2%D0%B0%D1%82%D0%B5%D0%B2%D0%BE%D1%81%D1%8F%D0%BD&password=Haiastan1987!")
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
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

func getDataCSV(params *Params) []byte {
	login()
	client := getClient(false)
	startDate := params.formatTime(params.StartDate, "2006-01-02T15:04:05Z")
	endDate := params.formatTime(params.EndDate, "2006-01-02T15:04:05Z")

	fmt.Println(startDate)
	formData := url.Values{}
	formData.Set("DateStart", startDate)
	formData.Set("DateEnd", endDate)
	formData.Set("Car", "")

	resp, err := client.PostForm("http://lk.cars7.ru/Data/ExportToCsv?type="+params.Category+"&category=0", formData)
	if err != nil {
		fmt.Println(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
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
	resp, err = client.Get("http://lk.cars7.ru/Data/GetFileCsv?url=" + url.QueryEscape(file.File))
	if err != nil {
		fmt.Println(err)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	return body
}
