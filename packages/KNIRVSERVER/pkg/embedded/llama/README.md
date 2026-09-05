# llama

`llama` is one headless program for provisioning and serving a local llama.cpp
chat model. On first run it detects an existing `llama-server` and TinyLlama
model (including the legacy locations used by `llama-cli` and
`llama-installer`). If either is missing, it clones/builds llama.cpp and
downloads the default GGUF model without prompting.

```bash
go run ./cmd/llama
```

The API binds to `127.0.0.1:8080` by default:

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"TinyLlama-1.1B-Chat-v1.0-Q4_0","messages":[{"role":"user","content":"Hello"}]}'
```

Available endpoints are `GET /health`, `GET /v1/models`, `POST
/v1/chat/completions`, and `POST /v1/completions`. The two POST endpoints are
passed through to llama.cpp's OpenAI-compatible server.

Use `--server-path` and `--model-path` to supply existing assets, `--data-dir`
to choose the installation directory, and `--no-install` to make missing assets
an error. Persistent configuration follows XDG locations (`~/.config/llama` and
`~/.local/share/llama` by default).
