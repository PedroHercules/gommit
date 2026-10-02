#!/usr/bin/env pwsh

# Script de build para gmit
# Compila para diferentes sistemas operacionais e arquiteturas

Write-Host "Iniciando build do gmit..." -ForegroundColor Green

# Criar diretório bin se não existir
if (!(Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin"
}

# Usar a versão declarada no package npm.
$VERSION = node -p "require('./package.json').version"

# Builds para diferentes plataformas
$builds = @(
    @{OS="linux"; ARCH="amd64"; EXT=""},
    @{OS="linux"; ARCH="arm64"; EXT=""},
    @{OS="darwin"; ARCH="amd64"; EXT=""},
    @{OS="darwin"; ARCH="arm64"; EXT=""},
	@{OS="windows"; ARCH="amd64"; EXT=".exe"},
	@{OS="windows"; ARCH="arm64"; EXT=".exe"}
)

foreach ($build in $builds) {
    $outputName = "gmit-$($build.OS)-$($build.ARCH)$($build.EXT)"
    $outputPath = "bin\$outputName"
    
    Write-Host "Compilando para $($build.OS)/$($build.ARCH)..." -ForegroundColor Yellow
    
    $env:GOOS = $build.OS
    $env:GOARCH = $build.ARCH
    
    go build -trimpath -ldflags "-s -w -X github.com/PedroHercules/gommit/pkg/interfaces/cli.version=$VERSION" -o $outputPath .
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ $outputName compilado com sucesso" -ForegroundColor Green
    } else {
        Write-Host "✗ Erro ao compilar $outputName" -ForegroundColor Red
        exit 1
    }
}

# Limpar variáveis de ambiente
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

Write-Host "\nBuild concluído! Binários disponíveis em:" -ForegroundColor Green
Get-ChildItem "bin\gmit-*" | ForEach-Object {
    $size = [math]::Round($_.Length / 1MB, 2)
    Write-Host "  $($_.Name) ($size MB)" -ForegroundColor Cyan
}

Write-Host "\nPara testar localmente, execute: .\bin\gmit-windows-amd64.exe" -ForegroundColor Yellow
