package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	logg "github.com/blind3dd/server-upload/pkg/logger"
)

type Value struct {
}

func newClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
			DisableKeepAlives:  false,
			WriteBufferSize:    8 * 1024,
			ForceAttemptHTTP2:  true,
		},
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New(strconv.Itoa(http.StatusTooManyRequests))
			}
			return nil
		},
	}
}

func SendRequest(ctx context.Context, file *os.File) (string, error) {
	client := newClient()
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/", file)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Authorization", "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==")

	res, err := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(res.Body)
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	//fmt.Println(string(data))
	return string(data), nil
}

func Stream(ctx context.Context, file *os.File, out chan<- string) error {
	for {
		v, err := SendRequest(ctx, file)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- v:
			fmt.Println(out)
			return nil
		}
	}
}

func main() {
	var ctx = context.TODO()
	logger := logg.NewLogger()

	logger.Println("attempting to prepare the data from file for the stream")
	file, err := os.Open("../../ingest.txt")
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)
	if err != nil {
		log.Fatal(err)
	}

	d, err := SendRequest(ctx, file)
	if err != nil {
		logger.Errorf("failed to send the request to the server, error: %v", err)
	}
	fmt.Println(d)
}
