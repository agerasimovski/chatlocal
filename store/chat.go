package store

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxTitleLen = 50

type ChatMessage struct {
	Sender string `json:"sender"`
	Text   string `json:"text"`
	Type   string `json:"type"`
	Time   string `json:"time"`
}

type ChatStore struct {
	dir string
}

func NewChatStore(dataDir string) (*ChatStore, error) {
	dir := filepath.Join(dataDir, "chats")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &ChatStore{dir: dir}, nil
}

func (c *ChatStore) userDir(userID string) string {
	return filepath.Join(c.dir, userID)
}

func (c *ChatStore) chatPath(userID, chatID string) string {
	return filepath.Join(c.userDir(userID), chatID+".json.gz")
}

func (c *ChatStore) metaPath(userID, chatID string) string {
	return filepath.Join(c.userDir(userID), chatID+".meta.json")
}

type ChatMeta struct {
	Title string `json:"title"`
}

func truncateTitle(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i] + "…"
		}
		n++
	}
	return s
}

func (c *ChatStore) Create(userID string) (chatID string, err error) {
	chatID = uuid.New().String()
	dir := c.userDir(userID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	p := c.chatPath(userID, chatID)
	empty := []ChatMessage{}
	return chatID, c.writeMessages(p, empty)
}

func (c *ChatStore) writeMessages(path string, msgs []ChatMessage) error {
	data, err := json.Marshal(msgs)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	_, err = gz.Write(data)
	if err != nil {
		return err
	}
	return gz.Close()
}

func (c *ChatStore) readMessages(path string) ([]ChatMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	var msgs []ChatMessage
	if err := json.NewDecoder(gz).Decode(&msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

// ChatInfo holds chat id and display title for listing.
type ChatInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (c *ChatStore) readMeta(userID, chatID string) (ChatMeta, error) {
	var m ChatMeta
	data, err := os.ReadFile(c.metaPath(userID, chatID))
	if err != nil {
		return m, err
	}
	_ = json.Unmarshal(data, &m)
	return m, nil
}

func (c *ChatStore) writeMeta(userID, chatID string, m ChatMeta) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(c.metaPath(userID, chatID), data, 0600)
}

func (c *ChatStore) List(userID string) ([]string, error) {
	infos, err := c.ListWithTitles(userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(infos))
	for i := range infos {
		ids[i] = infos[i].ID
	}
	return ids, nil
}

func (c *ChatStore) ListWithTitles(userID string) ([]ChatInfo, error) {
	dir := c.userDir(userID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []ChatInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json.gz") {
			continue
		}
		chatID := strings.TrimSuffix(name, ".json.gz")
		title := ""
		if m, err := c.readMeta(userID, chatID); err == nil && m.Title != "" {
			title = m.Title
		}
		if title == "" {
			title = "New chat"
		}
		result = append(result, ChatInfo{ID: chatID, Title: title})
	}
	return result, nil
}

func (c *ChatStore) Get(userID, chatID string) ([]ChatMessage, error) {
	p := c.chatPath(userID, chatID)
	return c.readMessages(p)
}

func (c *ChatStore) Append(userID, chatID string, msgs ...ChatMessage) error {
	p := c.chatPath(userID, chatID)
	existing, err := c.readMessages(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		existing = nil
	}
	// Set title from first user message when this is the first content
	if len(existing) == 0 && len(msgs) > 0 {
		for _, m := range msgs {
			if m.Sender == "You" && strings.TrimSpace(m.Text) != "" {
				title := truncateTitle(m.Text, maxTitleLen)
				_ = c.writeMeta(userID, chatID, ChatMeta{Title: title})
				break
			}
		}
	}
	existing = append(existing, msgs...)
	return c.writeMessages(p, existing)
}

func (c *ChatStore) Delete(userID, chatID string) error {
	chatFile := c.chatPath(userID, chatID)
	metaFile := c.metaPath(userID, chatID)
	err := os.Remove(chatFile)
	_ = os.Remove(metaFile) // best-effort
	return err
}
