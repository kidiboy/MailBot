package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
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
	log.Println(httpPortStr)
	server := http.NewServeMux()
	// example:
	// url: http://192.168.1.1:80/toTgm
	// body: {"From": "test@test.ru", "To": "123@test.ru", "Subject": "My test message",
	// 		  "Text": "Hallo! This is my test message"}
	server.HandleFunc("/toTgm", func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}
		var bodyDecode BodyToTgm
		err = json.Unmarshal(body, &bodyDecode)
		if err != nil {
			log.Println(err)
		}

		sendToTgm(bodyDecode.Subject, bodyDecode.Text, conf, bodyDecode.To)

	})
	err := http.ListenAndServe(httpPortStr, server)
	if err != nil {
		log.Println(err)
	}
}
