package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"encoding/json"

	"github.com/PuerkitoBio/goquery"
)

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

type Compence struct {
	ID       string    `json:"id"`
	OrderID  string    `json:"order_id"`
	Date     time.Time `json:"date"`
	Sum      string    `json:"sum"`
	PayDay   time.Time `json:"pay_day"`
	Status   string    `json:"status"`
	Comment  string    `json:"comment"`
	Vehicle  string
	Orderer  string
	Document []byte `json:"document"`
}

func cars7Compence(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	login(&params)
	client := getClient(false, "cars7")
	startDate := params.formatTime(params.StartDate, "2006-01-02T15:04:05.000Z")
	endDate := params.formatTime(params.EndDate, "2006-01-02T15:04:05.000Z")
	fmt.Printf("Запрос компенсаций с %s по %s", startDate, endDate)
	formData := url.Values{}
	formData.Set("DateStart", startDate)
	formData.Set("DateEnd", endDate)
	formData.Set("Timezone", "-3")

	resp, err := client.PostForm("https://lk.cars7.ru/Data/GetData?page=0&type=9&category=0&isClear=true", formData)
	if err != nil {
		fmt.Println("resp err: ", err)
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		fmt.Println("goquery err: ", err)
	}

	compences := []Compence{}
	layout := "02.01.2006 15:04:05"
	doc.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {
		compence := Compence{}

		s.Find(".button_edit").Each(func(_ int, id *goquery.Selection) {
			compence.ID = id.AttrOr("data-value", "0")
		})

		s.Find("td").Each(func(i int, data *goquery.Selection) {
			switch i {
			case 0:
				date, _ := time.Parse(layout, data.Text())
				compence.Date = date
			case 2:
				compence.Orderer = data.Text()
			case 4:
				date, _ := time.Parse(layout, data.Text())
				compence.PayDay = date
			}

		})
		resp, _ = client.Get(fmt.Sprintf("https://lk.cars7.ru/Data/NewItem?type=9&id=%v&tz=-3&copy=false", compence.ID))

		data, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			fmt.Println("goquery err: ", err)
		}

		data.Find("#Compensation_Lease_Identifier").Each(func(_ int, s *goquery.Selection) {
			compence.OrderID = s.AttrOr("value", "")
		})

		data.Find("#Compensation_Amount").Each(func(_ int, s *goquery.Selection) {
			compence.Sum = s.AttrOr("value", "")
		})

		data.Find("textarea").Each(func(_ int, s *goquery.Selection) {
			compence.Comment = s.Text()
		})

		data.Find("option[selected='selected']").Each(func(_ int, s *goquery.Selection) {
			compence.Status = s.Text()
		})

		compences = append(compences, compence)
	})

	body, _ = json.Marshal(compences)
	w.Write(body)

}
