package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	url2 "net/url"
	"os"
	"strconv"
)

type Conf struct {
	ProxyTgm struct {
		Ip   string
		Port int
	} `yaml:"proxy_tgm"`
	SmtpServer struct {
		Port  int
		Debug bool
	} `yaml:"smtp_server"`
	HttpServer struct {
		Port  int
		Debug bool
	} `yaml:"http_server"`
	TgmToken     string                `yaml:"tgm_token_bot"`
	TgmParseMode string                `yaml:"tgm_parse_mode"`
	NotifyChats  map[string]NotifyChat `yaml:"notify_chats"`
}

type NotifyChat struct {
	Email          string `yaml:"email"`
	ChatId         string `yaml:"chat_id"`
	WebPagePreview bool   `yaml:"web_page_preview"`
	Notification   bool   `yaml:"notification"`
}

func ReadConfig(path string) (*Conf, error) {
	confFile, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if confFile != nil {
		fileAll, _ := ioutil.ReadAll(confFile)
		log.Println(string(fileAll))
		log.Println("#########################################################")
		currConf := Conf{}

		err := yaml.Unmarshal(fileAll, &currConf)
		if err != nil {
			log.Fatalf("error: %v", err)
			return nil, err
		}
		log.Printf("Applyed config:\n%+v\n", currConf)
		log.Print("#########################################################")
		return &currConf, nil
	}
	return nil, fmt.Errorf("file %s is empty", path)
}

func sendToTgm(sbj string, text string, conf Conf, to string) {
	var sendText string
	if sbj != "" {
		sendText = "<b>" + sbj + "</b>\n\n" + text
	} else {
		sendText = text
	}

	//var url *url2.URL
	//var err error

	notifyChats := conf.NotifyChats

	requiredChat := findNotifyChat(to, notifyChats)

	tgmUrl, err := createTgmUrl(requiredChat, conf, sendText)
	if err != nil {
		log.Println(err)
	}

	sendHttpRequest(tgmUrl)

	/*cnt := len(conf.NotifyChats)
	for key, value := range conf.NotifyChats {
		fmt.Printf("key: %+v; value: %+v ", key, value)
		cnt -= 1
		fmt.Println(cnt)

		if to == value.Email {
			fmt.Println("Email found")
			prt := strconv.Itoa(conf.ProxyTgm.Port) //Convert Int to String
			wp := strconv.FormatBool(!value.WebPagePreview)
			ntf := strconv.FormatBool(!value.Notification)
			p := "http://" + conf.ProxyTgm.Ip + ":" + prt + "/" + conf.TgmToken + "/sendMessage?chat_id=" +
				value.ChatId + "&parse_mode=" + conf.TgmParseMode + "&disable_web_page_preview=" + wp +
				"&disable_notification=" + ntf + "&text=" + url2.QueryEscape(sendText)
			fmt.Printf(p)
			url, err = url2.Parse(p)
			break
		} else if cnt == 0 {
			prt := strconv.Itoa(conf.ProxyTgm.Port) //Convert Int to String
			wp := strconv.FormatBool(!conf.NotifyChats["chat_rest"].WebPagePreview)
			ntf := strconv.FormatBool(!conf.NotifyChats["chat_rest"].Notification)
			p := "http://" + conf.ProxyTgm.Ip + ":" + prt + "/" + conf.TgmToken + "/sendMessage?chat_id=" +
				conf.NotifyChats["chat_rest"].ChatId + "&parse_mode=" + conf.TgmParseMode +
				"&disable_web_page_preview=" + wp + "&disable_notification=" + ntf + "&text=" +
				url2.QueryEscape(sendText)
			fmt.Println(p)
			url, err = url2.Parse(p)
			fmt.Println("Email not found, chat_rest")
		}
	}

	fmt.Println(&url)
	//fmt.Println(r.Form)

	//generating the HTTP GET request
	request, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		log.Println(err)
	}

	//calling the URL
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Println(err)
	}

	//getting the response
	statuscode := response.StatusCode
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
	}
	//printing the response
	log.Println(statuscode, string(data))*/
}

func sendHttpRequest(tgmUrl *url2.URL) error {
	//generating the HTTP GET request
	request, err := http.NewRequest("GET", tgmUrl.String(), nil)
	if err != nil {
		log.Println(err)
		return err
	}

	//calling the URL
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Println(err)
		return err
	}

	//getting the response
	statusCode := response.StatusCode
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return err
	}
	//printing the response
	log.Println(statusCode, string(data))
	return nil
}

func findNotifyChat(to string, notifyChats map[string]NotifyChat) NotifyChat {
	//cnt := len(notifyChats)
	for key, currChat := range notifyChats {
		fmt.Printf("key: %+v; currChat: %+v ", key, currChat)
		if to == currChat.Email {
			fmt.Println("Email found")
			return currChat
		}
	}
	return notifyChats["chat_rest"]
}

func createTgmUrl(requiredChat NotifyChat, conf Conf, sendText string) (*url2.URL, error) {
	prt := ":" + strconv.Itoa(conf.ProxyTgm.Port) //Convert Int to String
	wp := strconv.FormatBool(!requiredChat.WebPagePreview)
	ntf := strconv.FormatBool(!requiredChat.Notification)
	strUrl := "http://" + conf.ProxyTgm.Ip + prt + "/" + conf.TgmToken + "/sendMessage?chat_id=" + requiredChat.ChatId +
		"&parse_mode=" + conf.TgmParseMode + "&disable_web_page_preview=" + wp + "&disable_notification=" + ntf +
		"&text=" + url2.QueryEscape(sendText)
	fmt.Println(strUrl)
	url, err := url2.Parse(strUrl)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return url, nil
}

func main() {
	conf, err := ReadConfig("config.yml")
	if err != nil {
		log.Fatal(err)
		return
	}

	go serverSmtpStart(*conf)
	serverHttpStart(*conf)

}
