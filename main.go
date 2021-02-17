package main

import (
	"flag"
	"fmt"
	"github.com/op/go-logging"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	url2 "net/url"
	"os"
	"strconv"
	"strings"
)

type Conf struct {
	ProxyTgm struct {
		Ip   string
		Port int
	} `yaml:"proxy_tgm"`
	SmtpServer struct {
		Port   int
		Debug  bool
		LogLvl string `yaml:"logLvl"`
	} `yaml:"smtp_server"`
	HttpServer struct {
		Port   int
		Debug  bool
		LogLvl string `yaml:"logLvl"`
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

func checkConfig(conf *Conf) {
	//TODO
}

func sendToTgm(sbj string, text string, conf Conf, to string, logger *logging.Logger) {
	var sendText string
	if sbj != "" {
		sendText = "<b>" + sbj + "</b>\n\n" + text
	} else {
		sendText = text
	}

	notifyChats := conf.NotifyChats

	requiredChat := findNotifyChat(to, notifyChats, logger)

	tgmUrl, err := createTgmUrl(requiredChat, conf, sendText, logger)
	if err != nil {
		logger.Error(err)
	}

	sendHttpRequest(tgmUrl, logger)
}

func sendHttpRequest(tgmUrl *url2.URL, logger *logging.Logger) error {
	//generating the HTTP GET request
	request, err := http.NewRequest("GET", tgmUrl.String(), nil)
	if err != nil {
		logger.Error(err)
		return err
	}

	//calling the URL
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		logger.Error(err)
		return err
	}

	//getting the response
	statusCode := response.StatusCode
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error(err)
		return err
	}
	//printing the response
	logger.Debug(statusCode, string(data))
	return nil
}

func findNotifyChat(to string, notifyChats map[string]NotifyChat, logger *logging.Logger) NotifyChat {
	for key, currChat := range notifyChats {
		logger.Debugf("key: %+v; currChat: %+v ", key, currChat)
		if to == currChat.Email {
			logger.Debugf("Email found: %s", currChat.Email)
			return currChat
		}
	}
	logger.Debugf("Email NOT found, use default")
	return notifyChats["chat_rest"]
}

func createTgmUrl(requiredChat NotifyChat, conf Conf, sendText string, logger *logging.Logger) (*url2.URL, error) {
	prt := ":" + strconv.Itoa(conf.ProxyTgm.Port) //Convert Int to String
	wp := strconv.FormatBool(!requiredChat.WebPagePreview)
	ntf := strconv.FormatBool(!requiredChat.Notification)
	strUrl := "http://" + conf.ProxyTgm.Ip + prt + "/" + conf.TgmToken + "/sendMessage?chat_id=" + requiredChat.ChatId +
		"&parse_mode=" + conf.TgmParseMode + "&disable_web_page_preview=" + wp + "&disable_notification=" + ntf +
		"&text=" + url2.QueryEscape(sendText)
	logger.Debugf("strUrl: %s", strUrl)
	url, err := url2.Parse(strUrl)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	return url, nil
}

var configLog = logging.MustGetLogger("forConfig")
var smtpLog = logging.MustGetLogger("smtpLog")
var httpLog = logging.MustGetLogger("httpLog")

// Example format string. Everything except the message has a custom color
// which is dependent on the log level. Many fields have a custom output
// formatting too, eg. the time returns the hour down to the milli second.
// Def format: `%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`
var formatToConfig = logging.MustStringFormatter(
	`%{color}%{time:2006/01/02 15:04:05.000}  %{shortfunc} CONF%{color:reset} %{message}`,
)
var format = logging.MustStringFormatter(
	`%{color}%{time:2006/01/02 15:04:05.000}  %{shortfunc} %{level:.4s}%{color:reset} %{message}`,
)

func main() {
	confBackend := logging.AddModuleLevel(
		logging.NewBackendFormatter(
			logging.NewLogBackend(os.Stdout, "", 0), formatToConfig))
	//	"CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG"
	confBackend.SetLevel(logging.INFO, "")
	configLog.SetBackend(confBackend)

	smtpBackend := logging.AddModuleLevel(
		logging.NewBackendFormatter(
			logging.NewLogBackend(os.Stdout, "", 0), format))
	smtpLog.SetBackend(smtpBackend)

	httpBackend := logging.AddModuleLevel(
		logging.NewBackendFormatter(
			logging.NewLogBackend(os.Stdout, "", 0), format))
	httpLog.SetBackend(httpBackend)

	var confPath string

	//parsing flag "--conf"
	flag.StringVar(&confPath, "conf", "config.yml", "Path to config")
	flag.Parse()
	configLog.Infof("path to config: %s", confPath)

	conf, err := ReadConfig(confPath)
	if err != nil {
		configLog.Critical(err)
		return
	}

	checkConfig(conf)

	setLogLevel(conf.SmtpServer.LogLvl, smtpLog, smtpBackend)
	setLogLevel(conf.HttpServer.LogLvl, httpLog, httpBackend)

	go serverSmtpStart(*conf)
	serverHttpStart(*conf)

}

func setLogLevel(confLogLevel string, logger *logging.Logger, backend logging.LeveledBackend) {
	var logLvl logging.Level
	//maps config log level to library level (github)
	switch strings.ToUpper(confLogLevel) {
	case "DEBUG":
		logLvl = logging.DEBUG
	case "INFO":
		logLvl = logging.INFO
	case "WORN":
		logLvl = logging.WARNING
	case "ERR":
		logLvl = logging.ERROR
	default:
		logLvl = logging.INFO
		logger.Warningf("the value of parameter \"logLvl\" in the configuration file is set incorrectly "+
			"(\"%s\")", confLogLevel)
		logger.Warningf("the default value was applied. logLvl: %s", logLvl)
	}

	backend.SetLevel(logLvl, "")
}
