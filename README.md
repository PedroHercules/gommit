# 🤖 Gommit - AI-Powered Git Commit Message Generator

Gommit é uma ferramenta de linha de comando que usa inteligência artificial para gerar mensagens de commit do Git automaticamente, seguindo as melhores práticas e o formato Conventional Commits.

## 🏗️ Arquitetura

Este projeto foi desenvolvido seguindo os princípios da **Clean Architecture**, proporcionando uma estrutura limpa, testável e fácil de manter. É ideal para aprender Go e padrões de arquitetura de software.

### 📁 Estrutura do Projeto

```
gommit/
├── cmd/
│   └── gommit/
│       └── main.go              # Ponto de entrada da aplicação
├── pkg/
│   ├── domain/                  # Camada de Domínio (regras de negócio)
│   │   ├── entities/            # Entidades do domínio
│   │   │   ├── commit.go        # Entidade Commit
│   │   │   ├── config.go        # Entidade Config
│   │   │   └── git_diff.go      # Entidade GitDiff
│   │   ├── repositories/        # Interfaces dos repositórios
│   │   │   ├── config_repository.go
│   │   │   ├── git_repository.go
│   │   │   └── llm_repository.go
│   │   └── usecases/            # Casos de uso (lógica de negócio)
│   │       ├── commit_usecase.go
│   │       ├── config_usecase.go
│   │       └── generate_commit_usecase.go
│   ├── infrastructure/          # Camada de Infraestrutura
│   │   ├── config/              # Implementação de configuração
│   │   │   ├── file_config_repository.go
│   │   │   └── keyring_service.go
│   │   ├── git/                 # Implementação Git
│   │   │   └── git_repository.go
│   │   └── llm/                 # Implementação LLM
│   │       └── openrouter_repository.go
│   ├── application/             # Camada de Aplicação
│   │   └── services/            # Serviços de aplicação
│   │       ├── commit_service.go
│   │       └── config_service.go
│   └── interfaces/              # Camada de Interface
│       └── cli/                 # Interface de linha de comando
│           ├── commands.go
│           └── help.go
├── go.mod                       # Dependências do Go
└── README.md                    # Este arquivo
```

### 🎯 Princípios da Clean Architecture

#### 1. **Camada de Domínio** (`pkg/domain/`)
- **Entidades**: Objetos de negócio fundamentais (`Commit`, `Config`, `GitDiff`)
- **Repositórios**: Interfaces que definem como acessar dados
- **Casos de Uso**: Lógica de negócio pura, independente de frameworks

#### 2. **Camada de Infraestrutura** (`pkg/infrastructure/`)
- **Implementações**: Código que interage com sistemas externos
- **Git**: Comandos Git via CLI
- **LLM**: Integração com APIs de IA (OpenRouter)
- **Config**: Armazenamento de configurações e credenciais

#### 3. **Camada de Aplicação** (`pkg/application/`)
- **Serviços**: Orquestram casos de uso e coordenam operações
- **Fluxo**: Conectam a interface do usuário com a lógica de negócio

#### 4. **Camada de Interface** (`pkg/interfaces/`)
- **CLI**: Interface de linha de comando
- **Entrada**: Processa comandos do usuário
- **Saída**: Apresenta resultados formatados

## 🚀 Instalação e Uso

### Pré-requisitos
- Go 1.21 ou superior
- Git instalado e configurado
- Chave de API do OpenRouter

### Instalação

```bash
# Clone o repositório
git clone https://github.com/PedroHercules/gommit.git
cd gommit

# Instale as dependências
go mod tidy

# Compile o projeto
go build -o gommit main.go
```

### Configuração

```bash
# Configure sua chave de API do OpenRouter
./gommit config set-key sk-or-v1-sua-chave-aqui

# (Opcional) Configure um modelo padrão
./gommit config set-model claude-3-sonnet

# Verifique a configuração
./gommit config summary
```

### Uso Básico

```bash
# Gerar e fazer commit automaticamente
./gommit

# Apenas gerar a mensagem (sem fazer commit)
./gommit --no-commit

# Usar um modelo específico
./gommit --model claude-3-haiku

# Verificar status do repositório
./gommit status

# Validar uma mensagem de commit
./gommit validate "feat: add new feature"

# Ver ajuda
./gommit help
```

## 📚 Guia de Aprendizado

### 🎓 Conceitos de Go Demonstrados

1. **Interfaces e Polimorfismo**
   - Veja `pkg/domain/repositories/` para interfaces bem definidas
   - Implementações em `pkg/infrastructure/`

2. **Estruturas e Métodos**
   - Entidades em `pkg/domain/entities/`
   - Métodos de validação e transformação

3. **Tratamento de Erros**
   - Padrão Go de retorno de erro
   - Wrapping de erros com contexto

4. **Organização de Pacotes**
   - Separação clara de responsabilidades
   - Imports bem organizados

5. **Injeção de Dependências**
   - Manual DI em `main.go`
   - Inversão de controle

### 🔍 Pontos de Estudo

#### Iniciante
1. **Entidades** (`pkg/domain/entities/`)
   - Como definir estruturas
   - Métodos de validação
   - Construtores

2. **Interfaces** (`pkg/domain/repositories/`)
   - Definição de contratos
   - Desacoplamento

#### Intermediário
3. **Casos de Uso** (`pkg/domain/usecases/`)
   - Lógica de negócio
   - Orquestração de operações

4. **Implementações** (`pkg/infrastructure/`)
   - Integração com sistemas externos
   - Tratamento de erros

#### Avançado
5. **Serviços** (`pkg/application/services/`)
   - Coordenação de casos de uso
   - Transformação de dados

6. **CLI** (`pkg/interfaces/cli/`)
   - Processamento de argumentos
   - Interface do usuário

### 🛠️ Como Adicionar Novas Funcionalidades

#### Exemplo: Adicionar Suporte a Templates

1. **Domínio**: Criar entidade `Template` em `pkg/domain/entities/`
2. **Repositório**: Definir interface em `pkg/domain/repositories/`
3. **Caso de Uso**: Implementar lógica em `pkg/domain/usecases/`
4. **Infraestrutura**: Implementar persistência em `pkg/infrastructure/`
5. **Aplicação**: Criar serviço em `pkg/application/services/`
6. **Interface**: Adicionar comandos em `pkg/interfaces/cli/`

## 🧪 Testes (Futuro)

A arquitetura facilita a criação de testes:

```bash
# Testes unitários (domínio)
go test ./pkg/domain/...

# Testes de integração (infraestrutura)
go test ./pkg/infrastructure/...

# Testes end-to-end (CLI)
go test ./pkg/interfaces/...
```

## 🤝 Contribuindo

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-funcionalidade`)
3. Commit suas mudanças (`git commit -am 'feat: adiciona nova funcionalidade'`)
4. Push para a branch (`git push origin feature/nova-funcionalidade`)
5. Abra um Pull Request

## 📄 Licença

MIT License - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🙏 Agradecimentos

- [OpenRouter](https://openrouter.ai/) pela API de IA
- [Conventional Commits](https://www.conventionalcommits.org/) pelo padrão
- Comunidade Go pelas melhores práticas

---

**Desenvolvido com ❤️ para aprender Go e Clean Architecture**