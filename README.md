# RSS Reader

Este é um aplicativo em Go (Golang) para extrair informações de um feed RSS de um site específico e gerar um arquivo XML contendo os itens do feed. Ele também inclui funcionalidades para comparar o XML gerado com um XML final e adicionar apenas os itens diferentes ao XML final.

## Funcionalidades

- Baixa o conteúdo HTML de um site.
- Extrai os itens de um feed RSS do site.
- Gera um arquivo XML contendo os itens do feed.
- Compara o XML gerado com um XML final e adiciona apenas os itens diferentes ao XML final.
- Limita o XML final a no máximo 10 itens.

## Uso

Para usar o aplicativo, basta executar o binário com um argumento especificando o arquivo de saída XML:

```bash
./rss-reader <arquivo_de_saida.xml>
```

## Exemplo

```
./rss-reader /tmp/feed.xml
```

## Compilação

```
GOARCH=amd64 GOOS=linux go build -o rssinformes -tags netgo -ldflags '-extldflags "-static"' main.go   ```

