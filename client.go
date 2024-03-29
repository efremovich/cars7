package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"golang.org/x/net/publicsuffix"
)

func redirectPolicyFunc(_ *http.Request, via []*http.Request) error {
	if len(via) > 10 {
		return errors.New("to many redirects")
	}
	return nil
}

func createJar(em string) {
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Printf("Ошибка при инициализации клиента: %v", err)
	}

	coockie[em] = jar
}

//getTLSClient получение защищенного клиента HTTPS
func getClient(isTLSEnabled bool, name string) http.Client {
	redir := redirectPolicyFunc

	if _, ok := coockie[name]; !ok {
		createJar(name)
	}
	client := http.Client{
		Jar:           coockie[name],
		CheckRedirect: redir,
	}
	if isTLSEnabled {
		tr := &http.Transport{TLSClientConfig: &tls.Config{Renegotiation: tls.RenegotiateOnceAsClient, InsecureSkipVerify: true}}
		client.Transport = tr
	}
	return client
}

// StreamToByte - Удаление Byte order mark — маркер последовательности байтов
func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	body := bytes.TrimPrefix(buf.Bytes(), []byte("\xef\xbb\xbf")) // Or []byte{239, 187, 191}
	return body
}

func (p *Params) formatTime(datestr string, format string) string {

	layout := "2006-01-02T15:04:05"
	t, err := time.Parse(layout, datestr)
	if err != nil {
		log.Println(err)
	}
	return t.Format(format)
}
