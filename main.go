package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/agerasimovski/llmapi"
)

var (
	web = flag.String("web", "localhost:8080", "Web server")
	// cert = flag.String("cert", "cert.pem", "Web certificate")
	// key = flag.String("key", "key.pem", "Certificate key")
	llm   = flag.String("llm", "localhost:11434/api/generate", "LLM server")
	model = flag.String("model", "gemma3", "LLM model")
)

func prompt(r *http.Request) (*http.Response, error) {
	type Message struct {
		Text string `json:"text"`
	}
	var message Message

	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	request := ollama.Request{Model: *llm, Prompt: message.Text}
	httpResponse, err := request.SendRequest("http://" + *llm)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return httpResponse, nil
}

func response(w http.ResponseWriter, httpResponse *http.Response) error {
	err := ollama.GetResponse(httpResponse, w)
	if err != nil {
		log.Println(err)
		return err
	}

	err = httpResponse.Body.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func promptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusBadRequest)
		log.Println("Method:", r.Method)
		return
	}

	httpResponse, err := prompt(r)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		log.Println("prompt:", err)
		return
	}

	err = response(w, httpResponse)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		log.Println("response", err)
	}

	return
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
	t, err := template.ParseFiles("view.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}

func main() {
	flag.Parse()
	fmt.Println("Web:", *web)
	fmt.Println("LLM:", *llm, *model)

	http.HandleFunc("/", viewHandler)
	http.HandleFunc("/prompt", promptHandler)
	log.Fatal(http.ListenAndServe(*web, nil))
	// log.Fatal(http.ListenAndServeTLS(*webUrl, "cert.pem", "key.pem", nil))
}
