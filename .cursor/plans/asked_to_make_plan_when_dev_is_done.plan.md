---
name: Development Roadmap
overview: Development plan for chatlocal — a self-hosted web chat interface for local LLMs via Ollama. Covers current state, architecture, and planned improvements across backend, frontend, security, and UX.
todos: []
isProject: true
---

# chatlocal Development Roadmap

## Current State

**What it is:** A self-hosted web chat application that wraps locally running LLMs (via Ollama) with a clean UI, user authentication, and persistent chat storage.

**Tech stack:**

- **Backend:** Go 1.25 standard library HTTP server ([main.go](main.go))
- **Frontend:** Vanilla HTML/CSS/JS, no build step ([view.html](view.html), [login.html](login.html))
- **Storage:** File-based JSON and gzip-compressed JSON (no database)
- **LLM:** Ollama API integration ([llmapi/ollama.go](llmapi/ollama.go))

**What works today:**

- User registration and login with bcrypt password hashing
- Cookie-based session management (7-day expiry)
- Per-user chat creation, listing, loading, and deletion
- Streaming LLM responses from Ollama
- Chat messages persisted as gzip-compressed JSON files
- Responsive sidebar with chat history
- Auto-generated chat titles from first message

**Project structure:**

```
chatlocal/
  main.go              # HTTP server, routes, handlers
  login.html           # Login/register page
  view.html            # Chat interface
  go.mod               # Go module
  store/
    auth.go            # Auth middleware
    users.go           # User store (bcrypt, file-based)
    session.go         # Session store (file-based, 7-day TTL)
    chat.go            # Chat store (gzip JSON per chat)
    errors.go          # Error definitions
  llmapi/
    ollama.go          # Ollama API client
    go.mod             # Submodule
  data/
    users.json         # User credentials
    sessions/          # Session files
    chats/{userID}/    # Per-user chat files (.json.gz + .meta.json)
```

---

## Phase 1: Core UX Improvements

These are high-impact, low-effort changes that improve daily usability.

### 1.1 Markdown rendering in responses

- Render assistant messages as markdown (headings, bold, italic, lists, code blocks)
- Use a lightweight library (e.g. marked.js or markdown-it via CDN)
- Add syntax highlighting for code blocks (e.g. highlight.js)
- Keep user messages as plain text

### 1.2 Chat title editing

- Allow users to rename chats by clicking the title in the sidebar
- Add a `PUT /chats/{id}` or `PATCH /chats/{id}` endpoint to update metadata
- Update the `.meta.json` file with the new title

### 1.3 Streaming indicator

- Show a typing/thinking indicator while the LLM is generating
- Disable input during streaming to prevent duplicate sends
- Add a "Stop generating" button to cancel in-flight requests

### 1.4 Conversation context

- Currently each prompt is sent independently to Ollama with no history
- Send previous messages as context to the LLM for multi-turn conversations
- Add a configurable context window limit (e.g. last N messages or token count)
- Update [llmapi/ollama.go](llmapi/ollama.go) to accept conversation history

### 1.5 Empty state and error handling

- Better error messages when Ollama is not running or model is unavailable
- Connection status indicator in the UI
- Retry logic for transient LLM failures

---

## Phase 2: Multi-Model and Configuration

### 2.1 Model selection per chat

- Allow users to choose which Ollama model to use per chat
- Add a model selector dropdown in the input area or chat settings
- Store selected model in chat metadata
- Fetch available models from Ollama API (`GET /api/tags`)

### 2.2 System prompt / persona

- Allow users to set a system prompt per chat (e.g. "You are a helpful coding assistant")
- Store system prompt in chat metadata
- Prepend system prompt to LLM requests

### 2.3 Server configuration page

- Admin-accessible settings page for LLM server URL, default model, etc.
- Alternatively, expose current CLI flags as environment variables too

---

## Phase 3: Security Hardening

### 3.1 HTTPS support

- Add TLS flag to serve over HTTPS (e.g. `-tls-cert` and `-tls-key` flags)
- Or document reverse proxy setup (nginx/caddy) for production

### 3.2 CSRF protection

- Add CSRF tokens to state-changing requests (POST/PUT/DELETE)
- Or use SameSite=Strict cookies and validate Origin header

### 3.3 Rate limiting

- Add per-user rate limiting on `/prompt` to prevent abuse
- Simple in-memory token bucket or sliding window

### 3.4 Session cleanup

- Background goroutine to periodically clean expired session files
- Currently expired sessions only cleaned on access

### 3.5 Password requirements and reset

- Enforce stronger password rules (uppercase, number, special char)
- Add password change functionality (requires current password)
- Optional: email-based password reset (requires SMTP config)

---

## Phase 4: Advanced Features

### 4.1 Chat search

- Full-text search across all user chats
- Search endpoint: `GET /chats/search?q=...`
- Decompress and search through gzip files server-side
- Display results with highlighted matches

### 4.2 Chat export

- Export individual chats as plain text, markdown, or JSON
- Add export button per chat in the UI
- Endpoint: `GET /chats/{id}/export?format=txt|md|json`

### 4.3 File/image attachments

- Support uploading files or images with messages
- Store attachments alongside chat files
- Display images inline in the chat

### 4.4 Dark mode

- Add a dark/light mode toggle
- Store preference in a cookie or user settings
- CSS variables already in place, just need alternate values

### 4.5 Keyboard shortcuts

- `Ctrl+N` for new chat
- `Ctrl+/` to toggle sidebar
- Up arrow to edit last message
- `Escape` to cancel streaming

---

## Phase 5: Deployment and Operations

### 5.1 Docker support

- Add `Dockerfile` for easy deployment
- Add `docker-compose.yml` with chatlocal + Ollama
- Document environment variable configuration

### 5.2 Logging and monitoring

- Structured logging (JSON format option)
- Request logging middleware with timing
- Health check endpoint (`GET /health`)

### 5.3 Backup and restore

- Script or endpoint to backup `data/` directory
- Import/restore from backup archive
- Optional: scheduled backup to external storage

### 5.4 Multi-user administration

- Admin role for user management
- Ability to view/delete users
- Usage statistics per user

---

## Priority Order

For immediate development, the recommended order is:

1. **Conversation context** (1.4) -- without this, multi-turn chat is broken
2. **Markdown rendering** (1.1) -- major UX improvement for code responses
3. **Streaming indicator** (1.3) -- polish the chat experience
4. **Model selection** (2.1) -- flexibility for different tasks
5. **Dark mode** (4.4) -- quick win, CSS variables ready
6. **Docker support** (5.1) -- simplifies deployment
7. **HTTPS / security** (3.x) -- required before any non-local deployment

