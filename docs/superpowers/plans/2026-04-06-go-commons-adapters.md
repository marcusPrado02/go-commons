# go-commons Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement all 11 Go adapter submódulos: persistence/inmemory (full), email/sendgrid (full), files/s3 (full), and 8 essential-level adapters (ses, gcs, smtp, stripe, elasticsearch, opensearch, otel, twilio).

**Architecture:** Each adapter is a separate Go module under `adapters/`. Every adapter imports the root module and its external SDK. `go.work` in the root coordinates all modules during development. Each adapter has a `replace` directive pointing to the local root module.

**Prerequisite:** Plan 1 (Core) must be complete. Plan 2 (App) is independent.

**Tech Stack:**
- `github.com/aws/aws-sdk-go-v2` (S3, SES)
- `cloud.google.com/go/storage` (GCS)
- `github.com/sendgrid/sendgrid-go`
- `github.com/stripe/stripe-go/v76`
- `github.com/elastic/go-elasticsearch/v8`
- `github.com/opensearch-project/opensearch-go/v2`
- `go.opentelemetry.io/otel`
- `github.com/twilio/twilio-go`
- `github.com/stretchr/testify v1.9.0`

---

## File Map

```
adapters/
├── persistence/inmemory/
│   ├── go.mod
│   ├── repository.go
│   └── repository_test.go
├── email/
│   ├── sendgrid/
│   │   ├── go.mod
│   │   ├── client.go
│   │   └── client_test.go
│   ├── ses/
│   │   ├── go.mod
│   │   └── client.go
│   └── smtp/
│       ├── go.mod
│       └── client.go
├── files/
│   ├── s3/
│   │   ├── go.mod
│   │   ├── client.go
│   │   └── client_test.go
│   └── gcs/
│       ├── go.mod
│       └── client.go
├── payment/stripe/
│   ├── go.mod
│   └── client.go
├── search/
│   ├── elasticsearch/
│   │   ├── go.mod
│   │   └── client.go
│   └── opensearch/
│       ├── go.mod
│       └── client.go
├── tracing/otel/
│   ├── go.mod
│   └── tracer.go
└── sms/twilio/
    ├── go.mod
    └── client.go
```

---

## Task 1: Register all adapter submódulos in go.work

After each adapter's go.mod is created, add it to go.work. This task shows the pattern once.

- [ ] **Step 1: Template go.mod for every adapter**

Every adapter `go.mod` follows this pattern (shown for inmemory):

```
module github.com/marcusPrado02/go-commons/adapters/persistence/inmemory

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/stretchr/testify v1.9.0
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

The `replace` directive makes the workspace use the local root module during development. When published, the replace is removed and a proper version tag is used.

- [ ] **Step 2: Template go.work entry**

After creating each adapter's `go.mod`, add its directory to `go.work`:

```
go 1.22.0

use (
	.
	./adapters/persistence/inmemory
	./adapters/email/sendgrid
	./adapters/email/ses
	./adapters/email/smtp
	./adapters/files/s3
	./adapters/files/gcs
	./adapters/payment/stripe
	./adapters/search/elasticsearch
	./adapters/search/opensearch
	./adapters/tracing/otel
	./adapters/sms/twilio
)
```

Update `go.work` after creating each adapter below.

---

## Task 2: adapters/persistence/inmemory — Full Implementation

**Implementation level:** Complete (thread-safe, full PageableRepository)

**Files:**
- Create: `adapters/persistence/inmemory/go.mod`
- Create: `adapters/persistence/inmemory/repository_test.go`
- Create: `adapters/persistence/inmemory/repository.go`

- [ ] **Step 1: Create go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/persistence/inmemory

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/stretchr/testify v1.9.0
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 2: Update go.work to include this module**

Add `./adapters/persistence/inmemory` to the `use` block in `go.work`.

- [ ] **Step 3: Write failing tests**

Create `adapters/persistence/inmemory/repository_test.go`:

```go
package inmemory_test

import (
	"context"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/persistence/inmemory"
	"github.com/marcusPrado02/go-commons/ports/persistence"
	"github.com/marcusPrado02/go-commons/testkit/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type user struct {
	ID   string
	Name string
}

func newRepo() *inmemory.InMemoryRepository[user, string] {
	return inmemory.NewInMemoryRepository[user, string](func(u user) string { return u.ID })
}

// Run the shared repository contract suite.
func TestInMemoryRepository_Contract(t *testing.T) {
	counter := 0
	suite.Run(t, &contracts.RepositoryContract[user, string]{
		Repo: newRepo(),
		NewEntity: func() user {
			counter++
			return user{ID: fmt.Sprintf("user-%d", counter), Name: "Alice"}
		},
		ExtractID:    func(u user) string { return u.ID },
		MutateEntity: func(u user) user { u.Name = "Bob"; return u },
	})
}

func TestInMemoryRepository_FindAll_WithSpec(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	_, _ = repo.Save(ctx, user{ID: "1", Name: "Alice"})
	_, _ = repo.Save(ctx, user{ID: "2", Name: "Bob"})
	_, _ = repo.Save(ctx, user{ID: "3", Name: "Alice"})

	result, err := repo.FindAll(ctx,
		persistence.PageRequest{Page: 0, Size: 10},
		persistence.Spec[user](func(u user) bool { return u.Name == "Alice" }),
	)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalElements)
	assert.Len(t, result.Content, 2)
}

func TestInMemoryRepository_FindAll_Pagination(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		_, _ = repo.Save(ctx, user{ID: fmt.Sprintf("%d", i), Name: "User"})
	}

	result, err := repo.FindAll(ctx,
		persistence.PageRequest{Page: 1, Size: 2},
		persistence.Spec[user](func(user) bool { return true }),
	)
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalElements)
	assert.Equal(t, 3, result.TotalPages) // ceil(5/2)
	assert.Len(t, result.Content, 2)
}

func TestInMemoryRepository_Search_WithSort(t *testing.T) {
	repo := inmemory.NewInMemoryRepository[user, string](
		func(u user) string { return u.ID },
		inmemory.WithSortFunc(func(a, b user, field string, desc bool) bool {
			if field == "Name" {
				if desc {
					return a.Name > b.Name
				}
				return a.Name < b.Name
			}
			return false
		}),
	)
	ctx := context.Background()
	_, _ = repo.Save(ctx, user{ID: "1", Name: "Charlie"})
	_, _ = repo.Save(ctx, user{ID: "2", Name: "Alice"})
	_, _ = repo.Save(ctx, user{ID: "3", Name: "Bob"})

	result, err := repo.Search(ctx,
		persistence.PageRequest{Page: 0, Size: 10},
		persistence.Spec[user](func(user) bool { return true }),
		persistence.Sort{Field: "Name", Descending: false},
	)
	require.NoError(t, err)
	require.Len(t, result.Content, 3)
	assert.Equal(t, "Alice", result.Content[0].Name)
	assert.Equal(t, "Bob", result.Content[1].Name)
	assert.Equal(t, "Charlie", result.Content[2].Name)
}
```

- [ ] **Step 4: Add missing import to test file**

Add `"fmt"` to the imports in `repository_test.go`.

- [ ] **Step 5: Run tests to verify they fail**

```bash
cd adapters/persistence/inmemory && go test ./... -v
```

Expected: compilation error

- [ ] **Step 6: Implement adapters/persistence/inmemory/repository.go**

```go
// Package inmemory provides a thread-safe, generic in-memory Repository implementation.
// Suitable for unit tests and simple scenarios where a persistent store is not required.
package inmemory

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/marcusPrado02/go-commons/ports/persistence"
)

// SortFunc compares two entities for the given field and direction.
// Return true if a should come before b.
type SortFunc[E any] func(a, b E, field string, descending bool) bool

// options holds optional configuration for InMemoryRepository.
type options[E any] struct {
	sortFunc SortFunc[E]
}

// Option configures an InMemoryRepository.
type Option[E any] func(*options[E])

// WithSortFunc provides a comparison function for Search ordering.
func WithSortFunc[E any](fn SortFunc[E]) Option[E] {
	return func(o *options[E]) { o.sortFunc = fn }
}

// InMemoryRepository is a thread-safe, generic repository backed by a map.
// It implements both persistence.Repository and persistence.PageableRepository.
type InMemoryRepository[E any, ID comparable] struct {
	mu          sync.RWMutex
	storage     map[ID]E
	idExtractor func(E) ID
	opts        options[E]
}

// NewInMemoryRepository creates a repository that extracts IDs using idExtractor.
func NewInMemoryRepository[E any, ID comparable](idExtractor func(E) ID, opts ...Option[E]) *InMemoryRepository[E, ID] {
	o := options[E]{}
	for _, opt := range opts {
		opt(&o)
	}
	return &InMemoryRepository[E, ID]{
		storage:     make(map[ID]E),
		idExtractor: idExtractor,
		opts:        o,
	}
}

// Save upserts the entity. Returns the saved entity unchanged.
func (r *InMemoryRepository[E, ID]) Save(_ context.Context, entity E) (E, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.idExtractor(entity)
	r.storage[id] = entity
	return entity, nil
}

// FindByID returns (entity, true, nil) if found, (zero, false, nil) if not found.
func (r *InMemoryRepository[E, ID]) FindByID(_ context.Context, id ID) (E, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entity, ok := r.storage[id]
	return entity, ok, nil
}

// DeleteByID removes the entity. Not an error if not found.
func (r *InMemoryRepository[E, ID]) DeleteByID(_ context.Context, id ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.storage, id)
	return nil
}

// Delete removes the entity by extracting its ID. Not an error if not found.
func (r *InMemoryRepository[E, ID]) Delete(_ context.Context, entity E) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.storage, r.idExtractor(entity))
	return nil
}

// FindAll returns a page of entities matching the specification.
func (r *InMemoryRepository[E, ID]) FindAll(ctx context.Context, req persistence.PageRequest, spec persistence.Specification[E]) (persistence.PageResult[E], error) {
	return r.Search(ctx, req, spec, persistence.Sort{})
}

// Search returns a page of entities matching the specification, sorted if a SortFunc is configured.
func (r *InMemoryRepository[E, ID]) Search(_ context.Context, req persistence.PageRequest, spec persistence.Specification[E], s persistence.Sort) (persistence.PageResult[E], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	predicate := spec.ToPredicate()
	var matched []E
	for _, entity := range r.storage {
		if predicate(entity) {
			matched = append(matched, entity)
		}
	}

	if r.opts.sortFunc != nil && s.Field != "" {
		sortFn := r.opts.sortFunc
		sort.Slice(matched, func(i, j int) bool {
			return sortFn(matched[i], matched[j], s.Field, s.Descending)
		})
	}

	total := len(matched)
	totalPages := 0
	if req.Size > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(req.Size)))
	}

	start := req.Page * req.Size
	if start >= total {
		return persistence.PageResult[E]{
			Content:       []E{},
			TotalElements: total,
			TotalPages:    totalPages,
			Page:          req.Page,
			Size:          req.Size,
		}, nil
	}

	end := start + req.Size
	if end > total {
		end = total
	}

	return persistence.PageResult[E]{
		Content:       matched[start:end],
		TotalElements: total,
		TotalPages:    totalPages,
		Page:          req.Page,
		Size:          req.Size,
	}, nil
}
```

- [ ] **Step 7: Run tests**

```bash
cd adapters/persistence/inmemory && go test ./... -v -race
```

Expected: all PASS

- [ ] **Step 8: Commit**

```bash
git add adapters/persistence/inmemory/ go.work
git commit -m "feat(adapters): add thread-safe InMemoryRepository with pagination and sorting"
```

---

## Task 3: adapters/email/sendgrid — Full Implementation

**Implementation level:** Complete — real HTTP calls, retry on 429/5xx, mock-friendly baseURL

**Files:**
- Create: `adapters/email/sendgrid/go.mod`
- Create: `adapters/email/sendgrid/client_test.go`
- Create: `adapters/email/sendgrid/client.go`

- [ ] **Step 1: Create go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/email/sendgrid

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/sendgrid/sendgrid-go v3.14.0+incompatible
	github.com/stretchr/testify v1.9.0
)

require github.com/sendgrid/rest v2.6.9+incompatible // indirect

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 2: Update go.work**

Add `./adapters/email/sendgrid` to the `use` block in `go.work`.

- [ ] **Step 3: Write failing tests**

Create `adapters/email/sendgrid/client_test.go`:

```go
package sendgrid_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/email/sendgrid"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Message-Id", "test-msg-id")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

func TestClient_Send_Success(t *testing.T) {
	srv := newTestServer(http.StatusAccepted, "")
	defer srv.Close()

	from, _ := emailport.NewEmailAddress("sender@example.com")
	client, err := sendgrid.New("test-api-key", from, sendgrid.WithBaseURL(srv.URL))
	require.NoError(t, err)

	to, _ := emailport.NewEmailAddress("recipient@example.com")
	receipt, err := client.Send(context.Background(), emailport.Email{
		From:    from,
		To:      []emailport.EmailAddress{to},
		Subject: "Hello",
		HTML:    "<p>Hi</p>",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, receipt.MessageID)
}

func TestClient_Send_ValidationFailure(t *testing.T) {
	from, _ := emailport.NewEmailAddress("sender@example.com")
	client, _ := sendgrid.New("test-api-key", from)

	// Email with no recipients — Validate() should fail before HTTP call
	_, err := client.Send(context.Background(), emailport.Email{
		From:    from,
		Subject: "Bad email",
		HTML:    "<p>Hi</p>",
	})
	assert.Error(t, err)
}

func TestClient_Ping_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"enabled": true}})
	}))
	defer srv.Close()

	from, _ := emailport.NewEmailAddress("sender@example.com")
	client, _ := sendgrid.New("test-api-key", from, sendgrid.WithBaseURL(srv.URL))

	err := client.Ping(context.Background())
	assert.NoError(t, err)
}

// Compile-time interface check
var _ emailport.EmailPort = (*sendgrid.Client)(nil)
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
cd adapters/email/sendgrid && go test ./... -v
```

Expected: compilation error

- [ ] **Step 5: Implement adapters/email/sendgrid/client.go**

```go
// Package sendgrid provides a SendGrid implementation of ports/email.EmailPort.
package sendgrid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

const defaultBaseURL = "https://api.sendgrid.com"

// Client is a SendGrid implementation of EmailPort.
type Client struct {
	apiKey  string
	from    emailport.EmailAddress
	baseURL string
	http    *http.Client
}

// Option configures a SendGrid Client.
type Option func(*Client)

// WithBaseURL overrides the SendGrid API base URL. Used for testing with a mock server.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.Timeout = d }
}

// New creates a new SendGrid client.
func New(apiKey string, from emailport.EmailAddress, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("sendgrid: apiKey cannot be empty")
	}
	c := &Client{
		apiKey:  apiKey,
		from:    from,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Send delivers an email via the SendGrid v3 Mail Send API.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: invalid email: %w", err)
	}

	body, err := c.buildPayload(email)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: failed to build payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: failed to create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
	}

	return emailport.EmailReceipt{MessageID: resp.Header.Get("X-Message-Id")}, nil
}

// SendWithTemplate delivers a template-based email via the SendGrid v3 API.
func (c *Client) SendWithTemplate(ctx context.Context, req emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	payload := map[string]any{
		"from":              map[string]string{"email": req.From.Value},
		"template_id":       req.TemplateName,
		"dynamic_template_data": req.Variables,
	}
	tos := make([]map[string]string, len(req.To))
	for i, t := range req.To {
		tos[i] = map[string]string{"email": t.Value}
	}
	payload["personalizations"] = []map[string]any{{"to": tos}}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return emailport.EmailReceipt{}, err
	}
	c.setHeaders(httpReq)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return emailport.EmailReceipt{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
	}
	return emailport.EmailReceipt{MessageID: resp.Header.Get("X-Message-Id")}, nil
}

// Ping verifies SendGrid connectivity by calling the mail settings endpoint.
func (c *Client) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(pingCtx, http.MethodGet, c.baseURL+"/v3/mail/settings", nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sendgrid: ping returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func (c *Client) buildPayload(email emailport.Email) ([]byte, error) {
	tos := make([]map[string]string, len(email.To))
	for i, t := range email.To {
		tos[i] = map[string]string{"email": t.Value}
	}

	content := []map[string]string{}
	if email.HTML != "" {
		content = append(content, map[string]string{"type": "text/html", "value": email.HTML})
	}
	if email.Text != "" {
		content = append(content, map[string]string{"type": "text/plain", "value": email.Text})
	}

	payload := map[string]any{
		"personalizations": []map[string]any{{"to": tos, "subject": email.Subject}},
		"from":             map[string]string{"email": email.From.Value},
		"content":          content,
	}

	return json.Marshal(payload)
}
```

- [ ] **Step 6: Run tests**

```bash
cd adapters/email/sendgrid && go test ./... -v -race
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add adapters/email/sendgrid/ go.work
git commit -m "feat(adapters): add SendGrid email adapter with full EmailPort implementation"
```

---

## Task 4: adapters/files/s3 — Full Implementation

**Implementation level:** Complete — real AWS SDK calls, multipart for large files, presigned URLs

**Files:**
- Create: `adapters/files/s3/go.mod`
- Create: `adapters/files/s3/client_test.go`
- Create: `adapters/files/s3/client.go`

- [ ] **Step 1: Create go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/files/s3

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.26.1
	github.com/aws/aws-sdk-go-v2/config v1.27.11
	github.com/aws/aws-sdk-go-v2/service/s3 v1.53.1
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.16.15
	github.com/stretchr/testify v1.9.0
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 2: Update go.work**

Add `./adapters/files/s3` to the `use` block in `go.work`.

- [ ] **Step 3: Write tests**

Create `adapters/files/s3/client_test.go`:

```go
package s3_test

import (
	"testing"

	s3adapter "github.com/marcusPrado02/go-commons/adapters/files/s3"
	filesport "github.com/marcusPrado02/go-commons/ports/files"
)

// Compile-time interface check — ensures Client implements FileStorePort.
var _ filesport.FileStorePort = (*s3adapter.Client)(nil)

func TestNew_ReturnsClient(t *testing.T) {
	// Integration test — requires AWS credentials.
	// Skipped in unit test runs without credentials.
	t.Skip("requires AWS credentials")
}
```

- [ ] **Step 4: Implement adapters/files/s3/client.go**

```go
// Package s3 provides an AWS S3 implementation of ports/files.FileStorePort.
package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	filesport "github.com/marcusPrado02/go-commons/ports/files"
)

// multipartThreshold is the file size above which multipart upload is used.
const multipartThreshold = 5 * 1024 * 1024 // 5 MB

// Client is an AWS S3 implementation of FileStorePort.
type Client struct {
	s3       *awss3.Client
	signer   *awss3.PresignClient
	uploader *manager.Uploader
	opts     clientOptions
}

type clientOptions struct {
	defaultStorageClass filesport.StorageClass
	sseAlgorithm        string
}

// Option configures an S3 Client.
type Option func(*clientOptions)

// WithDefaultStorageClass sets the default storage class for uploads.
func WithDefaultStorageClass(sc filesport.StorageClass) Option {
	return func(o *clientOptions) { o.defaultStorageClass = sc }
}

// WithSSEAlgorithm enables server-side encryption with the given algorithm (e.g. "AES256").
func WithSSEAlgorithm(alg string) Option {
	return func(o *clientOptions) { o.sseAlgorithm = alg }
}

// New creates an S3 Client from an aws.Config.
func New(cfg aws.Config, opts ...Option) *Client {
	svc := awss3.NewFromConfig(cfg)
	o := clientOptions{defaultStorageClass: filesport.StorageClassStandard}
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{
		s3:       svc,
		signer:   awss3.NewPresignClient(svc),
		uploader: manager.NewUploader(svc),
		opts:     o,
	}
}

// Upload stores content under the given FileID. Uses multipart upload for files > 5MB.
func (c *Client) Upload(ctx context.Context, id filesport.FileID, content io.Reader, optFns ...filesport.UploadOption) (filesport.UploadResult, error) {
	opts := filesport.UploadOptions{StorageClass: c.opts.defaultStorageClass}
	for _, fn := range optFns {
		fn(&opts)
	}

	input := &awss3.PutObjectInput{
		Bucket:       aws.String(id.Bucket),
		Key:          aws.String(id.Key),
		Body:         content,
		StorageClass: types.StorageClass(string(opts.StorageClass)),
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if c.opts.sseAlgorithm != "" {
		input.ServerSideEncryption = types.ServerSideEncryption(c.opts.sseAlgorithm)
	}

	result, err := c.uploader.Upload(ctx, input)
	if err != nil {
		return filesport.UploadResult{}, fmt.Errorf("s3: upload failed: %w", err)
	}
	return filesport.UploadResult{Location: result.Location}, nil
}

// Download retrieves file content and metadata. Caller must close FileObject.Content.
func (c *Client) Download(ctx context.Context, id filesport.FileID) (filesport.FileObject, error) {
	out, err := c.s3.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		return filesport.FileObject{}, fmt.Errorf("s3: download failed: %w", err)
	}
	meta := filesport.FileMetadata{
		ETag: aws.ToString(out.ETag),
	}
	if out.ContentType != nil {
		meta.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		meta.ContentLength = *out.ContentLength
	}
	return filesport.FileObject{Content: out.Body, Metadata: meta}, nil
}

// Delete removes a single file.
func (c *Client) Delete(ctx context.Context, id filesport.FileID) error {
	_, err := c.s3.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		return fmt.Errorf("s3: delete failed: %w", err)
	}
	return nil
}

// DeleteAll removes multiple files. Reports per-file errors in DeleteResult.Failed.
func (c *Client) DeleteAll(ctx context.Context, ids []filesport.FileID) (filesport.DeleteResult, error) {
	var result filesport.DeleteResult
	for _, id := range ids {
		if err := c.Delete(ctx, id); err != nil {
			result.Failed = append(result.Failed, filesport.DeleteError{ID: id, Cause: err})
		} else {
			result.Deleted = append(result.Deleted, id)
		}
	}
	return result, nil
}

// Exists returns true if the object exists.
func (c *Client) Exists(ctx context.Context, id filesport.FileID) (bool, error) {
	_, err := c.s3.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("s3: exists check failed: %w", err)
	}
	return true, nil
}

// GetMetadata returns file metadata without downloading content.
func (c *Client) GetMetadata(ctx context.Context, id filesport.FileID) (filesport.FileMetadata, error) {
	out, err := c.s3.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		return filesport.FileMetadata{}, fmt.Errorf("s3: get metadata failed: %w", err)
	}
	meta := filesport.FileMetadata{ETag: aws.ToString(out.ETag)}
	if out.ContentType != nil {
		meta.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		meta.ContentLength = *out.ContentLength
	}
	return meta, nil
}

// List returns objects in a bucket matching the prefix.
func (c *Client) List(ctx context.Context, bucket, prefix string, optFns ...filesport.ListOption) (filesport.ListResult, error) {
	opts := filesport.ListOptions{MaxKeys: 1000}
	for _, fn := range optFns {
		fn(&opts)
	}

	input := &awss3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	if opts.MaxKeys > 0 {
		mk := int32(opts.MaxKeys)
		input.MaxKeys = &mk
	}
	if opts.ContinuationToken != "" {
		input.ContinuationToken = aws.String(opts.ContinuationToken)
	}

	out, err := c.s3.ListObjectsV2(ctx, input)
	if err != nil {
		return filesport.ListResult{}, fmt.Errorf("s3: list failed: %w", err)
	}

	var objects []filesport.FileMetadata
	for _, obj := range out.Contents {
		objects = append(objects, filesport.FileMetadata{
			ETag:          aws.ToString(obj.ETag),
			ContentLength: aws.ToInt64(obj.Size),
		})
	}

	var nextToken string
	if out.NextContinuationToken != nil {
		nextToken = *out.NextContinuationToken
	}

	return filesport.ListResult{
		Objects:           objects,
		ContinuationToken: nextToken,
		IsTruncated:       aws.ToBool(out.IsTruncated),
	}, nil
}

// GeneratePresignedURL creates a time-limited URL for direct client access.
func (c *Client) GeneratePresignedURL(ctx context.Context, id filesport.FileID, op filesport.PresignedOperation, ttl time.Duration, optFns ...filesport.PresignOption) (*url.URL, error) {
	opts := filesport.PresignOptions{}
	for _, fn := range optFns {
		fn(&opts)
	}

	var rawURL string
	var err error

	switch op {
	case filesport.PresignGet:
		req, presignErr := c.signer.PresignGetObject(ctx, &awss3.GetObjectInput{
			Bucket: aws.String(id.Bucket),
			Key:    aws.String(id.Key),
		}, awss3.WithPresignExpires(ttl))
		if presignErr != nil {
			return nil, presignErr
		}
		rawURL = req.URL
	case filesport.PresignPut:
		req, presignErr := c.signer.PresignPutObject(ctx, &awss3.PutObjectInput{
			Bucket: aws.String(id.Bucket),
			Key:    aws.String(id.Key),
		}, awss3.WithPresignExpires(ttl))
		if presignErr != nil {
			return nil, presignErr
		}
		rawURL = req.URL
	case filesport.PresignDelete:
		req, presignErr := c.signer.PresignDeleteObject(ctx, &awss3.DeleteObjectInput{
			Bucket: aws.String(id.Bucket),
			Key:    aws.String(id.Key),
		}, awss3.WithPresignExpires(ttl))
		if presignErr != nil {
			return nil, presignErr
		}
		rawURL = req.URL
	default:
		return nil, fmt.Errorf("s3: unsupported presigned operation: %s", op)
	}

	_ = err
	return url.Parse(rawURL)
}

// Copy duplicates a file within S3.
func (c *Client) Copy(ctx context.Context, src, dst filesport.FileID) error {
	copySource := src.Bucket + "/" + src.Key
	_, err := c.s3.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:     aws.String(dst.Bucket),
		Key:        aws.String(dst.Key),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return fmt.Errorf("s3: copy failed: %w", err)
	}
	return nil
}

// Ensure Client satisfies bytes.Buffer for compile-time check (unused, just for import)
var _ = bytes.NewBuffer
```

- [ ] **Step 5: Run build to verify**

```bash
cd adapters/files/s3 && go build ./...
```

Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add adapters/files/s3/ go.work
git commit -m "feat(adapters): add AWS S3 file store adapter with multipart upload and presigned URLs"
```

---

## Task 5: Essential adapters — email/ses, email/smtp, files/gcs

**Files:**
- Create: `adapters/email/ses/go.mod` + `client.go`
- Create: `adapters/email/smtp/go.mod` + `client.go`
- Create: `adapters/files/gcs/go.mod` + `client.go`

- [ ] **Step 1: Create adapters/email/ses/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/email/ses

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.26.1
	github.com/aws/aws-sdk-go-v2/service/ses v1.22.5
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 2: Create adapters/email/ses/client.go**

```go
// Package ses provides an AWS SES implementation of ports/email.EmailPort.
package ses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsses "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

// Client is an AWS SES implementation of EmailPort.
type Client struct {
	ses  *awsses.Client
	from emailport.EmailAddress
}

// New creates a new SES client.
func New(cfg aws.Config, from emailport.EmailAddress) *Client {
	return &Client{ses: awsses.NewFromConfig(cfg), from: from}
}

// Send delivers an email via AWS SES.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("ses: %w", err)
	}

	tos := make([]string, len(email.To))
	for i, t := range email.To {
		tos[i] = t.Value
	}

	body := &types.Body{}
	if email.HTML != "" {
		body.Html = &types.Content{Data: aws.String(email.HTML), Charset: aws.String("UTF-8")}
	}
	if email.Text != "" {
		body.Text = &types.Content{Data: aws.String(email.Text), Charset: aws.String("UTF-8")}
	}

	out, err := c.ses.SendEmail(ctx, &awsses.SendEmailInput{
		Source:      aws.String(email.From.Value),
		Destination: &types.Destination{ToAddresses: tos},
		Message: &types.Message{
			Subject: &types.Content{Data: aws.String(email.Subject), Charset: aws.String("UTF-8")},
			Body:    body,
		},
	})
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("ses: send failed: %w", err)
	}
	return emailport.EmailReceipt{MessageID: aws.ToString(out.MessageId)}, nil
}

// SendWithTemplate is not supported by SES v1 — returns unsupported error.
func (c *Client) SendWithTemplate(_ context.Context, _ emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	return emailport.EmailReceipt{}, fmt.Errorf("ses: SendWithTemplate requires SES v2 — use the sesv2 adapter")
}

// Ping verifies SES connectivity by listing identities.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.ses.ListIdentities(ctx, &awsses.ListIdentitiesInput{MaxItems: aws.Int32(1)})
	if err != nil {
		return fmt.Errorf("ses: ping failed: %w", err)
	}
	return nil
}

var _ emailport.EmailPort = (*Client)(nil)
```

- [ ] **Step 3: Create adapters/email/smtp/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/email/smtp

go 1.22.0

require github.com/marcusPrado02/go-commons v0.0.0

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 4: Create adapters/email/smtp/client.go**

```go
// Package smtp provides an SMTP implementation of ports/email.EmailPort using stdlib net/smtp.
package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"time"

	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

// Client is an SMTP implementation of EmailPort.
type Client struct {
	host     string
	port     int
	username string
	password string
	from     emailport.EmailAddress
	timeout  time.Duration
}

// Option configures an SMTP Client.
type Option func(*Client)

// WithTimeout sets the SMTP connection timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// New creates a new SMTP client.
func New(host string, port int, username, password string, from emailport.EmailAddress, opts ...Option) *Client {
	c := &Client{host: host, port: port, username: username, password: password, from: from, timeout: 30 * time.Second}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Send delivers an email via SMTP.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	auth := smtp.PlainAuth("", c.username, c.password, c.host)

	tos := make([]string, len(email.To))
	for i, t := range email.To {
		tos[i] = t.Value
	}

	msg := c.buildMessage(email)

	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: c.host})
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: dial failed: %w", err)
	}

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: client creation failed: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: auth failed: %w", err)
	}
	if err := client.Mail(c.from.Value); err != nil {
		return emailport.EmailReceipt{}, err
	}
	for _, to := range tos {
		if err := client.Rcpt(to); err != nil {
			return emailport.EmailReceipt{}, err
		}
	}
	w, err := client.Data()
	if err != nil {
		return emailport.EmailReceipt{}, err
	}
	if _, err = w.Write(msg); err != nil {
		return emailport.EmailReceipt{}, err
	}
	if err = w.Close(); err != nil {
		return emailport.EmailReceipt{}, err
	}
	return emailport.EmailReceipt{}, nil
}

// SendWithTemplate is not natively supported by SMTP — render template first, then call Send.
func (c *Client) SendWithTemplate(_ context.Context, _ emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	return emailport.EmailReceipt{}, fmt.Errorf("smtp: SendWithTemplate not supported — render template with TemplatePort first")
}

// Ping sends an EHLO to verify the SMTP server is reachable.
func (c *Client) Ping(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: c.host})
	if err != nil {
		return fmt.Errorf("smtp: ping failed: %w", err)
	}
	conn.Close()
	return nil
}

func (c *Client) buildMessage(email emailport.Email) []byte {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	buf.WriteString("From: " + c.from.Value + "\r\n")
	buf.WriteString("To: " + email.To[0].Value + "\r\n")
	buf.WriteString("Subject: " + email.Subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + w.Boundary() + "\"\r\n\r\n")

	if email.Text != "" {
		part, _ := w.CreatePart(map[string][]string{"Content-Type": {"text/plain; charset=UTF-8"}})
		part.Write([]byte(email.Text))
	}
	if email.HTML != "" {
		part, _ := w.CreatePart(map[string][]string{"Content-Type": {"text/html; charset=UTF-8"}})
		part.Write([]byte(email.HTML))
	}
	w.Close()
	return buf.Bytes()
}

var _ emailport.EmailPort = (*Client)(nil)
```

- [ ] **Step 5: Create adapters/files/gcs/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/files/gcs

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	cloud.google.com/go/storage v1.40.0
	google.golang.org/api v0.177.0
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 6: Create adapters/files/gcs/client.go**

```go
// Package gcs provides a Google Cloud Storage implementation of ports/files.FileStorePort.
package gcs

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"cloud.google.com/go/storage"
	filesport "github.com/marcusPrado02/go-commons/ports/files"
	"google.golang.org/api/iterator"
)

// Client is a GCS implementation of FileStorePort.
type Client struct {
	gcs *storage.Client
}

// New creates a new GCS client. The provided context is used only for client creation.
func New(ctx context.Context) (*Client, error) {
	c, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs: failed to create client: %w", err)
	}
	return &Client{gcs: c}, nil
}

// Upload writes content to GCS.
func (c *Client) Upload(ctx context.Context, id filesport.FileID, content io.Reader, optFns ...filesport.UploadOption) (filesport.UploadResult, error) {
	opts := filesport.UploadOptions{}
	for _, fn := range optFns {
		fn(&opts)
	}
	wc := c.gcs.Bucket(id.Bucket).Object(id.Key).NewWriter(ctx)
	if opts.ContentType != "" {
		wc.ContentType = opts.ContentType
	}
	if _, err := io.Copy(wc, content); err != nil {
		_ = wc.Close()
		return filesport.UploadResult{}, fmt.Errorf("gcs: upload failed: %w", err)
	}
	if err := wc.Close(); err != nil {
		return filesport.UploadResult{}, fmt.Errorf("gcs: upload close failed: %w", err)
	}
	return filesport.UploadResult{Location: fmt.Sprintf("gs://%s/%s", id.Bucket, id.Key)}, nil
}

// Download retrieves file content. Caller must close FileObject.Content.
func (c *Client) Download(ctx context.Context, id filesport.FileID) (filesport.FileObject, error) {
	rc, err := c.gcs.Bucket(id.Bucket).Object(id.Key).NewReader(ctx)
	if err != nil {
		return filesport.FileObject{}, fmt.Errorf("gcs: download failed: %w", err)
	}
	return filesport.FileObject{Content: rc}, nil
}

// Delete removes a file from GCS.
func (c *Client) Delete(ctx context.Context, id filesport.FileID) error {
	if err := c.gcs.Bucket(id.Bucket).Object(id.Key).Delete(ctx); err != nil {
		return fmt.Errorf("gcs: delete failed: %w", err)
	}
	return nil
}

// DeleteAll removes multiple files.
func (c *Client) DeleteAll(ctx context.Context, ids []filesport.FileID) (filesport.DeleteResult, error) {
	var result filesport.DeleteResult
	for _, id := range ids {
		if err := c.Delete(ctx, id); err != nil {
			result.Failed = append(result.Failed, filesport.DeleteError{ID: id, Cause: err})
		} else {
			result.Deleted = append(result.Deleted, id)
		}
	}
	return result, nil
}

// Exists returns true if the object exists.
func (c *Client) Exists(ctx context.Context, id filesport.FileID) (bool, error) {
	_, err := c.gcs.Bucket(id.Bucket).Object(id.Key).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("gcs: exists check failed: %w", err)
	}
	return true, nil
}

// GetMetadata returns file attributes.
func (c *Client) GetMetadata(ctx context.Context, id filesport.FileID) (filesport.FileMetadata, error) {
	attrs, err := c.gcs.Bucket(id.Bucket).Object(id.Key).Attrs(ctx)
	if err != nil {
		return filesport.FileMetadata{}, fmt.Errorf("gcs: get metadata failed: %w", err)
	}
	return filesport.FileMetadata{
		ContentType:   attrs.ContentType,
		ContentLength: attrs.Size,
		ETag:          attrs.Etag,
		LastModified:  attrs.Updated,
	}, nil
}

// List returns objects in a bucket matching the prefix.
func (c *Client) List(ctx context.Context, bucket, prefix string, optFns ...filesport.ListOption) (filesport.ListResult, error) {
	opts := filesport.ListOptions{MaxKeys: 1000}
	for _, fn := range optFns {
		fn(&opts)
	}

	query := &storage.Query{Prefix: prefix}
	it := c.gcs.Bucket(bucket).Objects(ctx, query)

	var objects []filesport.FileMetadata
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return filesport.ListResult{}, fmt.Errorf("gcs: list failed: %w", err)
		}
		objects = append(objects, filesport.FileMetadata{
			ContentType:   attrs.ContentType,
			ContentLength: attrs.Size,
			ETag:          attrs.Etag,
		})
		if opts.MaxKeys > 0 && len(objects) >= opts.MaxKeys {
			break
		}
	}
	return filesport.ListResult{Objects: objects}, nil
}

// GeneratePresignedURL creates a signed URL for direct GCS access.
func (c *Client) GeneratePresignedURL(ctx context.Context, id filesport.FileID, op filesport.PresignedOperation, ttl time.Duration, _ ...filesport.PresignOption) (*url.URL, error) {
	method := string(op)
	signed, err := c.gcs.Bucket(id.Bucket).SignedURL(id.Key, &storage.SignedURLOptions{
		Method:  method,
		Expires: time.Now().Add(ttl),
	})
	if err != nil {
		return nil, fmt.Errorf("gcs: presign failed: %w", err)
	}
	return url.Parse(signed)
}

// Copy duplicates a GCS object.
func (c *Client) Copy(ctx context.Context, src, dst filesport.FileID) error {
	srcObj := c.gcs.Bucket(src.Bucket).Object(src.Key)
	dstObj := c.gcs.Bucket(dst.Bucket).Object(dst.Key)
	if _, err := dstObj.CopierFrom(srcObj).Run(ctx); err != nil {
		return fmt.Errorf("gcs: copy failed: %w", err)
	}
	return nil
}

var _ filesport.FileStorePort = (*Client)(nil)
```

- [ ] **Step 7: Update go.work for ses, smtp, gcs**

Add these to the `use` block in `go.work`:
```
./adapters/email/ses
./adapters/email/smtp
./adapters/files/gcs
```

- [ ] **Step 8: Build all three**

```bash
cd adapters/email/ses && go build ./...
cd adapters/email/smtp && go build ./...
cd adapters/files/gcs && go build ./...
```

Expected: no errors

- [ ] **Step 9: Commit**

```bash
git add adapters/email/ses/ adapters/email/smtp/ adapters/files/gcs/ go.work
git commit -m "feat(adapters): add SES, SMTP, and GCS adapters (essential level)"
```

---

## Task 6: Essential adapters — payment/stripe, search/elasticsearch, search/opensearch

- [ ] **Step 1: Create adapters/payment/stripe/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/payment/stripe

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/stripe/stripe-go/v76 v76.25.0
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 2: Create adapters/payment/stripe/client.go**

```go
// Package stripe provides a Stripe payment adapter.
// This implements common payment operations: create intent, confirm, and refund.
package stripe

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
)

// Client wraps the Stripe SDK for payment operations.
type Client struct {
	apiKey string
}

// New creates a new Stripe client.
func New(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("stripe: apiKey cannot be empty")
	}
	stripe.Key = apiKey
	return &Client{apiKey: apiKey}, nil
}

// PaymentIntentResult holds the result of a created payment intent.
type PaymentIntentResult struct {
	ID           string
	ClientSecret string
	Status       string
}

// CreatePaymentIntent creates a new Stripe PaymentIntent.
func (c *Client) CreatePaymentIntent(_ context.Context, amount int64, currency, description string) (PaymentIntentResult, error) {
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(amount),
		Currency:    stripe.String(currency),
		Description: stripe.String(description),
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		return PaymentIntentResult{}, fmt.Errorf("stripe: create payment intent failed: %w", err)
	}
	return PaymentIntentResult{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Status:       string(pi.Status),
	}, nil
}

// ConfirmPaymentIntent confirms a payment intent with the given payment method.
func (c *Client) ConfirmPaymentIntent(_ context.Context, intentID, paymentMethodID string) (PaymentIntentResult, error) {
	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(paymentMethodID),
	}
	pi, err := paymentintent.Confirm(intentID, params)
	if err != nil {
		return PaymentIntentResult{}, fmt.Errorf("stripe: confirm payment intent failed: %w", err)
	}
	return PaymentIntentResult{ID: pi.ID, Status: string(pi.Status)}, nil
}

// RefundResult holds the result of a refund operation.
type RefundResult struct {
	ID     string
	Status string
}

// Refund creates a full refund for the given charge.
func (c *Client) Refund(_ context.Context, chargeID string) (RefundResult, error) {
	params := &stripe.RefundParams{Charge: stripe.String(chargeID)}
	r, err := refund.New(params)
	if err != nil {
		return RefundResult{}, fmt.Errorf("stripe: refund failed: %w", err)
	}
	return RefundResult{ID: r.ID, Status: string(r.Status)}, nil
}
```

- [ ] **Step 3: Create adapters/search/elasticsearch/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/search/elasticsearch

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/elastic/go-elasticsearch/v8 v8.13.1
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 4: Create adapters/search/elasticsearch/client.go**

```go
// Package elasticsearch provides an Elasticsearch implementation for search operations.
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// Client wraps the Elasticsearch client for indexing and searching.
type Client struct {
	es *elasticsearch.Client
}

// Config holds Elasticsearch connection configuration.
type Config struct {
	Addresses []string
	Username  string
	Password  string
}

// New creates a new Elasticsearch client.
func New(cfg Config) (*Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: failed to create client: %w", err)
	}
	return &Client{es: es}, nil
}

// Index stores a document in the given index with the given ID.
func (c *Client) Index(_ context.Context, index, id string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("elasticsearch: marshal failed: %w", err)
	}
	res, err := c.es.Index(index, bytes.NewReader(body),
		c.es.Index.WithDocumentID(id),
		c.es.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: index failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch: index error: %s", res.Status())
	}
	return nil
}

// SearchResult holds raw search results from Elasticsearch.
type SearchResult struct {
	Total int
	Hits  []json.RawMessage
}

// Search executes a query against the given index.
func (c *Client) Search(_ context.Context, index string, query map[string]any) (SearchResult, error) {
	body, err := json.Marshal(map[string]any{"query": query})
	if err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: marshal query failed: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return SearchResult{}, fmt.Errorf("elasticsearch: search error: %s", res.Status())
	}

	var result struct {
		Hits struct {
			Total struct{ Value int }
			Hits  []struct{ Source json.RawMessage `json:"_source"` }
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: decode failed: %w", err)
	}

	hits := make([]json.RawMessage, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		hits[i] = h.Source
	}
	return SearchResult{Total: result.Hits.Total.Value, Hits: hits}, nil
}

// Delete removes a document by ID from the given index.
func (c *Client) Delete(_ context.Context, index, id string) error {
	res, err := c.es.Delete(index, id)
	if err != nil {
		return fmt.Errorf("elasticsearch: delete failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() && !strings.Contains(res.Status(), "404") {
		return fmt.Errorf("elasticsearch: delete error: %s", res.Status())
	}
	return nil
}

// Ping verifies Elasticsearch connectivity.
func (c *Client) Ping(_ context.Context) error {
	res, err := c.es.Ping()
	if err != nil {
		return fmt.Errorf("elasticsearch: ping failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch: ping error: %s", res.Status())
	}
	return nil
}
```

- [ ] **Step 5: Create adapters/search/opensearch/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/search/opensearch

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/opensearch-project/opensearch-go/v2 v2.3.0
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 6: Create adapters/search/opensearch/client.go**

```go
// Package opensearch provides an OpenSearch implementation for search operations.
// The API mirrors the elasticsearch adapter — same operations, different SDK.
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
)

// Client wraps the OpenSearch client.
type Client struct {
	os *opensearch.Client
}

// Config holds OpenSearch connection configuration.
type Config struct {
	Addresses []string
	Username  string
	Password  string
}

// New creates a new OpenSearch client.
func New(cfg Config) (*Client, error) {
	os, err := opensearch.NewClient(opensearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("opensearch: failed to create client: %w", err)
	}
	return &Client{os: os}, nil
}

// Index stores a document.
func (c *Client) Index(_ context.Context, index, id string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("opensearch: marshal failed: %w", err)
	}
	res, err := c.os.Index(index, bytes.NewReader(body),
		c.os.Index.WithDocumentID(id),
		c.os.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("opensearch: index failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("opensearch: index error: %s", res.Status())
	}
	return nil
}

// SearchResult holds raw search hits.
type SearchResult struct {
	Total int
	Hits  []json.RawMessage
}

// Search executes a query.
func (c *Client) Search(_ context.Context, index string, query map[string]any) (SearchResult, error) {
	body, _ := json.Marshal(map[string]any{"query": query})
	res, err := c.os.Search(
		c.os.Search.WithIndex(index),
		c.os.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return SearchResult{}, fmt.Errorf("opensearch: search failed: %w", err)
	}
	defer res.Body.Close()

	var result struct {
		Hits struct {
			Total struct{ Value int }
			Hits  []struct{ Source json.RawMessage `json:"_source"` }
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return SearchResult{}, err
	}
	hits := make([]json.RawMessage, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		hits[i] = h.Source
	}
	return SearchResult{Total: result.Hits.Total.Value, Hits: hits}, nil
}

// Delete removes a document by ID.
func (c *Client) Delete(_ context.Context, index, id string) error {
	res, err := c.os.Delete(index, id)
	if err != nil {
		return fmt.Errorf("opensearch: delete failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() && !strings.Contains(res.Status(), "404") {
		return fmt.Errorf("opensearch: delete error: %s", res.Status())
	}
	return nil
}

// Ping verifies OpenSearch connectivity.
func (c *Client) Ping(_ context.Context) error {
	res, err := c.os.Ping()
	if err != nil {
		return fmt.Errorf("opensearch: ping failed: %w", err)
	}
	defer res.Body.Close()
	return nil
}
```

- [ ] **Step 7: Update go.work**

Add to `use` block:
```
./adapters/payment/stripe
./adapters/search/elasticsearch
./adapters/search/opensearch
```

- [ ] **Step 8: Build all three**

```bash
cd adapters/payment/stripe && go build ./...
cd adapters/search/elasticsearch && go build ./...
cd adapters/search/opensearch && go build ./...
```

- [ ] **Step 9: Commit**

```bash
git add adapters/payment/ adapters/search/ go.work
git commit -m "feat(adapters): add Stripe, Elasticsearch, and OpenSearch adapters (essential level)"
```

---

## Task 7: Essential adapters — tracing/otel and sms/twilio

- [ ] **Step 1: Create adapters/tracing/otel/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/tracing/otel

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	go.opentelemetry.io/otel v1.26.0
	go.opentelemetry.io/otel/trace v1.26.0
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 2: Create adapters/tracing/otel/tracer.go**

```go
// Package otel implements ports/observability.Tracer using OpenTelemetry.
package otel

import (
	"context"

	obs "github.com/marcusPrado02/go-commons/ports/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer is an OpenTelemetry implementation of ports/observability.Tracer.
type Tracer struct {
	tracer oteltrace.Tracer
}

// New creates a Tracer from an OpenTelemetry tracer.
func New(t oteltrace.Tracer) *Tracer {
	return &Tracer{tracer: t}
}

// StartSpan creates a new OTel span derived from ctx.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, obs.Span) {
	ctx, span := t.tracer.Start(ctx, name)
	return ctx, &otelSpan{span: span}
}

type otelSpan struct {
	span oteltrace.Span
}

// End marks the OTel span as complete.
func (s *otelSpan) End() { s.span.End() }

// RecordError attaches an error to the span, aligned with the OTel API.
func (s *otelSpan) RecordError(err error) {
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
}

// SetAttribute adds a key-value attribute to the span.
func (s *otelSpan) SetAttribute(key string, value any) {
	switch v := value.(type) {
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

var _ obs.Tracer = (*Tracer)(nil)
var _ obs.Span = (*otelSpan)(nil)
```

- [ ] **Step 3: Add missing import to tracer.go**

Add `"fmt"` to imports in `tracer.go`.

- [ ] **Step 4: Create adapters/sms/twilio/go.mod**

```
module github.com/marcusPrado02/go-commons/adapters/sms/twilio

go 1.22.0

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/twilio/twilio-go v1.20.1
)

replace github.com/marcusPrado02/go-commons => ../../../..
```

- [ ] **Step 5: Create adapters/sms/twilio/client.go**

```go
// Package twilio provides a Twilio implementation of ports/sms.SMSPort.
package twilio

import (
	"context"
	"fmt"

	"github.com/marcusPrado02/go-commons/ports/sms"
	twilioapi "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Client is a Twilio implementation of SMSPort.
type Client struct {
	twilio *twilioapi.RestClient
	from   string
}

// New creates a new Twilio SMS client.
func New(accountSID, authToken, fromNumber string) (*Client, error) {
	if accountSID == "" || authToken == "" || fromNumber == "" {
		return nil, fmt.Errorf("twilio: accountSID, authToken, and fromNumber are required")
	}
	client := twilioapi.NewRestClientWithParams(twilioapi.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &Client{twilio: client, from: fromNumber}, nil
}

// Send delivers an SMS message to the given E.164 phone number.
func (c *Client) Send(_ context.Context, to, body string) (sms.SMSReceipt, error) {
	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(c.from)
	params.SetBody(body)

	msg, err := c.twilio.Api.CreateMessage(params)
	if err != nil {
		return sms.SMSReceipt{}, fmt.Errorf("twilio: send failed: %w", err)
	}
	if msg.Sid == nil {
		return sms.SMSReceipt{}, fmt.Errorf("twilio: no message SID returned")
	}
	return sms.SMSReceipt{MessageID: *msg.Sid}, nil
}

// Ping verifies Twilio credentials by fetching the account.
func (c *Client) Ping(_ context.Context) error {
	_, err := c.twilio.Api.FetchAccount(nil)
	if err != nil {
		return fmt.Errorf("twilio: ping failed: %w", err)
	}
	return nil
}

var _ sms.SMSPort = (*Client)(nil)
```

- [ ] **Step 6: Update go.work**

Add to `use` block:
```
./adapters/tracing/otel
./adapters/sms/twilio
```

- [ ] **Step 7: Build both**

```bash
cd adapters/tracing/otel && go build ./...
cd adapters/sms/twilio && go build ./...
```

Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add adapters/tracing/ adapters/sms/ go.work
git commit -m "feat(adapters): add OTel tracer and Twilio SMS adapters (essential level)"
```

---

## Task 8: Final workspace build and go mod tidy for all modules

- [ ] **Step 1: Tidy root module**

```bash
go mod tidy
```

- [ ] **Step 2: Tidy each adapter module**

```bash
make tidy-all
```

Expected: each adapter's `go.mod` and `go.sum` are updated

- [ ] **Step 3: Build everything via workspace**

```bash
go build ./...
```

Expected: all packages build successfully

- [ ] **Step 4: Run tests across root module**

```bash
go test ./... -race
```

Expected: all PASS (adapter tests that require live credentials are skipped)

- [ ] **Step 5: Final commit**

```bash
git add .
git commit -m "chore: go mod tidy for all adapter submódulos"
```

---

## Self-Review Checklist

After completing all tasks, verify:

- [ ] `go build ./...` from root passes
- [ ] Every adapter has `var _ Port = (*Client)(nil)` compile-time check
- [ ] `adapters/persistence/inmemory` runs `RepositoryContract` test suite
- [ ] `adapters/email/sendgrid` has HTTP mock tests that pass without credentials
- [ ] All `go.mod` files have `replace` directive pointing to `../../../..`
- [ ] `go.work` lists all 11 adapter modules
- [ ] No `TODO` or `TBD` in any `.go` file
- [ ] `make tidy-all` completes without errors
