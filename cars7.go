package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"encoding/json"

	"github.com/PuerkitoBio/goquery"
)

func login(params *Params) {
	client := getClient(true, "cars7")
	queryLoqin := url.QueryEscape(params.Login)
	loginURL := "http://lk.cars7.ru/Account/LoginApp?login=" + queryLoqin + "&password=" + params.Password
	_, err := client.Get(loginURL)

	if err != nil {
		fmt.Println(err)
	}
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
	client := getClient(true, "cars7")
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
	ID      string    `json:"id,omitempty"`
	OrderID string    `json:"order_id,omitempty"`
	Date    time.Time `json:"date,omitempty"`
	Sum     string    `json:"sum,omitempty"`
	PayDay  time.Time `json:"pay_day,omitempty"`
	Status  string    `json:"status,omitempty"`
	Comment string    `json:"comment,omitempty"`
	Vehicle string    `json:"vehicle,omitempty"`
	Orderer string    `json:"orderer,omitempty"`
}

type FileCompence struct {
	Name     string `json:"name,omitempty"`
	TypeDoc  string `json:"type_doc,omitempty"`
	Document []byte `json:"document,omitempty"`
}

func cars7Compence(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	login(&params)
	client := getClient(true, "cars7")

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

	compences := []Compence{}
	doc.Find(".change_page").Each(func(_ int, s *goquery.Selection) {
		resp, err := client.PostForm(fmt.Sprintf("https://lk.cars7.ru%v", s.AttrOr("href", "")), formData)
		if err != nil {
			fmt.Println("resp err: ", err)
		}
		compences = append(compences, getCompenceData(resp)...)
	})

	body, _ = json.Marshal(compences)
	w.Write(body)

}

func getCompenceData(resp *http.Response) []Compence {

	statuses := map[string]string{
		"Оплачен":    "1",
		"Не оплачен": "0",
		"Добровольная оплата картой или на Р/С": "6",
		"В суде":               "4",
		"В ожидании претензии": "7",
	}
	compences := []Compence{}
	compence := Compence{}
	client := getClient(true, "cars7")
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		fmt.Println("goquery err: ", err)
	}
	layout := "02.01.2006 15:04:05"
	doc.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {

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
		resp, _ := client.Get(fmt.Sprintf("https://lk.cars7.ru/Data/NewItem?type=9&id=%v&tz=-3&copy=false", compence.ID))

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
			compence.Status = statuses[s.Text()]
		})
		compences = append(compences, compence)
	})
	return compences
}

func cars7GetDocement(w http.ResponseWriter, r *http.Request) {
	var params Params
	body := StreamToByte(r.Body)
	err := json.Unmarshal(body, &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}

	login(&params)
	client := getClient(true, "cars7")

	resp, err := client.Get(fmt.Sprintf("https://lk.cars7.ru/Data/CreateClaim?type=1&id=%v", params.CompenceID))
	if err != nil {
		fmt.Println(err)
	}
	body, _ = ioutil.ReadAll(resp.Body)
	compence := FileCompence{}
	compence.Name = fmt.Sprintf("Притензия для %v", params.CompenceID)
	compence.TypeDoc = resp.Header.Get("content-type")
	compence.Document = body

	body, _ = json.Marshal(compence)
	w.Write(body)
}

func cars7CreateOrUpdateDocement(w http.ResponseWriter, r *http.Request) {
	// statuses := map[string]string{
	// 	"Оплачен":    "1",
	// 	"Не оплачен": "0",
	// 	"Добровольная оплата картой или на Р/С": "6",
	// 	"В суде":               "4",
	// 	"В ожидании претензии": "7",
	// }

	r.ParseMultipartForm(20 << 20)

	// Get handler for filename, size and headers
	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	dst, err := os.CreateTemp("", handler.Filename)
	defer dst.Close()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	incomeBody := r.FormValue("params")
	type CreateParams struct {
		Login      string `json:"login,omitempty"`
		Password   string `json:"password,omitempty"`
		OrderID    string `json:"order_id,omitempty"`
		Summ       string `json:"summ,omitempty"`
		Status     string `json:"status,omitempty"`
		Comment    string `json:"comment,omitempty"`
		CompenceID string `json:"compence_id,omitempty"`
	}
	var params CreateParams
	err = json.Unmarshal([]byte(incomeBody), &params)
	if err != nil {
		fmt.Println("JSON unmarshal error:", err)
	}
	parLoginPass := Params{
		Login:    params.Login,
		Password: params.Password,
	}
	login(&parLoginPass)
	client := getClient(true, "cars7")

	filedata, _ := os.Open(dst.Name())
	values := map[string]io.Reader{
		"file":                          filedata,
		"Compensation.Lease.Identifier": strings.NewReader(params.OrderID),
		"Compensation.Amount":           strings.NewReader(params.Summ),
		"Compensation.Comment":          strings.NewReader(params.Comment),
		"Compensation.Status":           strings.NewReader(params.Status),
		"Compensation.CompensationID":   strings.NewReader(params.CompenceID),
	}

	var b bytes.Buffer
	wmpd := multipart.NewWriter(&b)
	for key, r := range values {
		var part io.Writer

		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}

		if x, ok := r.(*os.File); ok {
			if handler.Size > 0 {
				part, err = wmpd.CreateFormFile(key, x.Name())
				if err != nil {
					fmt.Printf("form file err %v", err)
				}
			}
		} else {
			if part, err = wmpd.CreateFormField(key); err != nil {
				fmt.Printf("form field err %v", err)
			}
		}

		_, err := io.Copy(part, r)
		if err != nil {
			fmt.Printf("form copy file err %v", err)
		}

	}
	defer wmpd.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", "https://lk.cars7.ru/Data/SaveCompensation", &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", wmpd.FormDataContentType())

	// Submit the request
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	w.Write(body)
}
