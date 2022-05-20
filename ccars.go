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
	Token string
}

type CarsCCar struct {
	Cars     []string `json:"cars"`
	TimeFrom string   `json:"timeFrom"`
	TimeTo   string   `json:"timeTo"`
}

var tokens map[string]*respToken

type CCarsMileage []struct {
	ID           string `json:"_id"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	LicencePlate string `json:"licencePlate"`
	Vin          string `json:"vin"`
	Oid          string `json:"vehicle_id"`
	Distance     int    `json:"distance"`
	MileAge      int    `json:"mileage"`
	Start        string `json:"start"`
	Stop         string `json:"stop"`
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

	layout := "2006-01-02T15:04:05-07:00"
	if params.StartDate != "" {
		startDate, err = time.Parse(layout, params.StartDate+"+03:00")
		if err != nil {
			log.Println(err)
		}
	}
	if params.EndDate != "" {
		endDate, err = time.Parse(layout, params.EndDate+"+03:00")
		if err != nil {
			log.Println(err)
		}
	}
	carIDs := getccCars(params)

	days := int(endDate.Unix()-startDate.Unix()) / 60 / 60 / 24

	sendLayout := "2006-01-02T15:04-07:00"
	ccMileAge := make(map[string]CCarsMileage)
	for i := 0; i < days+1; i++ {
		ma := CCarsMileage{}
		st := startDate.AddDate(0, 0, i)

		cars := CarsCCar{
			TimeFrom: time.Date(
				st.Year(), st.Month(), st.Day(), 0, 0, 0, 0, now.Location(),
			).Format(sendLayout),
			TimeTo: time.Date(
				st.Year(), st.Month(), st.Day(), 23, 59, 59, 0, now.Location(),
			).Format(sendLayout),
		}
		cars.Cars = []string{}
		cars.Cars = append(cars.Cars, carIDs...)
		carsJson, _ := json.Marshal(cars)
		body = getCCarsApi(&params, "POST", "/api/v1/efficiency-management/efficiency", string(carsJson), url.Values{})
		_ = json.Unmarshal(body, &ma)

		ccMileAge[cars.TimeFrom] = ma
	}
	actions := CCarsMileage{}
	for date, value := range ccMileAge {
		for _, car := range value {
			car.Start = date
			car.Distance = car.MileAge
			actions = append(actions, car)
		}
	}
	body, _ = json.Marshal(actions)

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
		carData.Oid = car.VehicleID
		carData.IMEI = car.Vin
		carData.Name = car.LicencePlate
		carsData = append(carsData, carData)

	}
	body, _ = json.Marshal(carsData)
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
	body := getCCarsApi(
		params,
		"POST",
		"/api/v1/user-management/login",
		fmt.Sprintf(putData, params.Login, params.Password),
		url.Values{},
	)
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
