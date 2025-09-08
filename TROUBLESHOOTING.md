# Troubleshooting Gommit

## Erro EACCESS no Linux ao configurar chave API

### Problema
Ao executar `gmit config set-key <sua-chave>` no Linux, você pode receber um erro de EACCESS (permissão negada).

### Causa
Este erro geralmente ocorre quando:
1. O diretório home do usuário não tem permissões adequadas
2. O diretório `~/.gmit` não pode ser criado
3. Problemas com o keyring do sistema (GNOME Keyring, KWallet)
4. Permissões restritivas em diretórios existentes

### Melhorias Implementadas
O gmit agora inclui:
- **Correção automática de permissões**: O sistema verifica e corrige automaticamente as permissões dos diretórios
- **Fallback robusto**: Se não conseguir definir permissões ideais, tenta permissões mínimas (755)
- **Verificação de diretórios existentes**: Corrige permissões de diretórios já criados
- **Mensagens de erro mais claras**: Indica problemas específicos de permissão

### Soluções

#### 1. Verificar permissões do diretório home
```bash
ls -la ~/
# O diretório home deve ter permissões de escrita para o usuário
```

#### 2. O gmit agora cria automaticamente o diretório com permissões corretas
```bash
# O gmit tentará automaticamente:
# - Criar ~/.gmit com permissões 755
# - Corrigir permissões se o diretório já existir
# - Usar fallback se houver problemas de permissão
```

#### 3. Verificação manual (se necessário)
```bash
# Verificar permissões atuais
ls -la ~/.gmit

# Corrigir manualmente se necessário
chmod 755 ~/.gmit
chmod 755 ~/.gmit/keyring
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
gmit config set-key sua-chave-api
gmit config get-key
```

### Diagnóstico Avançado
Se ainda houver problemas, verifique:

```bash
# Verificar permissões detalhadas
ls -la ~/.gmit/
ls -la ~/.gmit/keyring/

# Verificar se o usuário tem acesso de escrita
touch ~/.gmit/test_write && rm ~/.gmit/test_write

# Verificar logs do sistema para erros de permissão
journalctl -u gmit --since "1 hour ago"
```

## Outros Problemas Comuns

### Comando não encontrado
Se `gmit` não for encontrado, verifique se está no PATH:
```bash
which gmit
echo $PATH
```

### Problemas de conectividade
Para testar a conectividade com a API:
```bash
gmit config validate
```