# Uso como base de conhecimento de nutrição

Esta cópia usa uma pasta, uma coleção e um banco vetorial separados dos documentos de programação que vieram no repositório.

O modelo de embeddings foi atualizado para `gemini-embedding-2`, pois o `text-embedding-004` usado no projeto original foi desativado.

## Preparação

1. Copie `.env.example` para `.env`.
2. Substitua `sua-api-key-aqui` pela chave do Google AI Studio.
3. Coloque apenas os PDFs desejados em `documentos_nutricao/`.
4. Compile os dois executáveis:

```powershell
go build -o base-nutricao-rag.exe .
go build -o ingest.exe ./cmd/ingest
```

5. Crie o índice:

```powershell
.\ingest.exe
```

6. Configure o cliente MCP para executar `base-nutricao-rag.exe`.

## Isolamento dos conteúdos

- `DOCS_DIR=documentos_nutricao` impede que `biblioteca_docs/` seja lida pelo ingestor.
- `DB_PATH=vector_db_nutricao` impede o reaproveitamento de um índice antigo.
- `COLLECTION_NAME=biblioteca_nutricao` mantém os fragmentos em uma coleção específica.

Se trocar completamente o acervo no futuro, apague o diretório local `vector_db_nutricao/` e execute o ingestor novamente. Remover apenas o PDF da pasta não elimina automaticamente os fragmentos já indexados.

PDFs escaneados como imagem precisam passar por OCR antes da ingestão.
