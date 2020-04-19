package email

import (
	"fmt"
	"net/smtp"
)

var SMTPServerHost = "smtp.gmail.com"
var SMTPServerPort = 587

var MailAddress = "benevolentbites.forward@gmail.com"
var MailPassword = "odyyyiujzvrcemav"

func SendEmail(to []string, subject string, body string) error {
	auth := smtp.PlainAuth("", MailAddress, MailPassword, SMTPServerHost)
	err := smtp.SendMail(fmt.Sprintf("%s:%v", SMTPServerHost, SMTPServerPort), auth, MailAddress, to, []byte(fmt.Sprintf(
		"Subject: %s\n\n%s", subject, body,
	)))
	if err != nil {
		return err
	}
	return nil
}
