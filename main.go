package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/agerasimovski/chatlocal/llmapi"
	"github.com/agerasimovski/chatlocal/store"
)

var (
	web  = flag.String("web", "localhost:8080", "Web server")
	data = flag.String("data", "data", "Data directory for users and chats")
	llm  = flag.String("llm", "localhost:11434/api/generate", "LLM server")
	model = flag.String("model", "gemma3", "LLM model")
)

type promptBody struct {
	Text   string `json:"text"`
	ChatID string `json:"chatId"`
}

func promptLLM(text string) (*http.Response, error) {
	request := ollama.Request{Model: *model, Prompt: text}
	return request.SendRequest("http://" + *llm)
}

func responseStream(w http.ResponseWriter, httpResponse *http.Response) error {
	return ollama.GetResponse(httpResponse, w)
}

// teeResponseWriter writes to both the client and a buffer (for saving the full response).
type teeResponseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (t *teeResponseWriter) Write(p []byte) (n int, err error) {
	t.buf.Write(p)
	return t.ResponseWriter.Write(p)
}

func (t *teeResponseWriter) Flush() {
	if f, ok := t.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func registerHandler(users *store.UserStore, sessions *store.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		email := strings.TrimSpace(strings.ToLower(body.Username))
		if _, err := mail.ParseAddress(email); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "username must be a valid email address"})
			return
		}
		if len(body.Password) < 8 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "password must be at least 8 characters long"})
			return
		}
		u, err := users.Register(email, body.Password)
		if err != nil {
			if err == store.ErrUserExists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "email already exists"})
				return
			}
			if err == store.ErrInvalidCredentials {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
				return
			}
			log.Println("register:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		sid, err := sessions.Create(u.ID)
		if err != nil {
			log.Println("session create:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     store.SessionCookieName,
			Value:    sid,
			Path:     "/",
			MaxAge:   7 * 24 * 3600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"username": u.Username})
	}
}

func loginHandler(users *store.UserStore, sessions *store.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		email := strings.TrimSpace(strings.ToLower(body.Username))
		u, err := users.Login(email, body.Password)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		sid, err := sessions.Create(u.ID)
		if err != nil {
			log.Println("session create:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     store.SessionCookieName,
			Value:    sid,
			Path:     "/",
			MaxAge:   7 * 24 * 3600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"username": u.Username})
	}
}

func logoutHandler(sessions *store.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cookie, _ := r.Cookie(store.SessionCookieName)
		if cookie != nil {
			_ = sessions.Delete(cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     store.SessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		w.WriteHeader(http.StatusOK)
	}
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("login.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = t.Execute(w, nil)
}

func promptHandler(chats *store.ChatStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		var body promptBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.Text == "" {
			http.Error(w, "text required", http.StatusBadRequest)
			return
		}
		userID := store.UserIDFromContext(r.Context())
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		chatID := strings.TrimSpace(body.ChatID)
		if chatID == "" {
			var err error
			chatID, err = chats.Create(userID)
			if err != nil {
				log.Println("chat create:", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
		httpResponse, err := promptLLM(body.Text)
		if err != nil {
			log.Println("prompt:", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		defer httpResponse.Body.Close()
		// Set headers before any write (first Write sends headers)
		w.Header().Set("X-Chat-Id", chatID)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		buf := new(bytes.Buffer)
		tee := &teeResponseWriter{ResponseWriter: w, buf: buf}
		if err := responseStream(tee, httpResponse); err != nil {
			log.Println("response:", err)
			return
		}
		now := time.Now().Format("3:04 PM")
		assistantText := strings.TrimSpace(buf.String())
		err = chats.Append(userID, chatID,
			store.ChatMessage{Sender: "You", Text: body.Text, Type: "sent", Time: now},
			store.ChatMessage{Sender: "LLM", Text: assistantText, Type: "received", Time: now},
		)
		if err != nil {
			log.Println("chat append:", err)
		}
	}
}

func meHandler(users *store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := store.UserIDFromContext(r.Context())
		u := users.ByID(userID)
		if u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": u.Username})
	}
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
	t, err := template.ParseFiles("view.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	_ = t.Execute(w, nil)
}

func chatsHandler(chats *store.ChatStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := store.UserIDFromContext(r.Context())
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		path := strings.TrimSuffix(r.URL.Path, "/")
		if path == "/chats" {
			switch r.Method {
			case http.MethodGet:
				infos, err := chats.ListWithTitles(userID)
				if err != nil {
					log.Println("chats list:", err)
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"chats": infos})
				return
			case http.MethodPost:
				chatID, err := chats.Create(userID)
				if err != nil {
					log.Println("chats create:", err)
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]string{"chatId": chatID})
				return
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
		}
		if strings.HasPrefix(path, "/chats/") && len(path) > 7 {
			chatID := path[7:]
			if chatID == "" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			switch r.Method {
			case http.MethodGet:
				msgs, err := chats.Get(userID, chatID)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						http.Error(w, "not found", http.StatusNotFound)
						return
					}
					log.Println("chats get:", err)
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"messages": msgs})
				return
			case http.MethodDelete:
				if err := chats.Delete(userID, chatID); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						http.Error(w, "not found", http.StatusNotFound)
						return
					}
					log.Println("chats delete:", err)
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func loginHandlerCombined(users *store.UserStore, sessions *store.SessionStore) http.HandlerFunc {
	loginAPI := loginHandler(users, sessions)
	loginPage := loginPageHandler
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			loginAPI(w, r)
			return
		}
		if r.Method == http.MethodGet {
			loginPage(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	flag.Parse()
	fmt.Println("Web:", *web)
	fmt.Println("LLM:", *llm, *model)
	fmt.Println("Data:", *data)

	users, err := store.NewUserStore(*data)
	if err != nil {
		log.Fatal("user store:", err)
	}
	sessions, err := store.NewSessionStore(*data)
	if err != nil {
		log.Fatal("session store:", err)
	}
	chats, err := store.NewChatStore(*data)
	if err != nil {
		log.Fatal("chat store:", err)
	}

	http.HandleFunc("/register", registerHandler(users, sessions))
	http.HandleFunc("/login", loginHandlerCombined(users, sessions))
	http.HandleFunc("/logout", logoutHandler(sessions))
	http.HandleFunc("/me", store.RequireAuth(users, sessions, meHandler(users)))
	http.HandleFunc("/chats", store.RequireAuth(users, sessions, chatsHandler(chats)))
	http.HandleFunc("/chats/", store.RequireAuth(users, sessions, chatsHandler(chats)))
	http.HandleFunc("/", store.RequireAuth(users, sessions, viewHandler))
	http.HandleFunc("/prompt", store.RequireAuth(users, sessions, promptHandler(chats)))
	log.Fatal(http.ListenAndServe(*web, nil))
}
