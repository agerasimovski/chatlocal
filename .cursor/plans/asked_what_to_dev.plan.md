---
name: User accounts and chat storage
overview: Add authentication (login/registration), session-based access control, and per-user chat persistence with one gzip-compressed JSON file per chat. The frontend will switch from localStorage to server-backed chats and include login/register UI.
todos: []
isProject: false
---

# User accounts and compressed chat storage

## Current state

- **Backend:** [main.go](main.go) serves GET `/` (view) and POST `/prompt` (streaming LLM via [llmapi/ollama.go](llmapi/ollama.go)); no auth, no server-side chat persistence.
- **Frontend:** [view.html](view.html) stores messages only in `localStorage` and POSTs to `/prompt` with no user context.

## Target behavior

- Each user has login credentials (username + password); registration and login endpoints.
- Authenticated users have a private space: chats are stored only for that user and not shared.
- Each chat is stored in a **separate compressed file** (e.g. one `.json.gz` per chat under a per-user directory).

## Architecture (high level)

```mermaid
flowchart LR
  subgraph client [Frontend]
    Login[Login/Register]
    ChatUI[Chat UI]
    Login --> ChatUI
  end
  subgraph server [Backend]
    Auth[Auth middleware]
    Sessions[Session store]
    Users[User store]
    Chats[Chat store]
    Auth --> Sessions
    Auth --> Users
    Prompt[/prompt]
    ChatsAPI[/chats]
    Auth --> Prompt
    Auth --> ChatsAPI
    Prompt --> Chats
    ChatsAPI --> Chats
  end
  subgraph storage [File storage]
    UserFiles[users.json or users dir]
    ChatFiles["data/chats/{userID}/*.json.gz"]
  end
  Users --> UserFiles
  Chats --> ChatFiles
  client --> server
```

## 1. User store and credentials

- **Location:** New package or types in the main app (e.g. `store` or inline in `main`).
- **Storage:** File-based to avoid new DB dependency: e.g. `data/users.json` (or one file per user under `data/users/`) holding:
  - username (unique), password hash (bcrypt), internal user ID (UUID).
- **Libraries:** `golang.org/x/crypto/bcrypt` for hashing; optional: `github.com/google/uuid` for IDs.
- **Endpoints:**
  - `POST /register` — body: `{ "username", "password" }`; validate; hash password; persist user; then log in (set session) and return 201 or error.
  - `POST /login` — body: `{ "username", "password" }`; lookup user; compare hash; set session; return 200 or 401.
  - `POST /logout` — clear session; return 200.

## 2. Session management

- **Mechanism:** Cookie-based session ID with server-side store (in-memory map or file-based so restarts don’t drop everyone; e.g. `data/sessions/`).
- **Middleware:** `requireAuth(h http.HandlerFunc)` that:
  - reads session cookie;
  - if invalid/missing → redirect GET requests to a login page and return 401 for API requests (e.g. `/prompt`, `/chats`);
  - if valid → set user ID in request context and call `h`.
- **Protected routes:** `/`, `/prompt`, and all `/chats/*` go through this middleware (or only `/prompt` and `/chats`; `/` can show login vs chat based on auth).

## 3. Chat storage (separate compressed files)

- **Layout:** One directory per user, one file per chat:
  - `data/chats/{userID}/{chatID}.json.gz`
  - `chatID`: UUID or unique string; `userID`: from user store.
- **Format per file:** Gzip-compressed JSON array of messages, e.g.:
  - `[{ "sender", "text", "type", "time" }, ...]`
- **Operations:**
  - **Create chat:** Generate new `chatID`, create empty or initial message array, write to `data/chats/{userID}/{chatID}.json.gz`.
  - **Append messages:** Read file, decompress, decode JSON, append user + assistant messages, re-encode and gzip write.
  - **List chats:** List files in `data/chats/{userID}/` and return list of `chatID` (and optional metadata, e.g. first line or timestamp from filename or a small sidecar).
  - **Load chat:** Read and decompress `{chatID}.json.gz`, return JSON array.
- **Libraries:** stdlib `compress/gzip`, `encoding/json`; no extra deps if chat IDs are UUID from `github.com/google/uuid` or time-based IDs.

## 4. API changes

- **POST /prompt**
  - Require auth (middleware).
  - Body: e.g. `{ "text", "chatId" }` (chatId optional; if missing, use default or create one).
  - After streaming LLM response, append user message and full assistant response to the chat file for the authenticated user (same semantics as today: one streamed response per request).
- **GET /chats** — list chat IDs (and optionally titles/dates) for the current user.
- **GET /chats/{id}** — return message array for that chat (auth, and ensure chat belongs to user).
- **POST /chats** — create new chat; return `{ "chatId" }` (and optionally initial empty messages).

Ensure all chat operations validate that `chatID` belongs to the authenticated user (path traversal safe: use `userID` from session and only read/write under `data/chats/{userID}/`).

## 5. Frontend changes ([view.html](view.html))

- **Login/register:** Add a login page (or section) with username/password and “Register” / “Login” actions calling `POST /register` and `POST /login`; on success, redirect or switch to chat view.
- **Auth on load:** On app load, call a small “whoami” or “session check” endpoint (e.g. GET `/me` or rely on 401 from `/chats`); if unauthenticated, show login/register; otherwise show chat.
- **Chat list:** Sidebar or dropdown to list chats from GET `/chats` and switch current chat.
- **Current chat:** Load messages from GET `/chats/{id}` instead of `localStorage`; “New chat” calls POST `/chats` and then GET `/chats/{id}` for the new id.
- **Sending message:** POST `/prompt` with `{ "text", "chatId" }` (and credentials via cookie); append sent + received messages to the current chat in the UI; server will persist to the compressed file.
- **Credentials:** Send cookies with `fetch` (use `credentials: 'include'` for same-origin).

## 6. File and directory layout (summary)

- `data/users.json` (or `data/users/*.json`) — user records (username, password hash, user ID).
- `data/sessions/` (or in-memory map) — session ID → user ID.
- `data/chats/{userID}/{chatID}.json.gz` — one compressed JSON file per chat per user.

## 7. Dependencies (go.mod)

- Add at least: `golang.org/x/crypto` (bcrypt).
- Optional: `github.com/google/uuid` for user and chat IDs; otherwise use `crypto/rand` or time-based IDs.

## 8. Backward compatibility / migration

- Existing users have no account: after deployment, first visit shows login/register; no automatic migration of localStorage chats (optional: add “import from browser” later).
- Existing `/prompt` without auth will return 401 once middleware is applied; frontend must be updated to use login and `chatId` in parallel.

## Implementation order

1. Add user store (structs, file read/write, bcrypt), registration and login handlers, session store and middleware.
2. Add chat store (create/list/load/append, gzip per file), wire `/chats` and `/chats/{id}` and POST `/chats`.
3. Update POST `/prompt` to require auth and optional `chatId`, and append to compressed chat file after stream.
4. Add login/register UI and session check; replace localStorage with GET/POST chats and prompt with `chatId` in view.html.
