package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ObjectsData struct {
	Alarms   Alarms      `json:"alarms"`
	CmdInfo  CmdInfo     `json:"cmdInfo"`
	ObjInfo  interface{} `json:"objInfo"`
	ObjsInfo ObjsInfo    `json:"objsInfo"`
	Result   string      `json:"result"`
}
type Alarms struct {
	Alarms []interface{} `json:"alarms"`
	Result string        `json:"result"`
}
type CmdInfo struct {
	Msgs   []interface{} `json:"msgs"`
	Result string        `json:"result"`
}
type Objs struct {
	Dir  int     `json:"dir"`
	Dt   string  `json:"dt"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Move int     `json:"move"`
	Oid  int     `json:"oid"`
	St   int     `json:"st"`
	IMEI string  `json:"imei"`
	Name string  `json:"Name"`
}
type ObjsInfo struct {
	Objs   []Objs `json:"objs"`
	Result string `json:"result"`
}

type MileAgeData struct {
	Oid       int         `json:"oid"`
	Result    string      `json:"result"`
	Extension interface{} `json:"extension"`
	Actions   []Actions   `json:"actions"`
	Total     Total       `json:"total"`
}
type Actions struct {
	ActionType        int     `json:"actionType"`
	Start             string  `json:"start"`
	Stop              string  `json:"stop"`
	Duration          string  `json:"duration"`
	ActionName        string  `json:"actionName"`
	Address           string  `json:"address"`
	Icon              string  `json:"icon"`
	Distance          float64 `json:"distance"`
	Fuel              float64 `json:"fuel"`
	MotoHours         string  `json:"motoHours"`
	ActionDoubleValue float64 `json:"actionDoubleValue"`
	Oid               int     `json:"oid"`
}
type Total struct {
	Distance    float64 `json:"distance"`
	Fuel        float64 `json:"fuel"`
	MotoHours   string  `json:"motoHours"`
	MoveTime    string  `json:"moveTime"`
	ParkingTime string  `json:"parkingTime"`
	Refielings  float64 `json:"refielings"`
	Drains      float64 `json:"drains"`
}

func getMileAge(w http.ResponseWriter, r *http.Request) {
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
	_ = json.Unmarshal(body, &objectsData)
	for _, obj := range objectsData.ObjsInfo.Objs {
		data := url.Values{}
		data.Set("from", startDate)
		oid := strconv.Itoa(obj.Oid)
		data.Set("oid", oid)
		data.Set("showFuelingsDrains", "false")
		data.Set("to", endtDate)
		body = getApi(&params, "POST", "/api/v2/quickreport/getreport", data)
		mileAgeData := MileAgeData{}
		_ = json.Unmarshal(body, &mileAgeData)
		for _, act := range mileAgeData.Actions {
			act.Oid = obj.Oid
			mileAgeActions = append(mileAgeActions, act)
		}
	}
	body, _ = json.Marshal(mileAgeActions)
	w.Write(body)

}

func getFortCars(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	loginFort(&params)

	now := time.Now()

	data := url.Values{}
	data.Set("date", now.Format("2006-01-02 15:04:05"))
	data.Set("oid", "0")
	data.Set("limit", "200")
	body = getApi(&params, "GET", "/api/Api.svc/getfullupdateinfo", data)
	carsData := []Objs{}
	objectsData := ObjectsData{}
	_ = json.Unmarshal(body, &objectsData)
	for _, obj := range objectsData.ObjsInfo.Objs {
		data := url.Values{}
		oid := strconv.Itoa(obj.Oid)
		data.Set("oid", oid)
		body = getApi(&params, "GET", "/api/Api.svc/objectinfo", data)
		objinfo := Objs{}
		_ = json.Unmarshal(body, &objinfo)
		obj.Name = objinfo.Name
		obj.IMEI = objinfo.IMEI
		carsData = append(carsData, obj)
	}

	body, _ = json.Marshal(carsData)
	w.Write(body)
}

func getApi(params *Params, method, urlPath string, data url.Values) []byte {

	client := getClient(false, params.Password)
	u := params.URLstring + urlPath
	var rbody io.Reader

	req, err := http.NewRequest(method, u, rbody)
	if err != nil {
		panic(err)
	}
	switch method {
	case "GET":
		u += "?" + data.Encode()
	case "PUT":
		req.Header.Set("Content-Type", "application/json")
	// rbody = strings.NewReader(putData)
	default:
		rbody = strings.NewReader(data.Encode())
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	req.Header.Set("Accept", "application/json")

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

func loginFort(params *Params) {
	client := getClient(false, "fortMonitor")
	resp, err := client.Get(params.URLstring + "/login.aspx")
	if err != nil {
		fmt.Println(err, "Ошибка авторизации fort-monitor")
	}

	data := url.Values{}

	data.Set("Timezone", "3")
	data.Set("tbLogin", params.Login)
	data.Set("tbPassword", params.Password)
	data.Set("ddlLanguage", "ru-ru")
	data.Set("CheckNewInterface", "on")
	data.Set("__EVENTTARGET", "lbEnter")
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	doc.Find("input[type=hidden]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		val, _ := s.Attr("value")
		data.Add(name, val)
	})
	result := getApi(params, "POST", "/login.aspx", data)
	fmt.Println(string(result))
}
