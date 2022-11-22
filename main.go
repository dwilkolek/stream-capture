package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	fmt.Println("Current date and time is: ", time.Now())
	_ = os.Mkdir("out", os.ModePerm)
	go setupCron()
	server()
}

func server() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}
	handler := http.FileServer(http.Dir("out"))
	err := http.ListenAndServe(":"+port, handler)
	fmt.Println(err)
}

func setupCron() {
	ffmpegPath := os.Getenv("FFMPEG")
	if ffmpegPath == "" {
		ffmpegPath = os.Args[1]
		log.Printf("using ffmpeg from arg: %s", ffmpegPath)
	}

	streamUrl := os.Getenv("STREAM_URL")
	if streamUrl == "" {
		streamUrl = os.Args[2]
		log.Printf("using streamUrl from arg: %s", streamUrl)
	}

	cronString := os.Getenv("CRON")
	if cronString == "" {
		cronString = os.Args[3]
		log.Printf("using cron from arg: %s", cronString)
	}

	rec_timeout := os.Getenv("REC_TIMEOUT")
	if rec_timeout == "" {
		rec_timeout = os.Args[4]
		log.Printf("using timeout from arg: %s", rec_timeout)
	}

	fmt.Printf("Cron setup\nFfmpeg: %s\nStream: %s\nCron: %s\nRecording timeout:%s\n", ffmpegPath, streamUrl, cronString, rec_timeout)
	cron := cron.New(cron.WithSeconds())
	cron.AddJob(cronString, CaptureJob{
		ffmpegPath:  ffmpegPath,
		streamUrl:   streamUrl,
		rec_timeout: rec_timeout,
	})
	cron.Start()

	fmt.Printf("DONE.\n\n")
}

type CaptureJob struct {
	ffmpegPath  string
	streamUrl   string
	rec_timeout string
}

func (captureJob CaptureJob) Run() {
	captureJob.capture()
}
func (captureJob CaptureJob) capture() {
	log.Println("Capture started")
	timeout, err := strconv.Atoi(captureJob.rec_timeout)
	if err != nil {
		log.Panicf("Failed to read timeout %s", captureJob.rec_timeout)
	}
	context_timeout := time.Duration(timeout+30) * time.Second
	capture_date := time.Now().Format("2006-01-02")
	context, _ := context.WithTimeout(context.Background(), context_timeout)
	cmd := exec.CommandContext(context, captureJob.ffmpegPath, "-i", captureJob.streamUrl, "-t", captureJob.rec_timeout, fmt.Sprintf("./out/%s.mp3", capture_date), "-y")
	log.Printf("Executing: %s\n", cmd.String())
	err = cmd.Run()
	if err != nil {
		log.Println(err)
	}
	fmt.Println("Capture done")
}
