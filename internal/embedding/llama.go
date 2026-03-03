package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var (
	httpClient  *http.Client
	serverAddr  string
	serverMutex sync.Mutex
	serverOnce  sync.Once
	serverReady bool
)

type EmbeddingService struct {
	modelPath string
	serverURL string
}

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func NewEmbeddingService(modelPath string) (*EmbeddingService, error) {
	if modelPath == "" {
		homeDir, _ := os.UserHomeDir()
		modelPath = filepath.Join(homeDir, ".textclaw", "models", "nomic-embed-text-v1.5-Q8_0.gguf")
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("embedding model not found at %s. Please download from https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF", modelPath)
	}

	httpClient = &http.Client{Timeout: 60 * time.Second}

	return &EmbeddingService{
		modelPath: modelPath,
		serverURL: "http://127.0.0.1:8080",
	}, nil
}

func (s *EmbeddingService) StartServer() error {
	var err error
	serverOnce.Do(func() {
		go func() {
			cmd := startLlamaServer(s.modelPath)
			if cmd != nil {
				cmd.Wait()
			}
		}()
		time.Sleep(3 * time.Second)
		serverReady = true
	})
	return err
}

func startLlamaServer(modelPath string) *exec.Cmd {
	cmd := exec.Command("llama-server",
		"-m", modelPath,
		"--port", "8080",
		"--embeddings",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start llama-server: %v\n", err)
		return nil
	}
	return cmd
}

func (s *EmbeddingService) GenerateEmbedding(text string) ([]float32, error) {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if !serverReady {
		if err := s.StartServer(); err != nil {
			return nil, fmt.Errorf("failed to start embedding server: %w", err)
		}
	}

	prefixedText := "search_query: " + text

	reqBody := EmbeddingRequest{
		Model: "embedding-model",
		Input: prefixedText,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := httpClient.Post(s.serverURL+"/v1/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d", resp.StatusCode)
	}

	var result EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	embedding := make([]float32, len(result.Data[0].Embedding))
	for i, v := range result.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

func GetDefaultModelPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".textclaw", "models", "nomic-embed-text-v1.5-Q8_0.gguf")
}

func CheckModelExists() bool {
	modelPath := GetDefaultModelPath()
	_, err := os.Stat(modelPath)
	return err == nil
}
