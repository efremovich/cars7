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
	data := url.Values{}
	data.Set("oid", "0")
	body = getApi(&params, "GET", "/api/Api.svc/getfullupdateinfo", data)

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
		mileAgeActions = append(mileAgeActions, mileAgeData.Actions...)
	}
	body, err = json.Marshal(mileAgeActions)
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

	data := url.Values{}
	data.Set("all", "true")
	data.Set("node", "root")
	body = getApi(&params, "GET", "/api/Api.svc/gettree", data)
	w.Write(body)
}

func getApi(params *Params, method, urlPath string, data url.Values) []byte {

	client := getClient(false, "fortMonitor")
	u := params.URLstring + urlPath
	var rbody io.Reader
	if method == "GET" {
		u += "?" + data.Encode()
	} else if method == "PUT" {
		// rbody = strings.NewReader(putData)
	} else {
		rbody = strings.NewReader(data.Encode())
	}

	req, err := http.NewRequest(method, u, rbody)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}
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
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	doc.Find("input[type=hidden]").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		val, _ := s.Attr("value")
		data.Add(name, val)
	})
	result := getApi(params, "POST", "/login.aspx", data)
	fmt.Println(string(result))
}
