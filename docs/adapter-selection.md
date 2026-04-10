# Adapter Selection Guide

## Email

| Adapter | Templates | Tracking | Auth | Best for |
|---|---|---|---|---|
| `email/smtp` | No (render first) | No | Username/password | Internal apps, transactional email via your own SMTP |
| `email/ses` | No (`SendWithTemplate` not supported) | Bounce/complaint via SNS | AWS IAM | Simple bulk email on AWS; upgrade to sesv2 for templates |
| `email/sesv2` | Yes (Handlebars `{{var}}`) | Bounce/complaint via SNS | AWS IAM | Full-featured email on AWS with template support |
| `email/sendgrid` | Yes (dynamic templates) | Opens/clicks/bounces | API Key | Marketing + transactional with advanced analytics |

> **Note:** `email/ses` returns `ErrTechnical` with a `migrate_to: adapters/email/sesv2` detail when `SendWithTemplate` is called. Migrate before using templates.

---

## File Storage

| Adapter | Presign | SSE | Multipart | Best for |
|---|---|---|---|---|
| `files/s3` | GET/PUT/DELETE | AES256, KMS | Auto (via s3manager) | AWS-native, most feature-complete |
| `files/gcs` | GET/PUT | Google-managed | Auto | GCP-native |

---

## Queue / Messaging

| Adapter | Ordering | Fan-out | Retry | Best for |
|---|---|---|---|---|
| `queue/sqs` | Per-queue FIFO (with FIFO queue type) | No (use SNS) | Visibility timeout + DLQ | AWS-native, serverless, simple pipelines |
| `queue/rabbitmq` | Per-queue | Via exchanges/bindings | Nack + requeue | Complex routing, on-premise, AMQP ecosystems |

> **Difference from EventBus:** `QueuePort` is for point-to-point (work queues). Use `ports/eventbus` for topic-based fan-out.

---

## Cache

| Adapter | TTL | Pub/Sub | Cluster | Best for |
|---|---|---|---|---|
| `cache/redis` | Yes | Yes (not via port) | Yes (with go-redis) | General purpose; most production deployments |

---

## Secrets

| Adapter | Rotation | Dynamic secrets | Auth | Best for |
|---|---|---|---|---|
| `secrets/awsssm` | Manual via Parameter Store | No | IAM role | AWS-native; SecureString with KMS encryption |
| `secrets/vault` | Yes (via Vault leases) | Yes (DB, AWS, PKI) | Token / AppRole / K8s | Multi-cloud, advanced secret lifecycle |

> **vault** uses only `net/http` from the stdlib — no external dependencies. Auth token management (renewal, AppRole exchange) is the caller's responsibility.

---

## Push Notifications

| Adapter | Platforms | Topics | Best for |
|---|---|---|---|
| `push/fcm` | Android, iOS (via APNs bridge), Web | Yes | Cross-platform push via Firebase |

---

## SMS

| Adapter | Two-way | Alpha sender | Best for |
|---|---|---|---|
| `sms/twilio` | Yes | Yes | Most production use cases |

---

## Compression

| Adapter | Formats | External deps | Best for |
|---|---|---|---|
| `compression/stdlib` | gzip, flate (deflate) | None | Serverless, minimal footprint |

> zstd and snappy are defined as `Format` constants but require a custom adapter implementation — the stdlib adapter only supports gzip and flate.

---

## Search

| Adapter | Version | Best for |
|---|---|---|
| `search/elasticsearch` | 8.x | Elasticsearch-native deployments |
| `search/opensearch` | 2.x | AWS OpenSearch Service or self-hosted OpenSearch |

The two adapters are nearly identical in API. Choose based on your provider.

---

## Tracing

| Adapter | Protocol | Best for |
|---|---|---|
| `tracing/otel` | OTLP (gRPC/HTTP) | OpenTelemetry-compatible backends (Jaeger, Tempo, Datadog, etc.) |
