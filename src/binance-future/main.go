package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
	"time"
)

var (
	controller     Controller
	configFileName string
	logger         *logrus.Logger
	config         Config
)

func init() {

}

// https://learnxinyminutes.com/docs/toml/

func main() {
	flag.StringVar(&configFileName, "config", "settings.toml", "configuration files")
	flag.Parse()

	data, _ := os.ReadFile(configFileName)
	if _, e := toml.Decode(string(data), &config); e != nil {
		fmt.Println(e)
		return
	}

	logger = logrus.New()
	logger.Formatter = new(logrus.TextFormatter) //default
	logger.Formatter = &nested.Formatter{
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
		TimestampFormat: time.RFC3339,
	}
	logger.Level = logrus.TraceLevel
	logger.Out = os.Stdout

	writer, err := os.OpenFile(config.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		log.Fatalf("create file log.txt failed: %v", err)
	}
	logger.SetOutput(io.MultiWriter(os.Stdout, writer))
	controller.init(&config)
	controller.open()
	logger.Info("Thunder Start..")
	waitCh := make(chan struct{})
	<-waitCh
}
