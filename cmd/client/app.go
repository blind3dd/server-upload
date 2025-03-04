package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	logg "github.com/blind3dd/server-upload/pkg/logger"
)

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
	//TODO convert above to less mock-like entity along with server code for auth handler
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

func SendRequestChunk(ctx context.Context, body *bytes.Buffer) (string, error) {
	client := newClient()
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/", body)
	var headers = map[string][]string{
		"Content-Type":              {"application/octet-stream", "multipart/x-mixed-replace; boundary=test", "text/xml"},
		"Authorization":             {"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="},
		"Content-Transfer-Encoding": {"binary"},
		"Content-Disposition":       {"form-data; name=file", "attachment; filename=ingest.txt"},
	}
	for k, v := range headers {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	resp, err := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	//fmt.Println(string(data))
	return string(data), nil
}

func main() {
	var ctx = context.TODO()
	logger := logg.NewLogger()
	filePath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		logger.Fatalf("could not get project path, error: %v", err)
	}
	pathFile := path.Join(path.Dir(string(filePath)), "server-upload", "ingest.txt")
	logger.Println("attempting to prepare the data from file for the stream")

	var wg sync.WaitGroup
	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		file, err := os.Open(pathFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer file.Close()
		respDataFile, err := SendRequest(ctx, file)
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println(respDataFile)
		//TODO move out of here
		jsonData := intoJson(respDataFile)
		fmt.Println(jsonData)
		//respData, err := SendRequestChunk(ctx, bytes.NewBufferString(jsonData))
		//if err != nil {
		//	logger.Fatal(err)
		//}
		//fmt.Println(respData)
	}(ctx)

	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		file, err := os.Open(pathFile)
		if err != nil {
			logger.Fatal(err)
		}
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		//chunk, _ := writer.CreatePart(h)
		chunk, _ := writer.CreateFormFile("file", path.Base(pathFile))
		io.Copy(chunk, file)
		writer.Close()
		resp, err := SendRequestChunk(ctx, body)
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println(resp)

	}(ctx)
	wg.Wait()
}

type intoTheStruct struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func intoJson(data string) string {
	var dataSlice = make([]intoTheStruct, 0)
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		keyVal := strings.Split(line, ":")
		dataSlice = append(dataSlice,
			intoTheStruct{
				Key:   strings.TrimSpace(keyVal[0]),
				Value: strings.TrimSpace(keyVal[1]),
			})
		// dataSlice[keyVal[0]] = keyVal[1]
	}
	bts, err := json.Marshal(dataSlice)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s", bts)
}

//func Stream(ctx context.Context, file *os.File, out chan<- string) error {
//	for {
//		v, err := SendRequest(ctx, file)
//		if err != nil {
//			return err
//		}
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		case out <- v:
//			fmt.Println(out)
//			return nil
//		}
//	}
//}
