package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/emersion/go-smtp"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	url2 "net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// The Backend implements SMTP server methods.
type Backend struct {conf Conf}

type Conf struct {
	ProxyTgm struct{
		Ip string
		Port int
	} `yaml:"proxy_tgm"`
	SmtpServer struct{
		Port int
		Debug bool
	} `yaml:"smtp_server"`
	TgmToken string `yaml:"tgm_token_bot"`
	TgmParseMode string `yaml:"tgm_parse_mode"`
	NotifyChats map[string]struct{
		Email string `yaml:"email"`
		ChatId string `yaml:"chat_id"`
		WebPagePreview bool `yaml:"web_page_preview"`
		Notification bool `yaml:"notification"`
	} `yaml:"notify_chats"`
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

// Login handles a login command with username and password.
func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	if username != "username" || password != "password" {
		return nil, errors.New("invalid username or password")
	}
	return &Session{bkd.conf}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	//return nil, smtp.ErrAuthRequired
	return &Session{bkd.conf}, nil
	//
}

// A Session is returned after successful login.
type Session struct{conf Conf}

func (s *Session) Mail(from string) error {
	log.Println("Mail from:", from)
	return nil
}

func (s *Session) Rcpt(to string) error {
	log.Println("Rcpt to:", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	msg, err1 := mail.ReadMessage(r)
	if err1 != nil {
		return err1
	}
	textArr, err2 := ioutil.ReadAll(msg.Body)
	if err2 != nil {
		return err2
	}
	prm := msg.Header
	text, err3 := DecodeUTF8(string(textArr))
	if err3 != nil {
		return err3
	}
	//prm, text := ParseMsg(r)
	log.Printf("%+v; %+v\n", prm, text)

	to := prm["To"][0]
	sbj, err3 := DecodeUTF8(prm["Subject"][0])
	if err3 != nil {
		return err3
	}


	text = strings.ReplaceAll(text, "<br />", "\n")

	var sendText string
	if sbj != "" {
		sendText = "<b>" + sbj + "</b>\n\n" + text
	} else {
		sendText = text
	}

	var url *url2.URL
	var err error

	cnt := len(s.conf.NotifyChats)
	for key, value := range  s.conf.NotifyChats {
		fmt.Printf("key: %+v; value: %+v ", key, value)
		cnt -= 1
		fmt.Println(cnt)
		prt := strconv.Itoa(s.conf.ProxyTgm.Port) //Convert Int to String
		wp := strconv.FormatBool(!value.WebPagePreview)
		ntf := strconv.FormatBool(!value.Notification)

		if to == value.Email {
			fmt.Println("Email found")
			p := "http://" + s.conf.ProxyTgm.Ip + ":" + prt + "/" + s.conf.TgmToken + "/sendMessage?chat_id=" +
				value.ChatId + "&parse_mode=" + s.conf.TgmParseMode + "&disable_web_page_preview=" + wp +
				"&disable_notification=" + ntf + "&text=" + url2.QueryEscape(sendText)
			fmt.Printf(p)
			url, err = url2.Parse(p)
			break
		} else if cnt == 0 {
			wp := strconv.FormatBool(!s.conf.NotifyChats["chat_rest"].WebPagePreview)
			ntf := strconv.FormatBool(!s.conf.NotifyChats["chat_rest"].Notification)
			p := "http://" + s.conf.ProxyTgm.Ip + ":" + prt + "/" + s.conf.TgmToken + "/sendMessage?chat_id=" +
				s.conf.NotifyChats["chat_rest"].ChatId + "&parse_mode=" + s.conf.TgmParseMode +
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
	log.Println(statuscode, string(data))

	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

func ParseMsg(r io.Reader) (map[string]string, string) {
	reader := bufio.NewReader(r)
	mapp := make(map[string]string)
	isMessageBody := false
	messageBody := ""
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		str := string(line)
		fmt.Printf("%s \n", line)
		if len(str) == 0 && !isMessageBody {
			isMessageBody = true
			continue
		}
		if !isMessageBody {
			i := strings.Index(str, ": ")
			key, val := str[:i], str[i+2:]
			mapp[key] = val
		} else {
			messageBody += str
			messageBody += "\n"
		}
	}
	return mapp, messageBody
}

func DecodeUTF8(p string) (string, error) {
	if strings.HasPrefix(p, "=?UTF-8?B?") {
		bytes, err := base64.StdEncoding.DecodeString(p[len("=?UTF-8?B?") : len(p)-2])
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	return p, nil
}

func main() {
	conf, err := ReadConfig("config.yml")
	if err != nil {
		log.Fatal(err)
		return
	}
	be := &Backend{*conf}
	serv := smtp.NewServer(be)
	serv.Addr = ":" + strconv.Itoa(conf.SmtpServer.Port)//":1025"
	serv.Domain = "localhost"
	serv.ReadTimeout = 10 * time.Second
	serv.WriteTimeout = 10 * time.Second
	serv.MaxMessageBytes = 1024 * 1024
	serv.MaxRecipients = 50
	serv.AllowInsecureAuth = true
	if conf.SmtpServer.Debug {
		serv.Debug = os.Stdout
	}

	log.Println("Starting server at", serv.Addr)
	if err := serv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}
