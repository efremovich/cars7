package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Devices struct {
	Devices []struct {
		ID                      int         `json:"id"`
		IP                      string      `json:"ip"`
		IsActive                bool        `json:"is_active"`
		Online                  bool        `json:"online"`
		OwnerUsername           string      `json:"owner_username"`
		UserID                  int         `json:"user_id"`
		LastReceiveTime         int         `json:"last_receive_time"`
		LastReceiveTimeRelative int         `json:"last_receive_time_relative"`
		Name                    string      `json:"name"`
		Color                   string      `json:"color"`
		Notes                   string      `json:"notes"`
		Serial                  string      `json:"serial"`
		VisibleDeviceType       interface{} `json:"visible_device_type"`
		Imei                    string      `json:"imei"`
	} `json:"devices"`
}

type Au struct {
	Requests []struct {
		DeviceID       int      `json:"device_id"`
		Mintime        int      `json:"mintime"`
		Maxtime        int      `json:"maxtime"`
		DataTypes      []string `json:"data_types"`
		RequestOptions struct {
			Gps struct {
				IncludeLastBefore bool `json:"include_last_before"`
			} `json:"gps"`
		} `json:"request_options"`
	} `json:"requests"`
}

func getZontMileAge(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startDate := firstOfMonth
	endDate := lastOfMonth

	if params.StartDate != "" {

		layout := "2006-01-02T15:04:05"
		startDate, err = time.Parse(layout, params.StartDate)
		if err != nil {
			log.Println(err)
		}
	}
	if params.EndDate != "" {
		layout := "2006-01-02T15:04:05"
		endDate, err = time.Parse(layout, params.EndDate)
		if err != nil {
			log.Println(err)
		}
	}

	loginZont(&params)

	carsData := []Objs{}

	err, body, carsData = carsDevices(params, err, body, carsData)

	for _, item := range carsData {
		request := `{"requests": [{
      "device_id": %v,
      "mintime": %v,
      "maxtime": %v,
      "data_types": [
        "gps",
      ],
      "request_options": {
        "gps": {
          "include_last_before": true
        }
      }
    }
  ]
}
`
		request = fmt.Sprintf(request, item.Oid, startDate.Unix(), endDate.Unix())
		fmt.Println(request)
	}
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
	carsData := []Objs{}

	err, body, carsData = carsDevices(params, err, body, carsData)

	body, err = json.Marshal(carsData)
	w.Write(body)
}

func carsDevices(params Params, err error, body []byte, carsData []Objs) (error, []byte, []Objs) {
	client := getClient(false, params.Password)
	resp, err := client.Get(params.URLstring + "/console/")

	body, _ = ioutil.ReadAll(resp.Body)

	regularExpression := regexp.MustCompile(`JSON\.parse\(\"(.*)\"\);`)

	result := strings.ReplaceAll(regularExpression.FindString(string(body)), `JSON.parse("`, "")
	result, err = strconv.Unquote(fmt.Sprintf(`"%s"`, result[:len(result)-3]))
	devicesData := Devices{}
	err = json.Unmarshal([]byte(result), &devicesData)
	if err != nil {
		fmt.Println(err)
	}
	deviceData := Objs{}
	for _, device := range devicesData.Devices {
		deviceData.IMEI = device.Imei
		deviceData.Name = device.Name
		deviceData.Oid = device.ID
		carsData = append(carsData, deviceData)
	}
	return err, body, carsData
}

func loginZont(params *Params) {
	client := getClient(false, params.Password)
	resp, err := client.Get(params.URLstring + "/login")
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
	getZontApi(params, "POST", "/login", data)

}

func getZontApi(params *Params, contentType, method, urlPath, putData string, data url.Values) []byte {

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
	if method == "POST" {
		req.Header.Set("Content-Type", contentType)
	} else if method == "PUT" {
		req.Header.Set("Content-Type", contentType)
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
