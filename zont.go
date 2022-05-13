package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Devices struct {
	Devices interface{} `json:"devices"`
}

func getZontMileAge(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	loginFort(&params)
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startDate := firstOfMonth.Format("2006-01-02")
	endtDate := lastOfMonth.Format("2006-01-02")

	if params.StartDate != "" {
		startDate = params.formatTime(params.StartDate, "2006-01-02 15:04:05")
	}
	if params.EndDate != "" {
		endtDate = params.formatTime(params.EndDate, "2006-01-02 15:04:05")
	}

	data := url.Values{}
	data.Set("oid", "0")
	data.Set("date", endtDate)
	data.Set("limit", "200")
	body = getApi(&params, "GET", "/api/Api.svc/getfullupdateinfo", data)

	mileAgeActions := []Actions{}
	objectsData := ObjectsData{}
	err = json.Unmarshal(body, &objectsData)
	for _, obj := range objectsData.ObjsInfo.Objs {
		data := url.Values{}
		data.Set("from", startDate)
		oid := strconv.Itoa(obj.Oid)
		data.Set("oid", oid)
		data.Set("showFuelingsDrains", "false")
		data.Set("to", endtDate)
		body = getApi(&params, "POST", "/api/v2/quickreport/getreport", data)
		mileAgeData := MileAgeData{}
		err = json.Unmarshal(body, &mileAgeData)
		for _, act := range mileAgeData.Actions {
			act.Oid = obj.Oid
			mileAgeActions = append(mileAgeActions, act)
		}
	}
	body, err = json.Marshal(mileAgeActions)
	w.Write(body)

}

func getZontCars(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	loginZont(&params)

	body = getApi(&params, "GET", "/console/", url.Values{})

	fmt.Println(body)

	now := time.Now()

	data := url.Values{}
	data.Set("date", now.Format("2006-01-02 15:04:05"))
	data.Set("oid", "0")
	data.Set("limit", "200")
	body = getApi(&params, "GET", "/api/Api.svc/getfullupdateinfo", data)
	carsData := []Objs{}
	objectsData := ObjectsData{}
	err = json.Unmarshal(body, &objectsData)
	for _, obj := range objectsData.ObjsInfo.Objs {
		data := url.Values{}
		oid := strconv.Itoa(obj.Oid)
		data.Set("oid", oid)
		body = getApi(&params, "GET", "/api/Api.svc/objectinfo", data)
		objinfo := Objs{}
		err = json.Unmarshal(body, &objinfo)
		obj.Name = objinfo.Name
		obj.IMEI = objinfo.IMEI
		carsData = append(carsData, obj)
	}

	body, err = json.Marshal(carsData)
	w.Write(body)
}

func loginZont(params *Params) {
	client := getClient(false, "fortMonitor")
	resp, err := client.Get(params.URLstring + "/login.aspx")
	if err != nil {
		fmt.Println(err, "Ошибка авторизации fort-monitor")
	}

	data := url.Values{}

	data.Set("username", params.Login)
	data.Set("password", params.Password)
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	doc.Find("input[type=hidden]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		val, _ := s.Attr("value")
		data.Add(name, val)
	})
	result := getApi(params, "POST", "/login", data)
	fmt.Println(string(result))
}
