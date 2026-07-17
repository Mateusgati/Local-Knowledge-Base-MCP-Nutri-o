package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/ledongthuc/pdf"
	"github.com/philippgille/chromem-go"
)

const (
	BATCH_SIZE    = 20
	CHUNK_SIZE    = 800
	CHUNK_OVERLAP = 150
)

type Config struct {
	DocsDir        string
	GoogleAPIKey   string
	EmbeddingModel string
	Collection     string
	DBPath         string
}

func loadConfig() Config {
	return Config{
		DocsDir:        getEnvOrDefault("DOCS_DIR", "documentos_nutricao"),
		GoogleAPIKey:   getEnvOrDefault("GOOGLE_API_KEY", ""),
		EmbeddingModel: getEnvOrDefault("EMBEDDING_MODEL", "gemini-embedding-2"),
		Collection:     getEnvOrDefault("COLLECTION_NAME", "biblioteca_nutricao"),
		DBPath:         getEnvOrDefault("DB_PATH", "vector_db_nutricao"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func generateDocID(content, source string, index int) string {
	data := fmt.Sprintf("%s|%s|%d", source, content[:min(50, len(content))], index)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

func init() {
	exe, err := os.Executable()
	if err == nil {
		envPath := filepath.Join(filepath.Dir(exe), ".env")
		_ = godotenv.Load(envPath)
	}
	_ = godotenv.Load()
}

// Google AI Embedding
type embedRequest struct {
	Model   string       `json:"model"`
	Content embedContent `json:"content"`
}
type embedContent struct {
	Parts []embedPart `json:"parts"`
}
type embedPart struct {
	Text string `json:"text"`
}
type embedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewGoogleAIEmbeddingFunc(apiKey, model string) chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s", model, apiKey)
		reqBody := embedRequest{
			Model:   fmt.Sprintf("models/%s", model),
			Content: embedContent{Parts: []embedPart{{Text: text}}},
		}
		jsonData, _ := json.Marshal(reqBody)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var result embedResponse
		json.Unmarshal(body, &result)
		if result.Error != nil {
			return nil, fmt.Errorf("API: %s", result.Error.Message)
		}
		return result.Embedding.Values, nil
	}
}

// PDF extraction com recovery
func extractTextFromPDF(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	for i := 1; i <= r.NumPage(); i++ {
		func() {
			defer func() { recover() }() // Ignora panics de páginas corruptas
			page := r.Page(i)
			if page.V.IsNull() {
				return
			}
			text, err := page.GetPlainText(nil)
			if err == nil {
				buf.WriteString(text)
				buf.WriteString("\n\n")
			}
		}()
	}
	return buf.String(), nil
}

func splitText(text string, chunkSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return nil
	}
	var chunks []string
	runes := []rune(text)
	for i := 0; i < len(runes); i += chunkSize - overlap {
		end := min(i+chunkSize, len(runes))
		chunk := strings.TrimSpace(string(runes[i:end]))
		if len(chunk) > 50 {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  📚 Ingestor de Documentos                                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n📁 %s | 🧠 %s\n", cfg.DocsDir, cfg.EmbeddingModel)

	if cfg.GoogleAPIKey == "" {
		log.Fatalf("❌ GOOGLE_API_KEY não definida")
	}
	fmt.Println("🔑 API Key: ****" + cfg.GoogleAPIKey[len(cfg.GoogleAPIKey)-4:])

	// Resolve paths
	docsDir, dbPath := cfg.DocsDir, cfg.DBPath
	if exe, err := os.Executable(); err == nil {
		if !filepath.IsAbs(docsDir) {
			docsDir = filepath.Join(filepath.Dir(exe), cfg.DocsDir)
		}
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(filepath.Dir(exe), cfg.DBPath)
		}
	}

	pdfFiles, _ := filepath.Glob(filepath.Join(docsDir, "*.pdf"))
	if len(pdfFiles) == 0 {
		fmt.Println("⚠️ Nenhum PDF encontrado")
		return
	}
	fmt.Printf("📚 %d PDFs encontrados\n\n", len(pdfFiles))

	// DB
	db, _ := chromem.NewPersistentDB(dbPath, false)
	collection, _ := db.GetOrCreateCollection(cfg.Collection, nil,
		NewGoogleAIEmbeddingFunc(cfg.GoogleAPIKey, cfg.EmbeddingModel))
	fmt.Printf("💾 Banco: %d docs existentes\n\n", collection.Count())

	var total, files int
	var failed []string

	for _, pdfPath := range pdfFiles {
		name := filepath.Base(pdfPath)
		fmt.Printf("📖 %s... ", name)

		text, err := extractTextFromPDF(pdfPath)
		if err != nil || len(strings.TrimSpace(text)) < 100 {
			fmt.Println("❌ sem texto")
			failed = append(failed, name)
			continue
		}

		chunks := splitText(text, CHUNK_SIZE, CHUNK_OVERLAP)
		if len(chunks) == 0 {
			fmt.Println("❌ sem chunks")
			continue
		}

		fmt.Printf("%d chunks ", len(chunks))

		indexed := 0
		for i := 0; i < len(chunks); i += BATCH_SIZE {
			end := min(i+BATCH_SIZE, len(chunks))
			var docs []chromem.Document
			for j := i; j < end; j++ {
				docs = append(docs, chromem.Document{
					ID:       generateDocID(chunks[j], name, j),
					Content:  chunks[j],
					Metadata: map[string]string{"source": name},
				})
			}
			if err := collection.AddDocuments(ctx, docs, runtime.NumCPU()); err == nil {
				indexed += len(docs)
			}
		}
		fmt.Printf("✅ %d indexados\n", indexed)
		total += indexed
		files++
	}

	fmt.Printf("\n✅ Concluído: %d arquivos, %d chunks, %d total no banco\n", files, total, collection.Count())

	if len(failed) > 0 {
		fmt.Println("\n⚠️ PDFs sem texto (corrompidos ou escaneados):")
		for _, f := range failed {
			fmt.Printf("   • %s\n", f)
		}
		fmt.Println("\n💡 Esses PDFs provavelmente são imagens escaneadas.")
		fmt.Println("   Use OCR (ex: Adobe, ABBYY) para convertê-los em texto.")
	}
}
