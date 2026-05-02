# B2 Manager — Go 

> CLI em Go para gerenciar o Backblaze B2 via API S3-compatível.  
> Lista buckets, navega pastas, faz upload de arquivos e retorna a URL pública — tudo no terminal.

---

## Sumário

- [Instalação](#instalação)
- [Configuração](#configuração)
- [Uso](#uso)
  - [Modo interativo](#modo-interativo)
  - [Modo CLI (não-interativo)](#modo-cli-não-interativo)
- [Segurança — credenciais locais](#segurança--credenciais-locais)
- [Dependências](#dependências)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Observações](#observações)

---

## Instalação

**Pré-requisito:** Go 1.22 ou superior.

```bash
# Instale as dependências
go mod tidy

# Compile
go build -o b2manager ./src

# Verifique
./b2manager --help
```

Ou rode diretamente sem compilar (útil em desenvolvimento):

```bash
go run ./src
```

---

## Configuração

O app aceita credenciais em três formas, verificadas nesta ordem:

### 1. Variáveis de ambiente / arquivo `.env`

Copie o `.env.example` para `.env` e preencha:

```env
B2_KEY_ID=seu_key_id_aqui
B2_APP_KEY=sua_application_key_aqui
B2_REGION=us-west-004

# Opcional: domínio personalizado (CDN ou Cloudflare)
CUSTOM_DOMAIN=https://cdn.seusite.com
```

> **Onde encontrar a região?**  
> No painel do B2, acesse seu bucket → o endpoint exibe a região.  
> Exemplo: `s3.us-west-004.backblazeb2.com` → região é `us-west-004`

> **Onde gerar as chaves?**  
> https://secure.backblaze.com/app_keys.htm  
> Permissões mínimas necessárias: `listBuckets`, `listFiles`, `readFiles`, `writeFiles`

### 2. Arquivo de credenciais criptografado (automático)

Se não houver `.env`, o app procura o arquivo `_b2-credentials.not-share-this-file` no mesmo diretório do binário. Esse arquivo é gerado automaticamente na primeira execução com credenciais manuais.

Veja a seção [Segurança](#segurança--credenciais-locais) para entender como ele funciona.

### 3. Input manual no primeiro uso

Sem `.env` e sem arquivo salvo, o app pergunta as credenciais uma vez e as salva automaticamente de forma criptografada para usos futuros.

---

## Uso

### Modo interativo

```bash
./b2manager
```

Abre o menu principal:

```
╭─────────────────────────────────╮
│  B2 Manager  v1.0               │
│                                 │
│  [1]  📦 Listar buckets         │
│  [2]  📂 Ver arquivos de bucket │
│  [3]  ⬆  Upload de arquivo      │
│  [4]  🚪 Sair                   │
╰─────────────────────────────────╯
```

**Fluxo de upload interativo:**

1. Selecione o bucket pelo número
2. Informe a pasta de destino (ou `Enter` para a raiz)
3. Informe o caminho local do arquivo
4. Confirme ou altere o nome remoto
5. Acompanhe o progresso em tempo real
6. Ao finalizar, receba a **URL pública** do arquivo (B2 ou domínio customizado)

**Listagem de arquivos:**

- Pastas são exibidas com seus arquivos internos (até 20 por pasta)
- Cada arquivo mostra tamanho, data e um emoji pelo tipo (🖼 imagem, 🎬 vídeo, 📄 PDF…)

---

### Modo CLI (não-interativo)

Ideal para scripts, pipelines de CI/CD ou integrações:

```bash
b2manager upload --bucket "meu-bucket" --file "./foto.png" --to "imagens/foto.png"
```

A saída é **somente a URL** do arquivo, facilitando o uso em scripts:

```bash
URL=$(./b2manager upload --bucket "assets" --file "./thumb.webp" --to "blog/thumb.webp")
echo "Imagem publicada em: $URL"
```

| Flag | Descrição | Obrigatório |
|---|---|---|
| `--bucket` | Nome do bucket de destino | ✓ |
| `--file` | Caminho local do arquivo | ✓ |
| `--to` | Caminho remoto de destino | ✓ |

Se `CUSTOM_DOMAIN` estiver configurado, a URL retornada usará o domínio customizado. Caso contrário, retorna a URL padrão do B2.

---

## Segurança — credenciais locais

Quando não há `.env`, o app salva as credenciais localmente de forma criptografada no arquivo `_b2-credentials.not-share-this-file`.

 Limitações honestas:

Esta proteção é eficaz contra **acesso offline** ao arquivo (cópia, exfiltração). Não protege contra:

- Malware rodando com as mesmas permissões do usuário
- Dump de memória do processo
- Atacante com acesso físico e root na máquina

### Regenerar credenciais

Se precisar recriar o arquivo (troca de chave, migração de máquina):

```bash
rm _b2-credentials.not-share-this-file
./b2manager  # vai pedir as credenciais novamente
```

---

## Dependências

| Pacote | Versão mínima | Uso |
|---|---|---|
| `aws/aws-sdk-go-v2/s3` | v2 | API S3-compatível do Backblaze B2 |
| `charmbracelet/lipgloss` | — | Estilos e cores no terminal |
| `schollz/progressbar/v3` | v3 | Barra de progresso no upload |
| `joho/godotenv` | — | Leitura do arquivo `.env` |
| `golang.org/x/term` | — | Input de senha sem eco no terminal |
| `golang.org/x/crypto/argon2` | — | Derivação de chave segura (KDF) |

---

## Observações

**URLs e privacidade**  
A URL gerada usa o formato `https://<endpoint>/file/<bucket>/<arquivo>`. Para buckets **privados**, o arquivo não é acessível publicamente sem uma URL assinada — o app não gera URLs assinadas.

**Upload em streaming**  
O arquivo é lido e enviado em stream, sem carregar o conteúdo inteiro na memória.

**Arquivos grandes**  
Para arquivos acima de ~100 MB, o B2 recomenda multipart upload. O AWS SDK v2 lida com isso automaticamente via `TransferManager` quando necessário.

**`go run` vs binário compilado**  
Ao usar `go run ./src`, o Go compila em `/tmp/go-build.../` — o app detecta isso e salva o arquivo de credenciais no diretório de trabalho atual em vez do diretório do binário temporário.