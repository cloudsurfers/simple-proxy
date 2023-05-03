package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var messageTemplate = `
{
  "text": "%s"
}`

func main() {
	integrationUri, ok := os.LookupEnv("INTEGRATION_URI")
	if !ok {
		log.Fatalf("ENV ERROR: 'INTEGRATION_URI' is missing")
	}
	originalPayload, ok := os.LookupEnv("RESPONSE_ON_SUCCESS")
	if !ok {
		log.Fatalf("ENV ERROR: 'ORIGINAL_PAYLOAD' is missing")
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		log.Printf("headers: %v", r.Header)
		log.Printf("method: %s", r.Method)
		log.Printf("url: %s", r.URL)
		bbytes, er := io.ReadAll(r.Body)
		if er != nil {
			log.Printf("error reading body: %v", er)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		_, er = w.Write([]byte("{'statusText': 'ok'}"))
		if er != nil {
			log.Printf("error writing response: %v", er)
			http.Error(w, "can't write response", http.StatusBadRequest)
			return
		}
		log.Printf("body: %s", bbytes)
		return
	})

	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("incoming request: %v", r)
		incomingBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		log.Printf("incoming body: %s", incomingBody)
		strgMessage := fmt.Sprintf("%s", incomingBody)
		strgMessage = strings.Replace(strgMessage, "\n", "\\n", -1)
		tmpMessage := []byte(fmt.Sprintf(messageTemplate, strgMessage))
		log.Printf("sending message: %s", tmpMessage)
		req, err := http.NewRequest(http.MethodPost, integrationUri, bytes.NewBuffer(tmpMessage))
		req.Header.Add("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("failed to call integration url %s with error %v", integrationUri, err)
			http.Error(w, "can't call integration url", http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()
		rBytes, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Printf("failed to read response body %v", err)
			http.Error(w, "can't read response body", http.StatusBadRequest)
			return
		}
		if resp.StatusCode > 299 {
			log.Printf("unexpected response status. HTTP %d. Sent SOAP body: %s", resp.StatusCode, string(rBytes))
			http.Error(w, "unexpected response status", http.StatusBadRequest)
			return
		}
		tmp := string(rBytes)
		if strings.Compare(tmp, "{\"success\":true}") == 0 {
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(originalPayload))
			if err != nil {
				log.Printf("error writing response: %v", err)
				http.Error(w, "can't write response", http.StatusBadRequest)
				return
			}
		}
	})
	http.ListenAndServe(":3000", nil)
}
