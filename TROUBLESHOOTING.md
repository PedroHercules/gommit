# Troubleshooting Gommit

## Erro EACCESS no Linux ao configurar chave API

### Problema
Ao executar `gommit config set-key <sua-chave>` no Linux, você pode receber um erro de EACCESS (permissão negada).

### Causa
Este erro geralmente ocorre quando:
1. O diretório home do usuário não tem permissões adequadas
2. O diretório `~/.gmit` não pode ser criado
3. Problemas com o keyring do sistema (GNOME Keyring, KWallet)

### Soluções

#### 1. Verificar permissões do diretório home
```bash
ls -la ~/
# O diretório home deve ter permissões de escrita para o usuário
```

#### 2. Criar manualmente o diretório de configuração
```bash
mkdir -p ~/.gmit
chmod 755 ~/.gmit
```

#### 3. Verificar se o keyring do sistema está funcionando
```bash
# Para sistemas com GNOME Keyring
gnome-keyring-daemon --version

# Para sistemas com KWallet
kwalletd5 --version
```

#### 4. Instalar dependências do keyring (se necessário)
```bash
# Ubuntu/Debian
sudo apt-get install gnome-keyring libsecret-1-0

# CentOS/RHEL/Fedora
sudo yum install gnome-keyring libsecret
# ou
sudo dnf install gnome-keyring libsecret
```

#### 5. Usar variável de ambiente como alternativa
Se o problema persistir, você pode usar a variável de ambiente:
```bash
export OPENROUTER_API_KEY="sua-chave-api"
```

Adicione esta linha ao seu `~/.bashrc` ou `~/.zshrc` para torná-la permanente.

### Verificação
Após aplicar as soluções, teste novamente:
```bash
gommit config set-key sua-chave-api
gommit config get-key
```

## Outros Problemas Comuns

### Comando não encontrado
Se `gommit` não for encontrado, verifique se está no PATH:
```bash
which gommit
echo $PATH
```

### Problemas de conectividade
Para testar a conectividade com a API:
```bash
gommit config validate
```