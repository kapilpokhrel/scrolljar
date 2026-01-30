package mailer

import (
	"bytes"
	"embed"
	"flag"
	"html/template"
	"os"
	"strconv"

	gomail "gopkg.in/mail.v2"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	dialer *gomail.Dialer
	sender string
}

type MailerCFG struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

func (cfg *MailerCFG) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&cfg.Host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	smtpPort, _ := strconv.ParseInt(os.Getenv("SMTP_PORT"), 10, 64)
	fs.IntVar(&cfg.Port, "smtp-port", int(smtpPort), "SMTP port")
	fs.StringVar(&cfg.Username, "smt-username", os.Getenv("SMTP_USERNAME"), "SMPT Username")
	fs.StringVar(&cfg.Password, "smt-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	fs.StringVar(&cfg.Sender, "smtp-sender", os.Getenv("SMTP_SENDER"), "SMTP sender")
}

func New(cfg MailerCFG) Mailer {
	dailer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	return Mailer{
		dialer: dailer,
		sender: cfg.Sender,
	}
}

func (m Mailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}
	subject := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(subject, "subject", data); err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(plainBody, "plainBody", data); err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(htmlBody, "htmlBody", data); err != nil {
		return err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	err = m.dialer.DialAndSend(msg)
	return err
}
