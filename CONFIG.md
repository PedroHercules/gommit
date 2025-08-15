# Configuração do Gommit

## Armazenamento Seguro de Chaves API

O Gommit utiliza o keyring do sistema operacional para armazenar chaves API de forma segura e multiplataforma

### Como funciona

- **Windows**: Utiliza o Windows Credential Manager
- **macOS**: Utiliza o Keychain
- **Linux**: Utiliza o Secret Service (GNOME Keyring, KWallet)

### Comandos de Configuração

#### Armazenar uma chave API

```bash
gommit config set-key <sua-chave-api>
```

Exemplo:

```bash
gommit config set-key sk-or-v1-abc123def456
```

#### Recuperar a chave API armazenada

```bash
gommit config get-key
```

### Prioridade de Chaves

O Gommit busca a chave API na seguinte ordem:

1. **Variável de ambiente** `OPENROUTER_API_KEY`
2. **Keyring do sistema** (armazenado com `gommit config set-key`)

Se nenhuma chave for encontrada, o comando falhará com uma mensagem explicativa.

### Segurança

- As chaves são armazenadas de forma criptografada no keyring do sistema
- Não são armazenadas em arquivos de texto simples
- Cada sistema operacional utiliza seu mecanismo nativo de segurança
- As chaves ficam protegidas pela autenticação do usuário do sistema

### Localização dos Arquivos de Configuração

O Gommit cria arquivos de configuração nos seguintes locais:

- **Windows**: `%APPDATA%\gommit\config.json`
- **macOS**: `~/Library/Application Support/gommit/config.json`
- **Linux**: `~/.config/gommit/config.json`

Esses arquivos contêm apenas configurações não-sensíveis. As chaves API ficam no keyring do sistema.
