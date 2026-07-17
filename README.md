<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version"/>
  <img src="https://img.shields.io/badge/MCP-Protocol-8B5CF6?style=for-the-badge&logo=anthropic&logoColor=white" alt="MCP Protocol"/>
  <img src="https://img.shields.io/badge/Google_AI-Embeddings-4285F4?style=for-the-badge&logo=google&logoColor=white" alt="Google AI"/>
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License"/>
</p>

<h1 align="center">📚 Local Knowledge Base MCP</h1>

<p align="center">
  <strong>Servidor MCP para RAG local com seus próprios PDFs</strong>
</p>

<p align="center">
  Transforme qualquer coleção de PDFs em uma base de conhecimento consultável por IA. Sem Docker, sem servidores externos — apenas um binário Go.
</p>

---

## 🎯 O que é isso?

Um servidor [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) que permite que assistentes de IA (Claude, Gemini, etc.) consultem **sua própria coleção de documentos** via busca semântica.

### Casos de Uso

- 📖 **Estudantes**: Indexe seus livros e apostilas, pergunte em linguagem natural
- 👨‍💻 **Desenvolvedores**: Crie uma base de conhecimento técnico personalizada
- 📋 **Profissionais**: Consulte manuais, documentação interna, normas técnicas
- 🔬 **Pesquisadores**: Busque em papers e literatura acadêmica

---

## ✨ Características

| Característica              | Descrição                                            |
| --------------------------- | ---------------------------------------------------- |
| **Zero Dependências**       | Sem Docker, sem servidores externos, sem Python      |
| **Banco Vetorial Embutido** | chromem-go persiste localmente no disco              |
| **Google AI Embeddings**    | Modelo `gemini-embedding-2`                          |
| **Protocolo MCP**           | Compatível com Claude Desktop, Antigravity, e outros |
| **100% Local**              | Seus PDFs nunca saem do seu computador\*             |

> \* Apenas o texto dos chunks é enviado à API do Google para gerar embeddings.

---

## 🏛️ Arquitetura

```
┌─────────────────────┐      JSON-RPC (stdio)      ┌──────────────────────────────────────────┐
│                     │◄──────────────────────────►│                                          │
│   Claude Desktop    │                            │        Local Knowledge Base MCP          │
│   Antigravity       │                            │  ┌──────────────┐    ┌────────────────┐  │
│   Outro MCP Client  │                            │  │  chromem-go  │    │   Google AI    │  │
│                     │                            │  │  (VectorDB)  │    │  (Embeddings)  │  │
└─────────────────────┘                            │  └──────────────┘    └────────────────┘  │
                                                   └──────────────────────────────────────────┘
```

---

## 🛠️ Tools Disponíveis

O servidor expõe duas ferramentas para clientes MCP:

### `consultar_base_conhecimento`

Busca semântica na base de conhecimento.

| Parâmetro  | Tipo   | Obrigatório | Descrição                         |
| ---------- | ------ | ----------- | --------------------------------- |
| `pergunta` | string | ✅          | Sua pergunta em linguagem natural |

**Retorna:** Os 5 fragmentos mais relevantes com score de similaridade, fonte e conteúdo.

### `verificar_status_vectordb`

Verifica o status do banco vetorial.

**Retorna:** Status do banco, contagem de documentos indexados e uso de memória.

---

## 🚀 Instalação

### Pré-requisitos

- **Go 1.22+**
- **Google AI API Key** — [Obter gratuitamente](https://aistudio.google.com/app/apikey)

### 1. Compilar

```bash
git clone https://github.com/Capman002/Local-Knowledge-Base-MCP---golang.git
cd Local-Knowledge-Base-MCP---golang

go build -o base-nutricao-rag.exe .
go build -o ingest.exe ./cmd/ingest
```

### 2. Configurar

Crie um arquivo `.env`:

```env
# Obrigatório
GOOGLE_API_KEY=sua-api-key-aqui

# Opcional - personalize para seu caso de uso
COLLECTION_NAME=biblioteca_nutricao
DB_PATH=vector_db_nutricao
DOCS_DIR=documentos_nutricao
EMBEDDING_MODEL=gemini-embedding-2
```

### 3. Adicionar seus PDFs

Coloque apenas os PDFs de nutrição na pasta configurada em `DOCS_DIR` (padrão: `documentos_nutricao/`):

```
documentos_nutricao/
├── livro_nutricao.pdf
├── diretriz_clinica.pdf
├── artigo_cientifico.pdf
└── ...
```

### 4. Indexar

```bash
./ingest.exe
```

### 5. Configurar Cliente MCP

Adicione ao Claude Desktop ou Antigravity:

```json
{
  "mcpServers": {
    "minha-base-conhecimento": {
      "command": "C:/caminho/para/base-nutricao-rag.exe",
      "args": [],
      "env": {}
    }
  }
}
```

---

## ⚙️ Personalização

### Variáveis de Ambiente

| Variável          | Padrão                   | Descrição                    |
| ----------------- | ------------------------ | ---------------------------- |
| `GOOGLE_API_KEY`  | _(obrigatório)_          | Chave da API Google AI       |
| `COLLECTION_NAME` | `biblioteca_nutricao`  | Nome da sua coleção          |
| `DB_PATH`         | `vector_db_nutricao`   | Onde salvar o banco vetorial |
| `DOCS_DIR`        | `documentos_nutricao`  | Pasta com seus PDFs          |
| `EMBEDDING_MODEL` | `gemini-embedding-2`   | Modelo de embeddings         |

### Exemplos de Configuração

**Base de Conhecimento Jurídico:**

```env
COLLECTION_NAME=legislacao_brasileira
DOCS_DIR=pdfs_juridicos
```

**Documentação Técnica:**

```env
COLLECTION_NAME=docs_empresa
DOCS_DIR=manuais
```

**Estudos Acadêmicos:**

```env
COLLECTION_NAME=papers_mestrado
DOCS_DIR=literatura
```

---

## 📝 Sobre a Configuração Padrão

Esta cópia foi configurada para nutrição. A pasta, a coleção, o banco vetorial e a descrição da ferramenta MCP foram isolados do caso de uso original de arquitetura de software.

---

## � Detalhes Técnicos

| Componente             | Valor          |
| ---------------------- | -------------- |
| Chunk Size             | 800 caracteres |
| Chunk Overlap          | 150 caracteres |
| Resultados por Query   | Top 5          |
| Dimensões do Embedding | Padrão do modelo |
| Comunicação            | MCP via stdio  |

---

## 📁 Estrutura do Projeto

```
├── main.go                 # Servidor MCP
├── cmd/ingest/main.go      # Ingestor de PDFs
├── documentos_nutricao/    # Seus PDFs (git-ignored)
├── vector_db_nutricao/     # Banco vetorial (git-ignored)
├── .env                    # Configurações (git-ignored)
└── .env.example            # Template
```

---

## 🐛 Solução de Problemas

| Problema                        | Solução                                   |
| ------------------------------- | ----------------------------------------- |
| "GOOGLE_API_KEY não definida"   | Crie o arquivo `.env` com sua API key     |
| "Nenhum PDF encontrado"         | Verifique a pasta `DOCS_DIR`              |
| "Contagem de documentos é zero" | Execute `ingest.exe`                      |
| "Erro ao extrair texto do PDF"  | O PDF pode ser imagem escaneada (use OCR) |

---

## 🤝 Contribuindo

Contribuições são bem-vindas! Veja [CONTRIBUTING.md](CONTRIBUTING.md).

---

## 📄 Licença

MIT — veja [LICENSE](LICENSE).

---

## 🔗 Dependências

- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) — Biblioteca MCP
- [philippgille/chromem-go](https://github.com/philippgille/chromem-go) — Banco vetorial embutido
- [ledongthuc/pdf](https://github.com/ledongthuc/pdf) — Parser de PDF
- [joho/godotenv](https://github.com/joho/godotenv) — Carregamento de .env
