# 📦 B2 Manager — Go Edition

CLI em Go para gerenciar o Backblaze B2 via API S3-compatível.  
Lista buckets, navega em pastas e faz upload de imagens com URL de retorno.

---

## 🚀 Instalação

```bash
# Pré-requisito: Go 1.22+
go version

# Instale as dependências


# Compile
go build -o b2manager ./src

# Ou rode direto (sem compilar)
go run .
```

-você também pode usar com um comando unico:

```shell
b2manager upload --bucket "meu-bucket" --file "./foto.png" --to "teste/foto.png"
```

---

## 🔐 Configuração

Copie o `.env.example` para `.env` e preencha:

```env
B2_KEY_ID=seu_key_id_aqui
B2_APP_KEY=sua_application_key_aqui
B2_REGION=us-west-004
```

> **Onde encontrar a região?**  
> No painel do B2, clique no seu bucket → a URL do endpoint mostra a região.  
> Ex: `s3.us-west-004.backblazeb2.com` → região é `us-west-004`

> **Gere as chaves em:**  
> https://secure.backblaze.com/app_keys.htm  
> Permissões: `listBuckets`, `listFiles`, `readFiles`, `writeFiles`

---

## ▶️ Uso

```bash
./b2manager
```

### Menu

```
[1]  📦 Listar buckets
[2]  📂 Ver arquivos de um bucket
[3]  ⬆  Upload de imagem
[4]  🚪 Sair
```

### Fluxo de upload

1. Escolha o bucket
2. Informe a pasta de destino (ou enter para raiz)
3. Informe o caminho local da imagem
4. Confirme/altere o nome remoto
5. Progresso em tempo real → ao finalizar exibe a **URL pública** do arquivo

---

## 📋 Dependências

| Pacote | Uso |
|---|---|
| `aws-sdk-go-v2/s3` | API S3-compatível do B2 |
| `charmbracelet/lipgloss` | Estilos e cores no terminal |
| `schollz/progressbar` | Barra de progresso no upload |
| `joho/godotenv` | Leitura do `.env` |
| `golang.org/x/term` | Input de senha sem eco |

---

## ⚠️ Observações

- A URL gerada usa o formato `https://<endpoint>/file/<bucket>/<arquivo>`.  
  Para buckets **privados**, o arquivo não será acessível publicamente sem uma URL assinada.
- O upload é feito em streaming (sem carregar tudo na RAM).
- Para arquivos > 100 MB, o B2 recomenda multipart — o SDK faz isso automaticamente via `TransferManager` se necessário.