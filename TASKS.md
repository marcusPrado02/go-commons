# TASKS.md — go-commons improvement backlog

Organized by area. Each task is self-contained and actionable.

---

## Documentação

- [ ] **DOC-01** Reescrever o `README.md` com: visão geral, feature matrix (kernel / app / adapters), quick start, e link para cada pacote
- [ ] **DOC-02** Criar `docs/architecture.md` explicando a Arquitetura Hexagonal adotada, com diagrama de camadas (kernel → ports → app → adapters)
- [ ] **DOC-03** Criar `docs/error-handling.md` guia de uso de `Problem`, `ErrorCode`, `Result[T]` e como propagar erros entre camadas
- [ ] **DOC-04** Criar `docs/adapter-selection.md` com tabela comparativa de quando usar SendGrid vs SES vs SMTP, S3 vs GCS, etc.
- [ ] **DOC-05** Criar `docs/contributing.md` com convenções de commit, padrão de teste, como adicionar um novo adapter e como rodar o projeto localmente
- [ ] **DOC-06** Adicionar doc comment em todos os tipos e funções exportados dos pacotes `ports/cache`, `ports/queue`, `ports/push`, `ports/compression`, `ports/excel` — atualmente sem comentários
- [ ] **DOC-07** Criar `docs/outbox.md` explicando o Transactional Outbox Pattern, garantias de entrega, e como configurar `OutboxPublisher` em produção
- [ ] **DOC-08** Criar `docs/resilience.md` documentando jitter backoff, estados do circuit breaker (`gobreaker`) e exemplos de `ResiliencePolicySet`
- [ ] **DOC-09** Criar `examples/` com ao menos 3 aplicações mínimas demonstrando: (1) aggregate + result + outbox, (2) scheduler + resilience, (3) inmemory repository + contract suite
- [ ] **DOC-10** Adicionar `CHANGELOG.md` com a convenção Keep a Changelog, registrando a v0.1.0 inicial

---

## Testes — portas sem cobertura

- [ ] **TEST-01** Adicionar testes para `ports/cache`: verificar assinatura da interface, zero value de `CachePort`, TTL=0 semântica documentada
- [ ] **TEST-02** Adicionar testes para `ports/queue`: `Message` fields, `Handler` tipo, `QueuePort` compile-time check
- [ ] **TEST-03** Adicionar testes para `ports/push`: `PushNotification` fields, compile-time interface check
- [ ] **TEST-04** Adicionar testes para `ports/compression`: `Format` constants (gzip, zstd, snappy), compile-time interface check
- [ ] **TEST-05** Adicionar testes para `ports/excel`: `Sheet`, `ExcelRequest` fields, zero values
- [ ] **TEST-06** Adicionar testes de contrato (`testkit/contracts`) para `EmailPort` — suite reutilizável análoga ao `RepositoryContract`
- [ ] **TEST-07** Adicionar testes de contrato para `FileStorePort` — upload, download, delete, exists, list, presign
- [ ] **TEST-08** Adicionar testes de contrato para `CachePort` — get/set/delete/exists com TTL
- [ ] **TEST-09** Adicionar testes de contrato para `SMSPort` — send e ping
- [ ] **TEST-10** Adicionar testes de contrato para `QueuePort` — publish e subscribe

---

## Testes — adapters sem cobertura

- [ ] **TEST-11** Adicionar testes para `adapters/email/smtp` usando `net/smtp` test server ou mock: Send com validação, multipart MIME, erro de auth
- [ ] **TEST-12** Adicionar testes para `adapters/email/ses` usando `httptest` interceptando o SDK: Send, SendWithTemplate retorna erro explicativo, Ping
- [ ] **TEST-13** Adicionar testes para `adapters/sms/twilio` com mock HTTP server: Send, Ping retorna erro quando SID é vazio
- [ ] **TEST-14** Adicionar testes para `adapters/payment/stripe` com mock HTTP server: CreatePaymentIntent valida amount > 0, Refund com chargeID vazio
- [ ] **TEST-15** Adicionar testes para `adapters/search/elasticsearch` com `httptest`: Index, Search retorna hits, Delete ignora 404, Ping
- [ ] **TEST-16** Adicionar testes para `adapters/search/opensearch` com `httptest`: mesma cobertura do Elasticsearch
- [ ] **TEST-17** Adicionar testes para `adapters/tracing/otel`: `StartSpan` retorna span não-nil, `SetAttribute` para cada tipo suportado, `RecordError` propaga status
- [ ] **TEST-18** Adicionar testes de integração para `adapters/files/s3` com `localstack` via Docker Compose (skipped se Docker não disponível)
- [ ] **TEST-19** Adicionar benchmark (`BenchmarkInMemoryRepository_FindAll`) para medir impacto de Specification + paginação com N=1000 e N=100000 entidades
- [ ] **TEST-20** Adicionar testes de race condition para `app/outbox`: dois goroutines chamando `Start` simultaneamente devem ser idempotentes

---

## Qualidade de código — correções

- [ ] **FIX-01** `app/scheduler/scheduler.go:54` — `_ = job.Run(context.Background())` descarta o erro silenciosamente; logar via `Logger` opcional ou expor via callback configurável
- [ ] **FIX-02** `app/scheduler/scheduler.go:49-52` — panic recovery descarta `r` sem nenhum log; adicionar campo `Logger` opcional em `defaultScheduler` e logar o panic com stack trace
- [ ] **FIX-03** `adapters/email/smtp/client.go` — `buildMessage` trata somente `email.To[0]` no header `To:`; corrigir para incluir todos os destinatários separados por vírgula
- [ ] **FIX-04** `adapters/email/smtp/client.go:124` — erros de `part.Write()` são ignorados; propagar ou ao menos logar
- [ ] **FIX-05** `adapters/email/smtp/client.go:109` — `conn.Close()` após erro no `smtp.NewClient` ignora o erro de fechamento; usar `defer` corretamente
- [ ] **FIX-06** `app/outbox/outbox.go` — validar que `PublishFunc` não é nil em `NewPublisher`; atualmente causa panic em runtime
- [ ] **FIX-07** `app/resilience/executor.go` — validar que `RetryMaxDelay >= RetryDelay` e que `RetryAttempts >= 0` em `NewExecutor`; retornar erro de configuração inválida
- [ ] **FIX-08** `ports/persistence/repository.go` — adicionar validação de `PageRequest.Size > 0` em `Spec()` ou documentar comportamento com `Size=0`
- [ ] **FIX-09** `adapters/files/s3/client.go` — `FileObject.Content` retornado por `Download` deve ser documentado como responsabilidade do caller fechar (`io.ReadCloser`)
- [ ] **FIX-10** `adapters/email/ses/client.go` — `SendWithTemplate` retorna erro genérico; melhorar mensagem indicando o caminho correto (sesv2) e retornar `kerrors.ErrTechnical` com detalhe

---

## Qualidade de código — melhorias

- [ ] **IMPROVE-01** Adicionar campo `Logger` opcional (ports/observability.Logger) ao `OutboxPublisher` para logar erros de `FetchPending`, `publish` e `MarkProcessed`
- [ ] **IMPROVE-02** Adicionar campo `Logger` opcional ao `ResilienceExecutor` para logar cada tentativa de retry, delay aplicado e abertura/fechamento do circuit breaker
- [ ] **IMPROVE-03** Adicionar campo `Logger` opcional ao `Scheduler` para logar início/fim de cada job e erros retornados por `Job.Run`
- [ ] **IMPROVE-04** `app/observability/health.go` — adicionar timeout configurável por `HealthCheck`; atualmente um check lento bloqueia indefinidamente
- [ ] **IMPROVE-05** `adapters/persistence/inmemory` — adicionar método `Clear()` para limpar o repositório entre testes sem recriar a instância
- [ ] **IMPROVE-06** `adapters/persistence/inmemory` — adicionar suporte a TTL por entidade via `WithTTL(d time.Duration)` option, com goroutine de expiração
- [ ] **IMPROVE-07** `kernel/errors/errors.go` — adicionar `Problem.WithDetails(map[string]any)` para merge em lote de detalhes ao invés de chamar `WithDetail` em cadeia
- [ ] **IMPROVE-08** `kernel/result/result.go` — adicionar `Result.Or(fallback T) T` e `Result.OrElse(func() T) T` para facilitar pipelines sem panic
- [ ] **IMPROVE-09** `ports/files/port.go` — adicionar `StorageClassIntelligentTiering` e `StorageClassColdline` aos enums de `StorageClass` para compatibilidade com AWS e GCS
- [ ] **IMPROVE-10** `adapters/email/sendgrid/client.go` — adicionar retry automático em respostas 429 e 5xx com backoff configurável, alinhado ao `app/resilience`

---

## Novos adapters

- [ ] **ADAPTER-01** Criar `adapters/cache/redis` — implementação de `CachePort` usando `github.com/redis/go-redis/v9`; incluir testes com `miniredis`
- [ ] **ADAPTER-02** Criar `adapters/queue/sqs` — implementação de `QueuePort` usando AWS SQS via `aws-sdk-go-v2`
- [ ] **ADAPTER-03** Criar `adapters/queue/rabbitmq` — implementação de `QueuePort` usando `github.com/rabbitmq/amqp091-go`
- [ ] **ADAPTER-04** Criar `adapters/compression/stdlib` — implementação de `CompressionPort` usando `compress/gzip` e `compress/flate` da stdlib, sem dependências externas
- [ ] **ADAPTER-05** Criar `adapters/push/fcm` — implementação de `PushPort` usando Firebase Cloud Messaging (`firebase-admin-go`)
- [ ] **ADAPTER-06** Criar `adapters/email/sesv2` — implementação completa de `EmailPort` usando AWS SES v2 com suporte a `SendWithTemplate`
- [ ] **ADAPTER-07** Criar `adapters/secrets/awsssm` — implementação de `SecretsPort` usando AWS SSM Parameter Store
- [ ] **ADAPTER-08** Criar `adapters/secrets/vault` — implementação de `SecretsPort` usando HashiCorp Vault HTTP API

---

## Novas portas

- [ ] **PORT-01** Criar `ports/transaction/port.go` — interface `TransactionManager` com `Begin`, `Commit`, `Rollback` e `WithTx(ctx, func) error` para gerenciamento de transações agnóstico de banco
- [ ] **PORT-02** Criar `ports/ratelimit/port.go` — interface `RateLimiter` com `Allow(ctx, key) bool` e `Wait(ctx, key) error` para throttling no nível de porta
- [ ] **PORT-03** Criar `ports/featureflag/port.go` — interface `FeatureFlagPort` com `IsEnabled(ctx, flag, userID) bool` para feature toggles
- [ ] **PORT-04** Criar `ports/audit/port.go` — interface `AuditLog` com `Record(ctx, AuditEvent) error`; `AuditEvent` inclui `ActorID`, `Action`, `Resource`, `OccurredAt`
- [ ] **PORT-05** Criar `ports/eventbus/port.go` — interface `EventBus` com `Publish(ctx, topic, event)` e `Subscribe(ctx, topic, handler)`; diferente de `QueuePort` por ser topic-based

---

## Tooling e CI/CD

- [ ] **TOOL-01** Criar `.github/workflows/ci.yml` com jobs: `test` (go test -race), `lint` (golangci-lint), `tidy-check` (go mod tidy + git diff --exit-code)
- [ ] **TOOL-02** Adicionar `make fmt` ao Makefile executando `gofmt -w` e `goimports -w` em todos os arquivos Go
- [ ] **TOOL-03** Adicionar `make bench` ao Makefile executando `go test -bench=. -benchmem ./...`
- [ ] **TOOL-04** Adicionar `make vulncheck` ao Makefile usando `govulncheck ./...` para detectar dependências com CVEs conhecidos
- [ ] **TOOL-05** Adicionar `make mock` ao Makefile usando `mockery` ou `moq` para gerar mocks das interfaces de porta em `testkit/mocks/`
- [ ] **TOOL-06** Adicionar linters ao `.golangci.yml`: `nilnil`, `gocritic`, `errname`, `cyclop` (max complexity 15), `exhaustruct` para structs críticas
- [ ] **TOOL-07** Criar `docker-compose.yml` com serviços para testes de integração: Redis, RabbitMQ, LocalStack (S3/SES/SQS), Elasticsearch, OpenSearch
- [ ] **TOOL-08** Adicionar `make coverage-report` que gera relatório HTML e falha se cobertura total do módulo raiz for < 70%

---

## Segurança e compliance

- [ ] **SEC-01** Adicionar `adapters/payment/stripe/webhook.go` com `VerifyWebhookSignature(payload []byte, sig, secret string) error` usando `stripe.ConstructEvent`
- [ ] **SEC-02** Criar `docs/security.md` com guia de: não logar `Problem.Details` em produção sem sanitização, uso correto do `LogSanitizer`, e rotação de secrets
- [ ] **SEC-03** Expandir `app/observability/sanitizer.go` para suportar redação recursiva em maps aninhados (`map[string]any` dentro de `map[string]any`)
- [ ] **SEC-04** Adicionar à lista de `defaultSensitiveKeys` do sanitizador: `api_key`, `private_key`, `access_token`, `refresh_token`, `ssn`, `cnpj`, `rg`
