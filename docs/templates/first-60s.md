# Primeiro 60 segundos se algo quebrou

## Diagnóstico Rápido

Execute esses 5 comandos nesta ordem:

```bash
# 1. Serviços estão rodando?
systemctl status pmos-api pmos-engine

# 2. Diagnóstico automatizado
pm-cli doctor

# 3. API responde?
curl http://localhost:8080/health

# 4. MCP servers conectados?
pm-cli mcp list

# 5. Logs recentes
journalctl -u pmos-api -n 50
```

## Cenários Comuns

### API não responde (health check falha)
1. `systemctl restart pmos-api`
2. Verifique `SUPABASE_URL` e `ANTHROPIC_API_KEY` em `/etc/pmos/env`
3. Confira porta em uso: `ss -tlnp | grep 8080`

### Recipe trava (status=running > 30min)
1. `pm-cli doctor --fix` — reseta runs presas automaticamente
2. Verifique se o executor está responsivo: `pm-cli bench executor`
3. Consulte logs: `journalctl -u pmos-engine --since "30 min ago"`

### MCP server offline
1. `pm-cli mcp list` — verifique status
2. `pm-cli mcp set <name> <config.json>` — reconfigure
3. Teste conexão direta: `curl <mcp-url>/health`

### Memória cheia / Episodes acumulando
1. `pm-cli memory status` — verifique contagem por scope
2. `pm-cli memory evict --max-age-days=30` — force eviction
3. Verifique `retention_policy` na recipe config
