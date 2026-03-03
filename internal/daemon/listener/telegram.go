package listener

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramAdapter struct {
	bot          *tgbotapi.BotAPI
	token        string
	workspaceDir string
	allowedUsers []string
	name         string
}

func NewTelegramAdapter(token string, workspaceDir string, allowedUsers []string) (*TelegramAdapter, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &TelegramAdapter{
		bot:          bot,
		token:        token,
		workspaceDir: workspaceDir,
		allowedUsers: allowedUsers,
		name:         "telegram",
	}, nil
}

func (a *TelegramAdapter) Name() string {
	return a.name
}

func (a *TelegramAdapter) Listen(ctx context.Context, handler MessageHandler) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := a.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			msg := a.normalizeMessage(update.Message)
			if msg == nil {
				continue
			}

			if !a.isAllowed(msg.Sender) {
				log.Printf("User %s not allowed, ignoring", msg.Sender)
				continue
			}

			if err := handler(ctx, *msg); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}

func (a *TelegramAdapter) isAllowed(username string) bool {
	if len(a.allowedUsers) == 0 {
		return true
	}

	for _, user := range a.allowedUsers {
		user = strings.TrimPrefix(user, "@")
		if user == username {
			return true
		}
	}
	return false
}

func (a *TelegramAdapter) normalizeMessage(msg *tgbotapi.Message) *Message {
	if msg == nil {
		return nil
	}

	sender := msg.From.UserName
	if sender == "" {
		sender = msg.From.FirstName
		if msg.From.LastName != "" {
			sender += " " + msg.From.LastName
		}
	}

	if sender == "" {
		sender = fmt.Sprintf("user_%d", msg.From.ID)
	}

	content := msg.Text
	contentType := "text"

	if content == "" {
		if msg.Photo != nil {
			contentType = "non-text"
			content = a.handlePhoto(msg)
		} else if msg.Document != nil {
			contentType = "non-text"
			content = a.handleDocument(msg)
		} else if msg.Voice != nil {
			contentType = "non-text"
			content = a.handleVoice(msg)
		}
	}

	return &Message{
		Sender:      sender,
		Content:     content,
		ContentType: contentType,
		ChatID:      fmt.Sprintf("%d", msg.Chat.ID),
		Timestamp:   msg.Time(),
		Raw:         msg,
	}
}

func (a *TelegramAdapter) handlePhoto(msg *tgbotapi.Message) string {
	photo := msg.Photo[len(msg.Photo)-1]
	fileID := photo.FileID

	file, err := a.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		log.Printf("Failed to get photo file: %v", err)
		return ""
	}

	return a.downloadFile(file, "photo", ".jpg")
}

func (a *TelegramAdapter) handleDocument(msg *tgbotapi.Message) string {
	doc := msg.Document
	file, err := a.bot.GetFile(tgbotapi.FileConfig{FileID: doc.FileID})
	if err != nil {
		log.Printf("Failed to get document file: %v", err)
		return ""
	}

	ext := ".bin"
	if doc.FileName != "" {
		ext = filepath.Ext(doc.FileName)
		if ext == "" {
			ext = ".bin"
		}
	}
	return a.downloadFile(file, "document", ext)
}

func (a *TelegramAdapter) handleVoice(msg *tgbotapi.Message) string {
	voice := msg.Voice
	file, err := a.bot.GetFile(tgbotapi.FileConfig{FileID: voice.FileID})
	if err != nil {
		log.Printf("Failed to get voice file: %v", err)
		return ""
	}

	return a.downloadFile(file, "voice", ".ogg")
}

func (a *TelegramAdapter) downloadFile(file tgbotapi.File, prefix, ext string) string {
	workspaceDir := filepath.Join(a.workspaceDir, "files")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		log.Printf("Failed to create files directory: %v", err)
		return ""
	}

	filename := fmt.Sprintf("%s_%d%s", prefix, time.Now().Unix(), ext)
	filepath := filepath.Join(workspaceDir, filename)

	url := file.Link(a.token)
	resp, err := httpGet(url)
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return ""
	}
	defer resp.Body.Close()

	f, err := os.Create(filepath)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		return ""
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		log.Printf("Failed to save file: %v", err)
		return ""
	}

	return filepath
}

func httpGet(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return resp, nil
}

func (a *TelegramAdapter) Send(target string, message string) error {
	chatID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	_, err = a.bot.Send(msg)
	return err
}
