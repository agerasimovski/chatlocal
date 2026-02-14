# chatlocal
An example web application that provides a browser-based interface for interacting with a locally deployed LLMs.

Custum local LLM deployment could be interesting use case for many organizations that want to keep their data inhouse, 
and in this small project I experiment with Go as an effective choice for building such applications.

## Build, install and run

If not already installed, install [Go](https://go.dev/learn/) development environment.

If not already installed, install [ollama](https://docs.ollama.com/) to run LLMs on your local machines.

Build the chatlocal:
```
git clone https://github.com/agerasimovski/chatlocal.git
cd chatlocal
go build
```
Make sure that Ollama server is running:
```
systemctl status ollama
```
Run chatlocal:
```
./chatlocal -web localhost:8080 -llm localhost:11434/api/generate -model gemma3
```
