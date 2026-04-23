# OpenClaw → PM-OS Port Package

Pacote de instruções para LLM externa portar SELETIVAMENTE 30+ items do OpenClaw (TypeScript) para PM-OS (Go).

## Contexto

OpenClaw (`https://github.com/openclaw/openclaw`) é personal AI assistant TS, 220k LOC, paradigma agent-first. PM-OS é engine Go orquestração task-first. **Não é port literal.** Extração de conceitos, snippets, schemas, algoritmos classificados em 4 tiers por método de adoção.

## Arquivos deste pacote

| Arquivo | Propósito |
|---|---|
| `README.md` | este arquivo |
| `PROMPT.md` | **instrução mestre** pra LLM externa (cola primeiro) |
| `PM-OS-CONTEXT.md` | briefing arquitetura alvo (anexa) |
| `ITEM-MAP.md` | lista dos 30+ items classificados por tier |
| `RULES.md` | convenções Go + restrições (anexa) |
| `CHECKLIST.md` | critérios aceitação por tier (anexa) |
| `SELF-AUDIT-PROMPT.md` | audit final antes de entregar |
| `EXTRACTION-PLAN.md` | análise-fonte com todos os conceitos + snippets de referência (anexa) |
| `NOTES/` | 19 notes originais de pesquisa (referência secundária) |

## Fluxo

1. Dis envia todos os docs desta pasta pra LLM externa
2. LLM externa clona openclaw local: `git clone --depth 1 https://github.com/openclaw/openclaw.git`
3. LLM externa lê `EXTRACTION-PLAN.md` + `NOTES/` como referência conceitual
4. LLM externa implementa cada item do `ITEM-MAP.md` seguindo método de tier
5. Entrega em `/tmp/openclaw-port-output/` espelhando estrutura pm-os/
6. Cada item inclui: código Go + tests + doc comments + PORT-REPORT linha
7. LLM externa roda `SELF-AUDIT-PROMPT.md` antes de entregar
8. Dis entrega pasta pro Akita (eu) no próximo turno
9. Akita revisa, integra, commit atômico por item

## Escopo

**Total:** ~30 items distribuídos em:

- **Tier A** (copy-paste direto): 8 items — schemas YAML, snippets bash, funções puras Go
- **Tier B** (port-and-adapt): 8 items — algoritmos TS→Go idiomático
- **Tier C** (capabilities JSON): 7 items — novos campos em recipe schema + validators
- **Tier D** (CLI + tools + tests + skills): 12+ items — comandos novos, test utilities, skill patterns

**Estimativa LOC Go final:** ~6000-8000 LOC (similar ao crewai port)
**Estimativa tempo LLM externa:** 6-12h sequencial, 3-5h paralelizado

## O que Akita faz depois

- Copia tier por tier pro PM-OS no branch `feat/openclaw-adoption`
- Resolve conflitos de imports com packages existentes
- Wire capabilities JSON no recipe validator
- CLI commands → registra em cobra root
- Test utilities → usa em testes de recipes existentes
- Keel + smoke test E2E

## Resultado esperado

PM-OS ganha ~30 features concretas em 1-2 semanas Akita integração. Valida ports CrewAI recém-integrados (skills/events/memory/mcp ganham wiring real via capabilities JSON + CLI).
