# Supercargo SDK for Go

The official Go SDK for [Supercargo](https://supercargo.dev), providing zero-dependency native struct tag annotations to define type-safe Data Contracts, governance metadata, and validation constraints directly in Go models.

## Installation

To install the SDK constants, use `go get`:

```bash
go get github.com/supercargo-dev/supercargo-sdk-go
```

> **Note:** Supercargo's AST parser reads standard Go struct tags statically, meaning importing the SDK is optional and introduces zero runtime overhead.

---

## Defining Data Contracts

Declare contract-level metadata on a blank struct field (`_ struct{}`) with the `supercargo.contract` tag, and field-level metadata with the `supercargo.field` tag.

### Example

```go
package domain

import "github.com/supercargo-dev/supercargo-sdk-go"

type UserSignup struct {
	// Contract-level metadata
	_ struct{} `supercargo.contract:"urn=urn:supercargo:contract:user_signup:v1,version=1.0.0,owner_team=identity-team,data_asset=users_v1,validation_policy=STRICT"`

	// Primary key and entity anchor - PII, salt context, and identity domain are auto-hydrated from the central entity registry
	UserID string `json:"userId" supercargo.field:"primary_key,entity=user,as=UUID"`

	// Validated string with regex pattern and minimum length
	Email string `json:"email" supercargo.field:"pii=true,pattern='^\\S+@\\S+$',min_length=1,not_empty=true"`

	// Numerical constraint
	Age int `json:"age" supercargo.field:"greater_than=18,less_than=120"`

	// Standard boolean field
	IsActive bool `json:"isActive"`
}
```

---

## Annotation Tag Reference

### Contract Struct Tags (`supercargo.contract`)

Place on `_ struct{}` at the top of your model:

| Key | Description | Example |
| :--- | :--- | :--- |
| `urn` | Canonical URN for the contract. | `urn=urn:supercargo:contract:orders:v1` |
| `version` | Semantic version string. | `version=1.0.0` |
| `owner_team` / `ownerTeam` | Owning team name. | `owner_team=payments-team` |
| `data_asset` / `dataAsset` | Associated data asset or topic name. | `data_asset=order_events` |
| `validation_policy` | Validation policy enum (`STRICT`, `LENIENT`, `MUTATE`). | `validation_policy=STRICT` |

### Field Struct Tags (`supercargo.field`)

Attach to individual struct fields:

| Key | Description | Example |
| :--- | :--- | :--- |
| `as` | Semantic data type hint (`UUID`, `TIMESTAMP`, `EMAIL`, etc.). | `as=UUID` |
| `pii` | Marks field as containing PII (`true`, `false`, or category name). | `pii=true` |
| `context_id` | Salt / hashing context for pseudonymization. | `context_id=user_salt` |
| `identity_domain` | Identity Domain URN for cross-system joining. | `identity_domain=urn:supercargo:identity_domain:user` |
| `rank` | Identity domain priority rank (1 = Primary). | `rank=1` |
| `entity` / `entity_ref` | Entity reference URN for federated identity anchoring. | `entity=urn:supercargo:entity:identity:user` |
| `not_empty` | Enforces non-empty constraint and marks field as `REQUIRED`. | `not_empty=true` |
| `min_length` | Minimum string or collection length. | `min_length=1` |
| `max_length` | Maximum string or collection length. | `max_length=255` |
| `pattern` | Regular expression pattern constraint. | `pattern='^\\S+@\\S+$'` |
| `greater_than` | Numerical lower bound (exclusive). | `greater_than=0` |
| `greater_than_or_equal`| Numerical lower bound (inclusive). | `greater_than_or_equal=18` |
| `less_than` | Numerical upper bound (exclusive). | `less_than=100` |
| `less_than_or_equal` | Numerical upper bound (inclusive). | `less_than_or_equal=99` |

---

## Scaffolding & Zero-Dependency Code Generation

To scaffold annotations directly into your project without any external module dependencies:

```bash
sc init annotations --lang go
```

This generates `supercargo_sdk/` helper constants in your current project.

---

## Hub Client & Health Governance

The Supercargo SDK for Go provides `clients.HubClient` (in package `github.com/supercargo-dev/supercargo-sdk-go/clients`) for interacting with the Supercargo Hub Service over gRPC. It enables Go microservices, ingestion daemons, and streaming processors to fetch Data Contracts, report operational health state anomalies, and evaluate downstream blast radius.

### Client Initialization & Configuration

Instantiate a client using `clients.NewHubClient(target, ...opts)`:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/supercargo-dev/supercargo-sdk-go/clients"
)

func main() {
	// Configure client with authentication, custom timeouts, and retry policies
	client, err := clients.NewHubClient(
		"hub.internal:50051",
		clients.WithToken("supercargo-service-token"),
		clients.WithTimeout(5*time.Second),
		clients.WithMaxRetries(3),
		clients.WithRetryDelay(500*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("failed to initialize hub client: %v", err)
	}
	defer client.Close()
}
```

#### Functional Options

| Option | Description |
| :--- | :--- |
| `clients.WithToken(token string)` | Sets static Bearer authorization token header. |
| `clients.WithTokenProvider(func(ctx context.Context) (string, error))` | Dynamic token provider for GCP Cloud Run IAM OIDC or OAuth2 token refreshes. |
| `clients.WithTimeout(d time.Duration)` | Per-RPC call deadline (default: `10s`). |
| `clients.WithMaxRetries(n int)` | Maximum retries on transient errors (`UNAVAILABLE`, `RESOURCE_EXHAUSTED`, `DEADLINE_EXCEEDED`). |
| `clients.WithRetryDelay(d time.Duration)` | Initial jittered exponential backoff delay (default: `500ms`). |
| `clients.WithInsecure()` | Configures plaintext gRPC transport credentials (for local development / testing). |
| `clients.WithTransportCredentials(creds credentials.TransportCredentials)` | Injects custom TLS transport credentials. |
| `clients.WithGRPCConn(conn *grpc.ClientConn)` | Reuses an existing `*grpc.ClientConn`. |

---

### Production Deployment on Google Cloud Run (IAM OIDC Authentication)

In Google Cloud environments (Cloud Run, GKE with Workload Identity, Compute Engine, or Cloud Composer), connecting to a production Hub service running on Cloud Run requires a Google-signed OpenID Connect (OIDC) ID token with the Cloud Run service URL as audience (`aud`). The caller's Service Account requires `roles/run.invoker` on the Hub service.

Use the official Google `idtoken` package (`google.golang.org/api/idtoken`) with `clients.WithTokenProvider` to automatically handle token acquisition, local caching, and refresh before the 1-hour expiration:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/supercargo-dev/supercargo-sdk-go/clients"
	hubv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/hub/v1"
	"google.golang.org/api/idtoken"
)

func main() {
	ctx := context.Background()

	// 1. Target Cloud Run service URL (must match the Cloud Run service audience)
	serviceURL := "https://hub-control-plane-xyz-ew.a.run.app"

	// 2. idtoken.NewTokenSource automatically obtains OIDC ID tokens via GCP metadata server
	// (GKE Workload Identity, Cloud Run, Compute Engine) or local ADC, caching and refreshing them.
	tokenSource, err := idtoken.NewTokenSource(ctx, serviceURL)
	if err != nil {
		log.Fatalf("failed to create Google Cloud IAM OIDC token source: %v", err)
	}

	// 3. Initialize HubClient with dynamic WithTokenProvider on standard port 443 with TLS
	client, err := clients.NewHubClient(
		"hub-control-plane-xyz-ew.a.run.app:443",
		clients.WithTokenProvider(func(ctx context.Context) (string, error) {
			token, err := tokenSource.Token()
			if err != nil {
				return "", err
			}
			return token.AccessToken, nil // Raw OIDC ID token
		}),
		clients.WithTimeout(10*time.Second),
		clients.WithMaxRetries(3),
	)
	if err != nil {
		log.Fatalf("failed to initialize HubClient: %v", err)
	}
	defer client.Close()

	// 4. Report health anomaly or query blast radius
	resp, err := client.ReportAnomaly(ctx, &hubv1.ReportAnomalyRequest{
		Urn:          "urn:sc:contract:marketing:customer_churn:v1",
		State:        hubv1.HealthState_HEALTH_STATE_DEGRADED,
		IncidentType: "PIPELINE_CRASH",
		Reason:       "OOM killed during Spark feature engineering stage",
		Reporter:     "pipeline-worker-pod-42",
	})
	if err != nil {
		log.Fatalf("failed to report anomaly: %v", err)
	}
	log.Printf("Recorded anomaly transition: %s", resp.TransitionId)
}
```

---

### Reporting Health State Anomalies (`ReportAnomaly`)

Use `ReportAnomaly` to notify the Hub when a service, stream consumer, or pipeline detects an operational anomaly or completes recovery.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/supercargo-dev/supercargo-sdk-go/clients"
	hubv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/hub/v1"
)

func reportIncident(client *clients.HubClient) {
	ctx := context.Background()

	req := &hubv1.ReportAnomalyRequest{
		Urn:          "urn:sc:product:payment_gateway:v1",
		State:        hubv1.HealthState_HEALTH_STATE_DEGRADED,
		Reason:       "Payment processor downstream 504 error rate exceeded 10%",
		IncidentType: "UPSTREAM_TIMEOUT",
		RunId:        "batch-2026-09-25T14:30:00Z",
		Reporter:     "billing-processor-worker-1",
		Metadata: map[string]string{
			"error_rate": "12.4%",
			"gateway":    "stripe",
			"region":     "us-central1",
		},
	}

	resp, err := client.ReportAnomaly(ctx, req)
	if err != nil {
		log.Printf("failed to report anomaly to Hub: %v", err)
		return
	}

	fmt.Printf("Anomaly recorded successfully. Transition ID: %s\n", resp.TransitionId)
}
```

---

### Inspecting Downstream Blast Radius (`GetBlastRadius`)

Use `GetBlastRadius` to evaluate downstream dependencies and owner teams before executing breaking changes or declaring an incident.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/supercargo-dev/supercargo-sdk-go/clients"
)

func inspectBlastRadius(client *clients.HubClient) {
	ctx := context.Background()

	// Query downstream blast radius up to 3 levels deep (0 for unbounded lineage traversal)
	resp, err := client.GetBlastRadius(ctx, "urn:sc:contract:crm_orders:v1", 3)
	if err != nil {
		log.Fatalf("failed to query blast radius: %v", err)
	}

	fmt.Printf("Root Asset: %s\n", resp.RootUrn)
	fmt.Printf("Total Downstream Assets: %d\n", resp.TotalDownstream)
	fmt.Printf("Affected Teams: %v\n", resp.AffectedTeams)

	for _, node := range resp.DownstreamNodes {
		fmt.Printf("  └─ [%s] %s (depth: %d, owner: %s)\n",
			node.Type, node.Urn, node.Depth, node.Owner)
	}
}
```
