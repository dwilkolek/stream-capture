package main

import (
	"bufio"
	"context"
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

	"github.com/jlaffaye/ftp"
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
		log.Printf("using ffmpeg from arg: %s\n", ffmpegPath)
	}

	streamUrl := os.Getenv("STREAM_URL")
	if streamUrl == "" {
		streamUrl = os.Args[2]
		log.Printf("using streamUrl from arg: %s\n", streamUrl)
	}

	cronString := os.Getenv("CRON")
	if cronString == "" {
		cronString = os.Args[3]
		log.Printf("using cron from arg: %s\n", cronString)
	}

	rec_timeout := os.Getenv("REC_TIMEOUT")
	if rec_timeout == "" {
		rec_timeout = os.Args[4]
		log.Printf("using timeout from arg: %s\n", rec_timeout)
	}

	ftp_location_str := os.Getenv("FTP")
	if ftp_location_str == "" {
		ftp_location_str = os.Args[5]
		log.Printf("using ftpLocation from args\n")
	}

	ftp_location := parseFtpString(ftp_location_str)
	fmt.Printf("Cron setup\nFfmpeg: %s\nStream: %s\nCron: %s\nRecording timeout:%s\nFtp: %s\n", ffmpegPath, streamUrl, cronString, rec_timeout, ftp_location.host+ftp_location.path)
	cron := cron.New(cron.WithSeconds())
	cron.AddJob(cronString, CaptureJob{
		ffmpegPath:   ffmpegPath,
		streamUrl:    streamUrl,
		rec_timeout:  rec_timeout,
		ftp_location: ftp_location,
	})
	cron.Start()

	fmt.Printf("DONE.\n\n")
}

func parseFtpString(ftpString string) FtpLocation {
	ftpReg := regexp.MustCompile(`(.*)\:(.*)@([A-Z\.\-a-z]+)(/.*)`)
	matches := ftpReg.FindStringSubmatch(ftpString)
	return FtpLocation{
		user: matches[1],
		pass: matches[2],
		host: matches[3],
		path: matches[4],
	}
}

type FtpLocation struct {
	host string
	user string
	pass string
	path string
}
type CaptureJob struct {
	ffmpegPath   string
	streamUrl    string
	rec_timeout  string
	ftp_location FtpLocation
}

func (captureJob CaptureJob) Run() {
	captureJob.process()
}
func (captureJob CaptureJob) process() {
	log.Println("Capture started")
	timeout, err := strconv.Atoi(captureJob.rec_timeout)
	filename := time.Now().Format("2006-01-02T15-04-05") + "-" + captureJob.rec_timeout
	if err != nil {
		log.Panicf("Failed to read timeout %s", captureJob.rec_timeout)
	}
	context_timeout := time.Duration(timeout+30) * time.Second
	outputFile := filepath.Join("out", fmt.Sprintf("%s.mp3", filename))
	log.Printf("Capture to file:: %s", outputFile)

	context, _ := context.WithTimeout(context.Background(), context_timeout)
	cmd := exec.CommandContext(context, captureJob.ffmpegPath, "-i", captureJob.streamUrl, "-t", captureJob.rec_timeout, outputFile, "-y")
	log.Printf("Executing: %s\n", cmd.String())
	err = cmd.Run()
	if err != nil {
		log.Println(err)
	}
	fmt.Println("Capture done")

	captureJob.sendToFtp(outputFile)
}

func (captureJob CaptureJob) sendToFtp(sourceFile string) {
	success := false
	c, err := ftp.Dial(captureJob.ftp_location.host+":21", ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Println(err)
		return
	}

	err = c.Login(captureJob.ftp_location.user, captureJob.ftp_location.pass)
	if err != nil {
		log.Println(err)
		return
	}

	dirs := strings.Split(captureJob.ftp_location.path, "/")
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
	success = true
	if err := c.Quit(); err != nil {
		log.Println(err)
		return
	}
	log.Printf("Sendout done: %s", sourceFile)

	defer file.Close()
	defer func() {
		if success {
			go func() {
				log.Printf("Removing file: %s", sourceFile)
				e := os.Remove(sourceFile)
				if e != nil {
					log.Println("Failed to delete file", e)
				}
			}()
		}
	}()
}
