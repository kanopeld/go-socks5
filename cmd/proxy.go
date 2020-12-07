package main

import (
	"github.com/kanopeld/go-socks5"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	logger := logrus.New()
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "error"
	}
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	logger.SetLevel(level)

	logger.Info("Logger started")
	// Create a SOCKS5 server
	conf := &socks5.Config{
		Logger: logger,
	}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.ListenAndServe("tcp", ":"+os.Getenv("PROXY_PORT")); err != nil {
		panic(err)
	}
}
