package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gomail "gopkg.in/gomail.v2"
)

var development = false

var emailErrorsTo = os.Getenv("EMAIL_ERRORS_TO")
var smtpHost = os.Getenv("EMAIL_SMTP_HOST")
var smtpPort = os.Getenv("EMAIL_SMTP_PORT")
var smtpUsername = os.Getenv("EMAIL_SMTP_USER")
var smtpPassword = os.Getenv("EMAIL_SMTP_PASS")
var smtpDialer *gomail.Dialer
var logger *zap.Logger

func emailErrorsFunc(entry zapcore.Entry) error {
	// DO NOT USE THE LOGGER INSIDE THIS FUNCTION
	if smtpDialer == nil || entry.Level != zapcore.ErrorLevel {
		return nil
	}
	m := gomail.NewMessage(
		gomail.SetCharset("UTF-8"),
	)
	m.SetHeader(
		"To",
		emailErrorsTo,
	)
	m.SetHeader(
		"Subject",
		fmt.Sprintf(
			"FOF Logged %s At %s",
			entry.Level.String(),
			entry.Time.String(),
		),
	)
	m.SetHeader(
		"From",
		"root@fofgaming.com",
	)
	m.SetBody(
		"text/plain",
		fmt.Sprintf(
			"%s\n\n\u00A0Caller\n――――――――\n%s\n\n\u00A0Stack Trace\n―――――――――――――\n%s",
			entry.Message,
			entry.Stack,
			entry.Caller.String(),
		),
	)
	return smtpDialer.DialAndSend(m)
}

func initEmail() bool {
	if emailErrorsTo == "" || smtpHost == "" || smtpPort == "" {
		return false
	}
	port, _ := strconv.Atoi(smtpPort)
	if port == 0 {
		return false
	}
	smtpDialer = gomail.NewDialer(smtpHost, port, smtpUsername, smtpPassword)
	smtpDialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	return true
}

func initLogger() {
	var options = []zap.Option{}

	if ok := initEmail(); ok {
		options = append(options, zap.Hooks(emailErrorsFunc))
	}

	if development {
		logger, _ = zap.NewDevelopment(options...)
	} else {
		logger, _ = zap.NewProduction(options...)
	}
}

func init() {
	flag.BoolVar(&development, "dev", development, "enable development mode (mainly logging changes)")
}
