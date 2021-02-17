package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/emersion/go-smtp"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
)

// The Backend implements SMTP server methods.
type Backend struct{ conf Conf }

// A Session is returned after successful login.
type Session struct{ conf Conf }

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

func (s *Session) Mail(from string) error {
	smtpLog.Debugf("Mail from: %s", from)
	return nil
}

func (s *Session) Rcpt(to string) error {
	smtpLog.Debugf("Rcpt to: %s", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	msg, err1 := mail.ReadMessage(r)
	if err1 != nil {
		return err1
	}
	mediaType, params, err11 := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err11 != nil {
		return err11
	}
	var body string
	var err2 error
	if strings.HasPrefix(mediaType, "multipart/") {
		body, err2 = findHtmlPart(multipart.NewReader(msg.Body, params["boundary"]))
	} else {
		var textArr []byte
		textArr, err2 = ioutil.ReadAll(msg.Body)
		body = string(textArr)
	}
	if err2 != nil {
		return err2
	}
	prm := msg.Header
	text, err3 := DecodeUTF8(body)
	if err3 != nil {
		return err3
	}
	//prm, text := ParseMsg(r)
	smtpLog.Debugf("Header: %+v; Body: %+v\n", prm, text)

	to := prm["To"][0]
	decoder := new(mime.WordDecoder)
	sbj, err3 := decoder.DecodeHeader(prm["Subject"][0])
	if err3 != nil {
		return err3
	}

	text = strings.ReplaceAll(text, "<br />", "\n")

	conf := s.conf

	sendToTgm(sbj, text, conf, to, smtpLog)

	return nil
}

func findHtmlPart(reader *multipart.Reader) (string, error) {
	for {
		p, err := reader.NextPart()
		//if err == io.EOF {
		//	return "", io.EOF
		//}
		if err != nil {
			return "", err
		}

		content, err := ioutil.ReadAll(p)
		if err != nil {
			return "", err
		}

		isHtml := strings.HasPrefix(p.Header.Get("Content-Type"), "text/html;")
		if isHtml {
			return string(content), nil
		}
	}
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

func serverSmtpStart(conf Conf) {
	be := &Backend{conf}
	serv := smtp.NewServer(be)
	serv.Addr = ":" + strconv.Itoa(conf.SmtpServer.Port) //":1025"
	serv.Domain = "localhost"
	serv.ReadTimeout = 10 * time.Second
	serv.WriteTimeout = 10 * time.Second
	serv.MaxMessageBytes = 1024 * 1024
	serv.MaxRecipients = 50
	serv.AllowInsecureAuth = true
	if strings.ToUpper(conf.SmtpServer.LogLvl) == "DEBUG" {
		serv.Debug = os.Stdout
	}

	smtpLog.Infof("Starting SMTP server at %s port", serv.Addr)
	if err := serv.ListenAndServe(); err != nil {
		smtpLog.Fatal(err)
	}
}
