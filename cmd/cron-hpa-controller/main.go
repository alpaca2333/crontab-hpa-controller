package main

import (
	cron_hpa "cron-hpa-controller/internal/pkg/cron-hpa"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	logrus.SetLevel(logrus.TraceLevel)

	_, err := cron_hpa.NewController()
	if err != nil {
		logrus.Errorf("Cannot initialize new cron hpa controller: %v", err.Error())
		return
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
