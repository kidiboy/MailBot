package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

type BodyToTgm struct {
	From	string
	To		string
	Subject	string
	Text	string
}

func serverHttpStart(conf Conf) {
	httpPortStr := ":" + strconv.Itoa(conf.HttpServer.Port)
	httpLog.Infof("Starting HTTP server at %s port", httpPortStr)
	server := http.NewServeMux()
	// example:
	// url: http://192.168.1.1:80/toTgm
	// body: {"From": "test@test.ru", "To": "123@test.ru", "Subject": "My test message",
	// 		  "Text": "Hallo! This is my test message"}
	server.HandleFunc("/toTgm", func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			httpLog.Error(err)
		}
		var bodyDecode BodyToTgm
		err = json.Unmarshal(body, &bodyDecode)
		if err != nil {
			httpLog.Error(err)
		}

		sendToTgm(bodyDecode.Subject, bodyDecode.Text, conf, bodyDecode.To, httpLog)

	})
	err := http.ListenAndServe(httpPortStr, server)
	if err != nil {
		httpLog.Error(err)
	}
}
