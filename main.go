package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	logger "github.com/blind3dd/server-upload/pkg/logger"
	wj "github.com/blind3dd/server-upload/pkg/writer"
)

type Adapter func(http.Handler) http.Handler

type Response struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

func (a Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.ServeHTTP(w, r)
}

func wrap(h http.Handler, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

func muxAdapter(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}
}

func authCheck(logger *logrus.Entry) Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			logger.Println("performing auth check")
			user, pass, ok := r.BasicAuth()
			if !ok {
				logger.Error("failed to read request header")
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(Response{
					Code: 400, Message: "failed to read request header"},
				); err != nil {
					panic(err)
				}
				return
			}
			var user32b, pass32b [32]byte
			user32b = sha256.Sum256([]byte(user))
			pass32b = sha256.Sum256([]byte(pass))
			expectedUser := sha256.Sum256([]byte(us))
			expectedPass := sha256.Sum256([]byte(pw))
			if user32b != expectedUser || pass32b != expectedPass {
				logger.Error("auth check failed")
				w.WriteHeader(http.StatusUnauthorized)
				if err := json.NewEncoder(w).Encode(Response{
					Code: http.StatusUnauthorized, Message: "authorization invalid"},
				); err != nil {
					panic(err)
				}
				return
			} else {
				logger.Println("auth check successful")
			}
			h.ServeHTTP(w, r)
		})
	}
}

func indexHandler() Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(struct {
				Code    int32  `json:"code"`
				Message string `json:"message"`
			}{
				Code:    int32(http.StatusOK),
				Message: "HB_OK",
			})
			if err != nil {
				panic(err)
			}

			h.ServeHTTP(w, r)
		})
	}
}

type Reader struct {
	r       io.Reader
	buf     bytes.Buffer
	size    int64
	maxSize int64
}

func (r *Reader) Read(b []byte) (n int, err error) {
	n, err = r.r.Read(b)
	if n > 0 {
		r.size += int64(n)
		if n, err := r.buf.Write(b[:n]); err != nil {
			return n, err
		}
	}
	return n, err
}

func processReader(logger *logrus.Entry, reader *Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.New(fmt.Sprintf("failed to parse token: error: %v", err))
		}

		logger.Printf("token: %v", token)
	}
}

func (r *Reader) GetCurrentBuffer() bytes.Buffer {
	return r.buf
}

func jsonHandler(logger *logrus.Entry) Adapter {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
				logger.Error("failed to read data from the stream")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				if err := json.NewEncoder(w).Encode(Response{
					Code: http.StatusUnsupportedMediaType, Message: "unsupported media type"},
				); err != nil {
					panic(err)
				}
				return
			}
			sizeLimit := int64(64 * 1024)
			b := bytes.Buffer{}
			b.Write(make([]byte, 0, sizeLimit))
			reader := Reader{
				r:       r.Body,
				buf:     b,
				size:    0,
				maxSize: sizeLimit,
			}
			err := processReader(logger, &reader)
			if err != nil {
				panic(err)
			}
			rb := reader.GetCurrentBuffer()
			logger.Println(len(rb.Bytes()))
			r.Body = io.NopCloser(&rb)
			data, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					panic(err)
				}
			}(r.Body)
			logger.Println("attempting to write data from handler")
			if _, err := w.Write(data); err != nil {
				panic(err)
			}

			//h.ServeHTTP(w, r)
		})
	}
}

func aHandler(logger *logrus.Entry) Adapter {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Println("attempting to read data from the stream")
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/octet-stream" {
				logger.Error("failed to read data from the stream")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				if err := json.NewEncoder(w).Encode(Response{
					Code: http.StatusUnsupportedMediaType, Message: "unsupported media type"},
				); err != nil {
					panic(err)
				}
				return
			}
			data, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					panic(err)
				}
			}(r.Body)
			logger.Println("attempting to write data from handler")
			if _, err := w.Write(data); err != nil {
				panic(err)
			}
			logger.Println("server successfully handled provided stream data")
		})
	}
}

func main() {
	mux := http.NewServeMux()
	logger := logger.NewLogger()
	mux.Handle("/", wrap(indexHandler(), aHandler(logger), authCheck(logger)))
	//mux.HandleFunc("/v1/handle", aHandler)
	logger.Printf("starting http server at %s", lisAddr)
	ctx := context.Background()
	srv := &http.Server{
		Addr:         lisAddr,
		Handler:      muxAdapter(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil {
			logger.Fatalf("failed to serve the http server, reason: %v", err)
		}
	}()

	logger.Printf("http server is running at port: %s", srv.Addr)
	//wg.Wait()
	// TODO(publish to queue rather than file)
	wg.Add(1)
	defer wg.Done()
	j := wj.NewJournal()
	j.AddEntry("I found a deer")
	j.AddEntry("I cried")
	j.AddEntry("Then I Realized It was not the deer but a horse")
	j.AddEntry("not a donkey eater after all. Phew.")
	j.AddEntry("beer? Nope. Telemetry")
	logger.Printf("entries written: %v", j.Stringer())
	wj.SaveToFile(j, "ingest.txt")
	p := wj.Persistence{LineSeparator: "\r\n"}
	p.SaveToFile(j, "ingest_windows.txt")
	//wg.Wait()

	wg.Add(1)
	defer wg.Done()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		err := srv.Shutdown(ctx)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}()
	wg.Wait()
}

var (
	us      = "Aladdin"
	pw      = "open sesame"
	lisAddr = ":8080"
)

//wg.Add(1)
//defer wg.Done()
//logger.Println("attempting to send the request to server")
////outputCh := make(chan string)
//d, err := cl.SendRequest(ctx, file)
//if err != nil {
//	logger.Errorf("failed to send the request to the server, error: %v", err)
//}
//fmt.Println(d)
