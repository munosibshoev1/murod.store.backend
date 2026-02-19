package utils

import (
    "gopkg.in/gomail.v2"
)

func SendEmail(from, to, subject, body string) error {
    m := gomail.NewMessage()
    m.SetHeader("From", from)
    m.SetHeader("To", to)
    m.SetHeader("Subject", subject)
    m.SetBody("text/plain", body)

    d := gomail.NewDialer("smtp.beget.com", 465, "recoverycashback@nadim.shop", "Jgy09kU%4Sdd")

    return d.DialAndSend(m)
}
