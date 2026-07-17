package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/philippgille/chromem-go"
)

// ============================================================================
// SENIOR ARCHITECT RAG - Zero-Dependency MCP Server
// ============================================================================

const (
	SERVER_NAME    = "Base-Nutricao-RAG"
	SERVER_VERSION = "3.0.0"
)

type RAGServer struct {
	db         *chromem.DB
	collection *chromem.Collection
}

type Config struct {
	GoogleAPIKey   string
	EmbeddingModel string
	Collection     string
	DBPath         string
}

func loadConfig() Config {
	return Config{
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

func init() {
	exe, err := os.Executable()
	if err == nil {
		envPath := filepath.Join(filepath.Dir(exe), ".env")
		_ = godotenv.Load(envPath)
	}
	_ = godotenv.Load()
}

// ============================================================================
// GOOGLE AI EMBEDDING FUNCTION
// ============================================================================

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
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func NewGoogleAIEmbeddingFunc(apiKey, model string) chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s", model, apiKey)

		reqBody := embedRequest{
			Model: fmt.Sprintf("models/%s", model),
			Content: embedContent{
				Parts: []embedPart{{Text: text}},
			},
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("erro ao serializar: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("erro ao criar request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("erro na requisição: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler resposta: %w", err)
		}

		var result embedResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("erro ao parsear: %w", err)
		}

		if result.Error != nil {
			return nil, fmt.Errorf("API error: %s", result.Error.Message)
		}

		if len(result.Embedding.Values) == 0 {
			return nil, fmt.Errorf("embedding vazio")
		}

		return result.Embedding.Values, nil
	}
}

func main() {
	cfg := loadConfig()

	fmt.Fprintln(os.Stderr, "╔════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stderr, "║  🥗 Base de Nutrição RAG - MCP Server                      ║")
	fmt.Fprintln(os.Stderr, "╚════════════════════════════════════════════════════════════╝")
	fmt.Fprintf(os.Stderr, "🧠 Embeddings: %s\n", cfg.EmbeddingModel)
	fmt.Fprintf(os.Stderr, "📚 Collection: %s\n", cfg.Collection)
	fmt.Fprintf(os.Stderr, "💾 DB Path: %s\n\n", cfg.DBPath)

	if cfg.GoogleAPIKey == "" {
		log.Fatalf("❌ FATAL: GOOGLE_API_KEY não definida.\n")
	}
	fmt.Fprintln(os.Stderr, "🔑 Google API Key: ****"+cfg.GoogleAPIKey[len(cfg.GoogleAPIKey)-4:])

	dbPath := cfg.DBPath
	if !filepath.IsAbs(dbPath) {
		exe, err := os.Executable()
		if err == nil {
			dbPath = filepath.Join(filepath.Dir(exe), cfg.DBPath)
		}
	}

	fmt.Fprintln(os.Stderr, "💾 Carregando banco vetorial...")

	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		log.Fatalf("❌ FATAL: Erro ao abrir banco: %v", err)
	}
	fmt.Fprintln(os.Stderr, "✅ Banco vetorial: ONLINE")

	embeddingFunc := NewGoogleAIEmbeddingFunc(cfg.GoogleAPIKey, cfg.EmbeddingModel)

	collection, err := db.GetOrCreateCollection(cfg.Collection, nil, embeddingFunc)
	if err != nil {
		log.Fatalf("❌ FATAL: Erro ao criar collection: %v", err)
	}
	fmt.Fprintf(os.Stderr, "📚 Collection '%s': %d documentos\n", cfg.Collection, collection.Count())

	ragServer := &RAGServer{
		db:         db,
		collection: collection,
	}

	s := server.NewMCPServer(
		SERVER_NAME,
		SERVER_VERSION,
		server.WithToolCapabilities(true),
	)

	registerTools(s, ragServer)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "🟢 Servidor pronto. Aguardando comandos via Stdio...")

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Erro fatal: %v", err)
	}
}

func registerTools(s *server.MCPServer, rag *RAGServer) {
	toolConsulta := mcp.NewTool(
		"consultar_base_conhecimento",
		mcp.WithDescription(`Consulta uma base de conhecimento especializada em nutrição.
Use para localizar informações nos documentos de nutrição fornecidos pelo usuário.
As respostas devem se apoiar nos fragmentos recuperados e informar a fonte.`),
		mcp.WithString("pergunta",
			mcp.Required(),
			mcp.Description("A pergunta sobre nutrição que deseja pesquisar nos documentos"),
		),
	)
	s.AddTool(toolConsulta, rag.HandleConsulta)

	toolStatus := mcp.NewTool(
		"verificar_status_vectordb",
		mcp.WithDescription("Verifica o status do banco vetorial"),
	)
	s.AddTool(toolStatus, rag.HandleStatus)
}

func (rs *RAGServer) HandleConsulta(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("pergunta")
	if err != nil || query == "" {
		return mcp.NewToolResultError("Parâmetro 'pergunta' é obrigatório"), nil
	}

	fmt.Fprintf(os.Stderr, "[RAG] Query: %s\n", query)

	results, err := rs.collection.Query(ctx, query, 5, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("❌ Erro: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(
			"⚠️ Nenhum resultado para: '" + query + "'\n" +
				"Execute ingest.exe para indexar os PDFs primeiro.",
		), nil
	}

	resultado := fmt.Sprintf("📚 Resultados para: '%s'\n\n", query)

	for i, doc := range results {
		source := doc.Metadata["source"]
		if source == "" {
			source = "Desconhecido"
		}
		resultado += fmt.Sprintf("━━━ FRAGMENTO %d (%.2f) ━━━\n", i+1, doc.Similarity)
		resultado += fmt.Sprintf("📄 Fonte: %s\n", source)
		resultado += fmt.Sprintf("📝 Conteúdo:\n%s\n\n", doc.Content)
	}

	fmt.Fprintf(os.Stderr, "[RAG] Retornando %d resultados\n", len(results))
	return mcp.NewToolResultText(resultado), nil
}

func (rs *RAGServer) HandleStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := "🔍 Status do Sistema\n\n"
	status += "✅ Banco Vetorial: ONLINE\n"
	status += fmt.Sprintf("📚 Documentos: %d\n", rs.collection.Count())
	status += fmt.Sprintf("💾 RAM: %.1f MB\n", float64(m.Alloc)/1024/1024)

	return mcp.NewToolResultText(status), nil
}
