# Makefile para gmit

.PHONY: build clean test install help

# Variáveis
VERSION ?= $(shell node -p "require('./package.json').version")
BINARY_NAME := gmit
BIN_DIR := bin
LDFLAGS := -s -w -X github.com/PedroHercules/gommit/pkg/interfaces/cli.version=$(VERSION)

# Build padrão (sistema atual)
build:
	@echo "Compilando $(BINARY_NAME) para o sistema atual..."
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "✓ Build concluído: $(BIN_DIR)/$(BINARY_NAME)"

# Build para todas as plataformas
build-all:
	@echo "Compilando $(BINARY_NAME) para todas as plataformas..."
	@mkdir -p $(BIN_DIR)
	@echo "Compilando para linux/amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Compilando para linux/arm64..."
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 .
	@echo "Compilando para darwin/amd64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@echo "Compilando para darwin/arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Compilando para windows/amd64..."
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Compilando para windows/arm64..."
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-arm64.exe .
	@echo "✓ Build concluído para todas as plataformas!"
	@ls -lh $(BIN_DIR)/$(BINARY_NAME)-*

# Executar testes
test:
	@echo "Executando testes..."
	go test ./...

# Limpar arquivos de build
clean:
	@echo "Limpando arquivos de build..."
	rm -rf $(BIN_DIR)
	@echo "✓ Limpeza concluída"

# Instalar no sistema
install: build
	@echo "Instalando $(BINARY_NAME)..."
	cp $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✓ $(BINARY_NAME) instalado em /usr/local/bin/"

# Mostrar ajuda
help:
	@echo "Comandos disponíveis:"
	@echo "  build      - Compila para o sistema atual"
	@echo "  build-all  - Compila para todas as plataformas"
	@echo "  test       - Executa os testes"
	@echo "  clean      - Remove arquivos de build"
	@echo "  install    - Instala no sistema (requer sudo)"
	@echo "  help       - Mostra esta ajuda"

# Target padrão
default: build
