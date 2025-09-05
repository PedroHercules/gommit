# Gommit - AI-Powered Git Commit Message Generator

Gommit é uma ferramenta de linha de comando que usa inteligência artificial para gerar mensagens de commit do Git automaticamente, seguindo as melhores práticas e o formato Conventional Commits. A ferramenta também oferece funcionalidades para geração de descrições de Pull Requests e validação de mensagens de commit.

## Funcionalidades Principais

- **Geração Automática de Commits**: IA analisa suas mudanças e gera mensagens seguindo Conventional Commits
- **Descrições de Pull Request**: Cria descrições detalhadas para seus PRs automaticamente
- **Validação de Mensagens**: Verifica se suas mensagens seguem as melhores práticas
- **Configuração Flexível**: Suporte a múltiplos modelos de IA e configurações personalizadas
- **Armazenamento Seguro**: Chaves de API armazenadas com segurança no keyring do sistema
- **Interface Intuitiva**: CLI simples e fácil de usar com ajuda contextual
- **Arquitetura Limpa**: Código bem estruturado seguindo Clean Architecture

## Arquitetura

Este projeto foi desenvolvido seguindo os princípios da **Clean Architecture**, proporcionando uma estrutura limpa, testável e fácil de manter. É ideal para aprender Go e padrões de arquitetura de software.

### 📁 Estrutura do Projeto

```
gommit/
├── main.go                      # Ponto de entrada da aplicação
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
│   │       ├── generate_commit_usecase.go
│   │       └── pull_request_usecase.go
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
│   │       ├── commit/          # Serviços de commit
│   │       ├── config/          # Serviços de configuração
│   │       └── pull-request/    # Serviços de pull request
│   └── interfaces/              # Camada de Interface
│       └── cli/                 # Interface de linha de comando
│           ├── commands.go      # Processamento de comandos
│           └── help.go          # Sistema de ajuda
├── go.mod                       # Dependências do Go
├── go.sum                       # Checksums das dependências
├── install.js                   # Script de instalação
├── package.json                 # Configuração Node.js
└── README.md                    # Este arquivo
```

### Princípios da Clean Architecture

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
./gommit commit --commit

# Apenas gerar a mensagem (sem fazer commit)
./gommit commit --dry-run

# Usar um modelo específico
./gommit commit --model claude-3-sonnet

# Gerar descrição de Pull Request
./gommit pr

# Validar uma mensagem de commit
./gommit validate "feat: add new feature"

# Ver ajuda
./gommit help
```

### Guia Completo de Uso

Para instruções detalhadas, exemplos práticos e solução de problemas, consulte o **[Guia de Uso Completo](USAGE.md)**.

O guia inclui:
- Todos os comandos e opções disponíveis
- Fluxos de trabalho recomendados
- Exemplos práticos de uso
- Configuração avançada
- Solução de problemas comuns
- Scripts de automação

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

### Pontos de Estudo

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

### Como Adicionar Novas Funcionalidades

#### Exemplo: Adicionar Suporte a Templates

1. **Domínio**: Criar entidade `Template` em `pkg/domain/entities/`
2. **Repositório**: Definir interface em `pkg/domain/repositories/`
3. **Caso de Uso**: Implementar lógica em `pkg/domain/usecases/`
4. **Infraestrutura**: Implementar persistência em `pkg/infrastructure/`
5. **Aplicação**: Criar serviço em `pkg/application/services/`
6. **Interface**: Adicionar comandos em `pkg/interfaces/cli/`

## Dependências

O projeto utiliza as seguintes dependências principais:

- **github.com/99designs/keyring**: Armazenamento seguro de credenciais
- **github.com/joho/godotenv**: Carregamento de variáveis de ambiente
- **Go 1.24+**: Versão mínima do Go

```bash
# Instalar dependências
go mod tidy

# Verificar dependências
go mod verify

# Atualizar dependências
go get -u ./...
```

## Testes (Futuro)

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

## Agradecimentos

- [OpenRouter](https://openrouter.ai/) pela API de IA
- [Conventional Commits](https://www.conventionalcommits.org/) pelo padrão de mensagens
- [99designs/keyring](https://github.com/99designs/keyring) pelo armazenamento seguro
- [joho/godotenv](https://github.com/joho/godotenv) pelo gerenciamento de variáveis
- Comunidade Go pelas melhores práticas e padrões de arquitetura
- Clean Architecture por Robert C. Martin pelos princípios de design

---

**Desenvolvido para aprender Go e Clean Architecture**