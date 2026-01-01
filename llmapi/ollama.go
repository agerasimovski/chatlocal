package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Request Ollama JSON request
type Request struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

func (request Request) SendRequest(url string) (*http.Response, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		return nil, err
	}

	return httpResponse, nil
}

type Response struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

func GetResponse(httpResponse *http.Response, writer http.ResponseWriter) error {
	var sentence string
	scanner := bufio.NewScanner(httpResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()

		var data Response
		err := json.Unmarshal([]byte(line), &data)
		if err != nil {
			return err
		}
		if data.Response == "\n\n" || data.Response == "\n" || data.Done {
			_, err := fmt.Fprintf(writer, "%s\n\n", sentence)
			if err != nil {
				log.Println(err)
				return err
			}
			writer.(http.Flusher).Flush()
			sentence = ""
		} else {
			sentence += data.Response
		}
		// fmt.Print(data.Response)
	}

	return nil
}
