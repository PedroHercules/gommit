#!/bin/bash

# Script de build para gmit
# Compila para diferentes sistemas operacionais e arquiteturas

set -e

echo "Iniciando build do gmit..."

# Criar diretório bin se não existir
mkdir -p bin

# Usar a versão declarada no package npm.
VERSION="$(node -p "require('./package.json').version")"

# Builds para diferentes plataformas
builds=(
    "linux:amd64:"
    "linux:arm64:"
    "darwin:amd64:"
    "darwin:arm64:"
    "windows:amd64:.exe"
    "windows:arm64:.exe"
)

for build in "${builds[@]}"; do
    IFS=':' read -r os arch ext <<< "$build"
    output_name="gmit-${os}-${arch}${ext}"
    output_path="bin/${output_name}"
    
    echo "Compilando para ${os}/${arch}..."
    
    GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-s -w -X github.com/PedroHercules/gommit/pkg/interfaces/cli.version=$VERSION" -o "$output_path" .
    
    if [ $? -eq 0 ]; then
        # Set executable permissions for Unix binaries
        if [ "$os" != "windows" ]; then
            chmod +x "$output_path"
        fi
        echo "✓ $output_name compilado com sucesso"
    else
        echo "✗ Erro ao compilar $output_name"
        exit 1
    fi
done

echo ""
echo "Build concluído! Binários disponíveis em:"
for file in bin/gmit-*; do
    if [ -f "$file" ]; then
        size=$(du -h "$file" | cut -f1)
        echo "  $(basename "$file") ($size)"
    fi
done

echo ""
echo "Para testar localmente:"
echo "  Linux: ./bin/gmit-linux-amd64"
echo "  macOS: ./bin/gmit-darwin-amd64"
echo "  Windows: ./bin/gmit-windows-amd64.exe"
