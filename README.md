# llmwrap
An example web application that provides a browser-based interface for interacting with a locally deployed LLMs.

Custum local LLM deployment could be interesting use case for many organizations that want to keep their data inhouse, 
and in this small project I experiment with Go as an effective choice for building such applications.

## Build, install and run

If not already installed, install [Go](https://go.dev/learn/) development environment.

If not already installed, install [ollama](https://docs.ollama.com/) to run LLMs on your local machines.

Build the llmwarp:
```
git clone https://github.com/agerasimovski/llmwrap.git
cd llmwrap
go build
```
Make sure that Ollama server is running:
```
systemctl status ollama
```
Run llmwrap:
```
./llmwrap -web localhost:8080 -llm localhost:11434/api/generate -model gemma3
```

## Conclusion
To grow into a real-world application, we would need to add standard features such as account management, conversation history, file upload and processing, and professional UI.
However, aside from the UI, Go already provides robust standard packages that enable these features to be implemented in a timely manner.
