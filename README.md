# chatlocal

A lightweight, self-contained web application that provides a browser-based chat interface for interacting with locally deployed LLMs via [Ollama](https://ollama.com/). Built entirely in Go with a vanilla HTML/CSS/JavaScript frontend -- no heavy frameworks, no external databases.

Custom local LLM deployment is an increasingly valuable use case for organizations that want to keep their data in-house, maintain full control over their AI infrastructure, and avoid sending sensitive information to third-party cloud services. This project demonstrates that Go is an effective and performant choice for building such applications.

## Features

- **Local-first privacy** -- All data stays on your machine. Chat history, user accounts, and sessions are stored as files on disk. No cloud dependencies.
- **Streaming responses** -- LLM output is streamed to the browser in real time, sentence by sentence, for a responsive chat experience.
- **Multi-user support** -- Email-based registration and login with bcrypt-hashed passwords and session-based authentication (HTTP-only cookies, 7-day expiration).
- **Persistent chat history** -- Conversations are saved as gzip-compressed JSON files, organized per user. Create, browse, and delete past chats from the sidebar.
- **Auto-generated chat titles** -- Each conversation is automatically titled based on the first message.
- **Configurable LLM backend** -- Point to any Ollama-compatible endpoint and choose your model at startup.
- **Minimal dependencies** -- Only two external Go modules: `golang.org/x/crypto` (bcrypt) and `github.com/google/uuid`.
- **Single binary deployment** -- Compile once, run anywhere. No runtime dependencies beyond Ollama.

## Architecture

```
┌──────────────┐       HTTP        ┌──────────────────────┐     Ollama API     ┌─────────┐
│   Browser    │◄─────────────────►│   chatlocal (Go)     │◄──────────────────►│ Ollama  │
│  (HTML/JS)   │  streaming SSE    │                      │   /api/generate    │ Server  │
└──────────────┘                   │  ┌────────────────┐  │                    └─────────┘
                                   │  │  store/         │  │
                                   │  │  - users.go     │  │
                                   │  │  - sessions.go  │  │──► data/
                                   │  │  - chat.go      │  │    ├── users.json
                                   │  │  - auth.go      │  │    ├── sessions/
                                   │  └────────────────┘  │    └── chats/{userId}/
                                   │  ┌────────────────┐  │
                                   │  │  llmapi/        │  │
                                   │  │  - ollama.go    │  │
                                   │  └────────────────┘  │
                                   └──────────────────────┘
```

**Request flow:** User submits a message via the browser. The Go server authenticates the request, forwards the prompt to Ollama's `/api/generate` endpoint, and streams the response back to the client in real time. Both the user message and the LLM response are persisted to the chat store.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Main chat interface (requires auth) |
| `GET/POST` | `/login` | Login page and authentication |
| `POST` | `/register` | User registration |
| `POST` | `/logout` | Session termination |
| `GET` | `/me` | Current user info |
| `POST` | `/prompt` | Send message to LLM (streaming response) |
| `GET` | `/chats` | List user's chats |
| `POST` | `/chats` | Create a new chat |
| `GET` | `/chats/{id}` | Get chat messages |
| `DELETE` | `/chats/{id}` | Delete a chat |

## Prerequisites

- [Go](https://go.dev/learn/) 1.21 or later
- [Ollama](https://docs.ollama.com/) installed and running on your machine
- An Ollama model pulled locally (e.g., `ollama pull gemma3`)

## Build and Run

Clone and build:

```bash
git clone https://github.com/agerasimovski/chatlocal.git
cd chatlocal
go build
```

Make sure the Ollama server is running:

```bash
systemctl status ollama
```

If it is not running, start it:

```bash
systemctl start ollama
```

Run chatlocal:

```bash
./chatlocal -web localhost:8080 -data data -llm localhost:11434/api/generate -model gemma3
```

Then open [http://localhost:8080](http://localhost:8080) in your browser, register an account, and start chatting.

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-web` | `localhost:8080` | Address and port for the web server |
| `-data` | `data` | Directory for storing user data, sessions, and chats |
| `-llm` | `localhost:11434/api/generate` | Ollama API endpoint |
| `-model` | `gemma3` | LLM model name to use |

## Project Structure

```
chatlocal/
├── main.go          # Application entry point, HTTP routing
├── go.mod           # Go module definition
├── view.html        # Main chat interface (single-page app)
├── login.html       # Login and registration page
├── store/           # Data persistence layer
│   ├── users.go     #   User registration and login
│   ├── sessions.go  #   Session management
│   ├── chat.go      #   Chat storage (gzip-compressed JSON)
│   ├── auth.go      #   Authentication middleware
│   └── errors.go    #   Custom error definitions
├── llmapi/          # LLM integration
│   └── ollama.go    #   Ollama streaming API client
└── data/            # Runtime data (created automatically)
    ├── users.json
    ├── sessions/
    └── chats/
```

## Developed By

This application was developed by **Cursor AI** (powered by Claude), an AI coding assistant by [Anysphere](https://anysphere.inc/). The entire codebase -- backend, frontend, architecture, and documentation -- was authored through AI-assisted development within the Cursor IDE.

## License

This project is provided as-is for educational and experimental purposes.
