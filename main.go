package lambda

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/jlaffaye/ftp"
)

var FFMPEG_PATH string = "ffmpeg"

func init() {
	functions.HTTP("TriggerCapture", triggerCapture)
}

func triggerCapture(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var triggerCapture TriggerCapture
	err := decoder.Decode(&triggerCapture)
	if err != nil {
		panic(err)
	}

	storeHandler := storeLocation(triggerCapture.StoreLocation)
	job := CaptureJob{
		streamUrl:    triggerCapture.StreamUrl,
		recTimeout:   triggerCapture.RecTimeout,
		storeHandler: storeHandler,
	}
	job.process()
}

func storeLocation(storeLocationString string) StrorageHandler {
	ftpReg := regexp.MustCompile(`ftp://(.*)\:(.*)@([A-Z\.\-a-z]+)(/.*)`)
	matches := ftpReg.FindStringSubmatch(storeLocationString)
	if len(matches) == 5 {
		return FtpStorage{
			User: matches[1],
			Pass: matches[2],
			Host: matches[3],
			Path: matches[4],
		}
	}
	panic("Cannot process storeLocation")

}

type FtpStorage struct {
	Host string
	User string
	Pass string
	Path string
}

type TriggerCapture struct {
	StreamUrl     string `json:"streamUrl"`
	RecTimeout    string `json:"recTimeout"`
	StoreLocation string `json:"storeLocation"`
}

type CaptureJob struct {
	streamUrl    string
	recTimeout   string
	storeHandler StrorageHandler
}

type StrorageHandler interface {
	store(string)
}

func (captureJob CaptureJob) process() {
	log.Println("Capture started")
	timeout, err := strconv.Atoi(captureJob.recTimeout)
	filename := time.Now().Format("2006-01-02T15-04-05") + "-" + captureJob.recTimeout
	if err != nil {
		log.Panicf("Failed to read timeout %s", captureJob.recTimeout)
	}
	context_timeout := time.Duration(timeout+30) * time.Second
	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.mp3", filename))
	log.Printf("Capture to file:: %s", outputFile)

	context, _ := context.WithTimeout(context.Background(), context_timeout)
	cmd := exec.CommandContext(context, FFMPEG_PATH, "-i", captureJob.streamUrl, "-t", captureJob.recTimeout, outputFile, "-y")

	log.Printf("Executing: %s\n", cmd.String())
	err = cmd.Run()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Capture done")
	captureJob.storeHandler.store(outputFile)
}

func (ftpStorage FtpStorage) store(sourceFile string) {
	c, err := ftp.Dial(ftpStorage.Host+":21", ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Println(err)
		return
	}

	err = c.Login(ftpStorage.User, ftpStorage.Pass)
	if err != nil {
		log.Println(err)
		return
	}

	dirs := strings.Split(ftpStorage.Path, "/")
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		log.Printf("cd to %s\n", dir)
		err = c.ChangeDir(dir)
		if err != nil {
			log.Printf("failed, mkdir+cd: %s\n", dir)
			c.MakeDir(dir)
			c.ChangeDir(dir)
		}
	}

	log.Printf("Sending file: %s", sourceFile)
	_, filename := filepath.Split(sourceFile)
	file, err := os.Open(sourceFile)
	if err != nil {
		log.Println(err)
		return
	}

	reader := bufio.NewReader(file)
	err = c.Stor(filename, reader)
	if err != nil {
		log.Println(err)
		return
	}
	if err := c.Quit(); err != nil {
		log.Println(err)
		return
	}
	log.Printf("Sendout done: %s", sourceFile)

	defer file.Close()
	defer os.Remove(sourceFile)
}
