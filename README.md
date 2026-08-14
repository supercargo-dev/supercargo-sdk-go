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

	// Federated entity anchor with UUID type hint and PII metadata
	UserID string `json:"userId" supercargo.field:"as=UUID,entity=urn:supercargo:entity:identity-team:user,pii=true,context_id=user_salt,identity_domain=urn:supercargo:identity_domain:user,rank=1"`

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
