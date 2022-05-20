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
	"github.com/connerdouglass/go-geo"
)

type GeoData struct {
	Coordinate geo.Coordinate `json:"coordinate,omitempty"`
	Time       time.Time      `json:"time,omitempty"`
	Speed      float64        `json:"speed,omitempty"`
}

type Response struct {
	Ok        bool `json:"ok"`
	Responses []struct {
		DeviceID      int             `json:"device_id"`
		Ok            bool            `json:"ok"`
		TimeTruncated bool            `json:"time_truncated"`
		Gps           [][]interface{} `json:"gps"`
		Timings       struct {
			Gps struct {
				Wall float64 `json:"wall"`
				Proc float64 `json:"proc"`
			} `json:"gps"`
		} `json:"timings"`
	} `json:"responses"`
}
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

	loginZont(&params)

	carsData := []Objs{}

	err, body, carsData = carsDevices(params, err, body, carsData)
	actions := []Actions{}
	tmpData := make(map[string]*Actions, 0)

	for _, item := range carsData {
		request := `{"requests": [{
					  "device_id": %v,
					  "mintime": %v,
					  "maxtime": %v,
					  "data_types": [
						"gps"
					  ],
					  "request_options": {
						"gps": {
						  "include_last_before": true
						}
					  }
					}
				  ]
				}`
		request = fmt.Sprintf(request, item.Oid, startDate.Local().Unix(), endDate.Local().Unix())
		body = getZontApi(&params, "application/json", "POST", "/api/load_data", request, url.Values{})
		respReq := Response{}
		err := json.Unmarshal(body, &respReq)
		if err != nil {
			fmt.Println(err)
		}
		geoData := []GeoData{}
		for _, resp := range respReq.Responses {
			for _, gps := range resp.Gps {
				geoData = append(geoData, getGeoData(gps))
			}
		}
		prevGeo := geo.Coordinate{}
		// prevDate := startDate.Add(-1 * time.Minute)

		distance := 0.00
		for _, gd := range geoData {

			if prevGeo.Latitude != 0 && prevGeo.Longitude != 0 {
				distance = float64(geo.DistanceBetween(prevGeo, gd.Coordinate))
				timeStamp := gd.Time.Format("20060102")
				t1 := tmpData[timeStamp]

				if t1 == nil {
					action := Actions{}
					t1 = &action
				}

				t1.Distance += distance / 1000
				t1.Oid = item.Oid
				t1.Start = time.Date(gd.Time.Year(), gd.Time.Month(), gd.Time.Day(), 0, 0, 0, 0, currentLocation).Format(layout)
				t1.Stop = time.Date(gd.Time.Year(), gd.Time.Month(), gd.Time.Day(), 23, 59, 59, 0, currentLocation).Format(layout)

				// 	actions = append(actions, action)
				tmpData[timeStamp] = t1

			}
			prevGeo = gd.Coordinate
		}
		for _, value := range tmpData {
			actions = append(actions, *value)

		}
	}
	body, err = json.Marshal(actions)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(body)

}

func getGeoData(t interface{}) GeoData {
	listSlice, ok := t.([]interface{})
	geoData := GeoData{}

	if !ok {
		return geoData
	}

	timeUnix := listSlice[0].(float64)
	geoData.Time = time.Unix(int64(timeUnix), 0)

	geoData.Coordinate.Latitude = listSlice[2].(float64)
	geoData.Coordinate.Longitude = listSlice[1].(float64)
	geoData.Speed = listSlice[3].(float64)
	return geoData
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
	getZontApi(params, "application/x-www-form-urlencoded", "POST", "/login", "", data)

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

	req.Header.Set("ZONT-Brand", "zont")
	req.Header.Set("X-ZONT-Client", "web")
	req.Header.Set("X-ZONT-Client-Version", "2.66.3")
	req.Header.Set("X-ZONT-Guest", "false")
	req.Header.Set("X-ZONT-User", "229227")
	req.Header.Set("ZONT-WebGL-Support", "webgl2")

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
