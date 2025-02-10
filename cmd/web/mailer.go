package main

import (
	"bytes"
	"fmt"
	"github.com/vanng822/go-premailer/premailer"
	mail "github.com/xhit/go-simple-mail/v2"
	"html/template"
	"sync"
	"time"
)

type Mail struct {
	Domain      string
	Host        string
	Port        int
	Username    string
	Password    string
	Encryption  string
	FromAddress string
	FromName    string
	Wait        *sync.WaitGroup
	MailChan    chan Message
	ErrorChan   chan error
	DoneChan    chan bool
}

type Message struct {
	From          string
	FromName      string
	To            []string
	Subject       string
	Attachment    []string
	AttachmentMap map[string]string
	Data          any
	DataMap       map[string]any
	Template      string
}

func (app *Config) listenForMail() {
	for {
		select {
		case msg := <-app.Mailer.MailChan:
			go app.Mailer.send(msg)
		case err := <-app.Mailer.ErrorChan:
			app.ErrorLog.Println(err)
		case <-app.Mailer.DoneChan:
			return
		}
	}
}

func (m *Mail) send(msg Message) {
	defer m.Wait.Done()

	if msg.Template == "" {
		msg.Template = "mail"
	}
	if msg.From == "" {
		msg.From = m.FromAddress
	}
	if msg.FromName == "" {
		msg.FromName = m.FromName
	}
	if msg.AttachmentMap == nil {
		msg.AttachmentMap = make(map[string]string)
	}
	if len(msg.DataMap) == 0 {
		msg.DataMap = make(map[string]any)
	}
	msg.DataMap["message"] = msg.Data

	formattedMsg, err := m.buildHtml(msg)
	if err != nil {
		m.ErrorChan <- err
	}
	plainMsg, err := m.buildPlain(msg)
	if err != nil {
		m.ErrorChan <- err
	}

	server := mail.NewSMTPClient()
	server.Host = m.Host
	server.Port = m.Port
	server.Username = m.Username
	server.Password = m.Password
	server.Encryption = m.getEnc(m.Encryption)
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		m.ErrorChan <- err
	}

	email := mail.NewMSG()
	email.SetFrom(msg.From).AddTo(msg.To[0]).SetSubject(msg.Subject)
	email.SetBody(mail.TextPlain, plainMsg)
	email.AddAlternative(mail.TextHTML, formattedMsg)

	if len(msg.Attachment) > 0 {
		for _, file := range msg.Attachment {
			email.AddAttachment(file)
		}
	}

	if len(msg.AttachmentMap) > 0 {
		for key, value := range msg.AttachmentMap {
			email.AddAttachment(value, key)
		}
	}

	err = email.Send(smtpClient)
	if err != nil {
		m.ErrorChan <- err
	}
}

func (m *Mail) buildHtml(msg Message) (string, error) {
	templateScheme := fmt.Sprintf("./cmd/web/templates/%s.html.gohtml", msg.Template)
	templ, err := template.New("email-html").ParseFiles(templateScheme)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	if err = templ.ExecuteTemplate(&tpl, "body", msg.DataMap); err != nil {
		return "", err
	}

	formattedMsg, err := m.inlineCss(tpl.String())
	if err != nil {
		return "", err
	}

	return formattedMsg, nil
}

func (m *Mail) buildPlain(msg Message) (string, error) {
	templateScheme := fmt.Sprintf("./cmd/web/templates/%s.plain.gohtml", msg.Template)
	templ, err := template.New("email-plain").ParseFiles(templateScheme)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	if err = templ.ExecuteTemplate(&tpl, "body", msg.DataMap); err != nil {
		return "", err
	}

	return tpl.String(), nil
}

func (m *Mail) inlineCss(s string) (string, error) {
	options := premailer.Options{
		RemoveClasses:     false,
		CssToAttributes:   false,
		KeepBangImportant: true,
	}

	prem, err := premailer.NewPremailerFromString(s, &options)
	if err != nil {
		return "", err
	}
	html, err := prem.Transform()
	if err != nil {
		return "", err
	}

	return html, nil
}

func (m *Mail) getEnc(e string) mail.Encryption {
	switch e {
	case "tls":
		return mail.EncryptionSTARTTLS
	case "ssl":
		return mail.EncryptionSSLTLS
	case "none":
		return mail.EncryptionNone
	default:
		return mail.EncryptionSTARTTLS
	}
}

func (m *Mail) terminate() {
	close(m.MailChan)
	close(m.ErrorChan)
	close(m.DoneChan)
}
