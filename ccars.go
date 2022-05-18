package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

type respCar struct {
	Oid  string `json:"oid"`
	IMEI string `json:"imei"`
	Name string `json:"Name"`
}
type respCcarCar struct {
	Rows []struct {
		Vin          string `json:"vin"`
		LicencePlate string `json:"licencePlate"`
		Brand        string `json:"brand"`
		Model        string `json:"model"`
		VehicleID    string `json:"vehicle_id"`
		DeviceID     string `json:"device_id"`
		Esn          string `json:"esn"`
		ID           string `json:"id"`
	} `json:"rows"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalPages int `json:"totalPages"`
}

type respToken struct {
	Token        string
	refreshToken string
}

type CarsCCar struct {
	Cars     []string `json:"cars"`
	TimeFrom string   `json:"timeFrom"`
	TimeTo   string   `json:"timeTo"`
}

var tokens map[string]*respToken

type CCarsMileage struct {
	Cars []struct {
		ID      string      `json:"_id"`
		Details interface{} `json:"details"`
	} `json:"cars"`
}

type DataMileAge struct {
	Mileage int     `json:"mileage"`
	InRoute float64 `json:"inRoute"`
	Stops   int     `json:"stops"`
}

// func (p *CCarsMileage) UnmarshalJSON(data []byte) (err error) {
// 	var result CCarsMileage
// 	err = json.Unmarshal(data, &result)
// 	keys := reflect.ValueOf(result).MapKeys()
// 	for _, item := range keys {
// 		fmt.Println(item)
// 	}
// 	return

// }

func getMileAgeCcar(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	loginCcars(&params)

	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startDate := firstOfMonth
	endDate := lastOfMonth

	layout := "2006-01-02T15:04:05 -0700"
	if params.StartDate != "" {
		startDate, err = time.Parse(layout, params.StartDate+" +0300")
		if err != nil {
			log.Println(err)
		}
	}
	if params.EndDate != "" {
		endDate, err = time.Parse(layout, params.EndDate+" +0300")
		if err != nil {
			log.Println(err)
		}
	}
	carIDs := getccCars(params)
	cars := CarsCCar{
		TimeFrom: startDate.Format(layout),
		TimeTo:   endDate.Format(layout),
	}
	cars.Cars = append(cars.Cars, carIDs...)
	carsJson, err := json.Marshal(cars)

	body = getCCarsApi(&params, "POST", "/api/v1/efficiency-management/report/routes-and-stops", string(carsJson), url.Values{})
	ccMileAge := make(map[string]interface{})

	err = json.Unmarshal([]byte(body), &ccMileAge)

	body, err = json.Marshal(body)

	w.Write(body)

}

func getCCars(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	loginCcars(&params)

	body = getCCarsApi(&params, "GET", "/api/v1/vehicle-management/cars", "", url.Values{})

	rrespCar := respCcarCar{}

	err = json.Unmarshal([]byte(body), &rrespCar)
	if err != nil {
		fmt.Println(err, "error unmashal")
	}

	carsData := []respCar{}
	for _, car := range rrespCar.Rows {

		carData := respCar{}
		carData.Oid = car.ID
		carData.IMEI = car.Vin
		carData.Name = car.LicencePlate
		carsData = append(carsData, carData)

	}
	body, err = json.Marshal(carsData)
	w.Write(body)

}

func loginCcars(params *Params) {

	if tokens == nil {
		tokens = make(map[string]*respToken)
		tokens[params.Password] = &respToken{}
	}

	client := getClient(false, params.Password)
	_, err := client.Get(params.URLstring + "/login")
	if err != nil {
		fmt.Println(err, "Ошибка авторизации fort-monitor")
	}

	data := url.Values{}
	data.Set("username", params.Login)
	data.Set("password", params.Password)
	putData := `{"email":"%v","password":"%v"}`
	body := getCCarsApi(params, "POST", "/api/v1/user-management/login", fmt.Sprintf(putData, params.Login, params.Password), url.Values{})
	rToken := respToken{}
	err = json.Unmarshal(body, &rToken)
	if err != nil {
		fmt.Println(err, "error unmashal")
	}

	tokens[params.Password] = &rToken

}
func getccCars(params Params) []string {
	type me struct {
		ID string `json:"_id"`
	}

	loginCcars(&params)
	body := getCCarsApi(&params, "GET", "/api/v1/user-management/me", "", url.Values{})
	mei := me{}
	err := json.Unmarshal(body, &mei)
	if err != nil {
		fmt.Println(err, "error unmashal")
	}

	body = getCCarsApi(&params, "GET", fmt.Sprintf("/api/v1/user-management/users/%v", mei.ID), "", url.Values{})
	ncars := CarsCCar{}
	err = json.Unmarshal(body, &ncars)
	if err != nil {
		fmt.Println(err, "error unmashal")
	}

	return ncars.Cars
}

func getCCarsApi(params *Params, method, urlPath, putData string, data url.Values) []byte {
	contentType := "application/json"

	client := getClient(false, params.Password)
	u := params.URLstring + urlPath
	var rbody io.Reader
	if method == "GET" {
		u += "?" + data.Encode()
	} else if len(putData) > 0 {
		rbody = strings.NewReader(putData)
	} else {
		rbody = strings.NewReader(data.Encode())
	}

	req, err := http.NewRequest(method, u, rbody)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", contentType)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokens[params.Password].Token))

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Println(string(body))
	}
	return body
}
