# Supercargo SDK for Go

This is the official Go SDK for [Supercargo](https://supercargo.dev), providing zero-dependency native struct tag annotations to enable type-safe metadata definition for your Supercargo contracts.

## Installation

To install the SDK, use `go get`:

```bash
go get github.com/supercargo-dev/supercargo-sdk-go
```

## Usage

Supercargo relies on standard Go struct tags to define your data contracts and hint the underlying schema types. 

By importing the `supercargo` package, you can safely use the provided constants to define your data hints without introducing heavy external dependencies to your core domain models.

### Example

```go
package domain

import "github.com/supercargo-dev/supercargo-sdk-go"

type UserEvent struct {
	// A standard UUID field, hinted to ensure downstream systems (like databases)
	// format and index it correctly.
	EventID string `supercargo:"UUID"`

	// A timestamp field.
	CreatedAt string `supercargo:"TIMESTAMP"`
	
	// A dynamic map field.
	Metadata map[string]string `supercargo:"MAP_STRING_STRING"`
}
```

## Available Data Type Hints

The SDK provides the following constants to use as struct tag values:

*   `supercargo.HintNone` (`"NONE"`): Represents no specific type hint.
*   `supercargo.HintTimestamp` (`"TIMESTAMP"`): Represents a timestamp data type.
*   `supercargo.HintMapStringString` (`"MAP_STRING_STRING"`): Represents a map of string to string.
*   `supercargo.HintUUID` (`"UUID"`): Represents a Universally Unique Identifier.

## Tag Key
The standard tag key used for struct tags is `"supercargo"` (available programmatically via `supercargo.TagKey`).

## Scaffolding

If you prefer to avoid adding external dependencies entirely, you can use the Supercargo CLI to scaffold these definitions directly into your codebase:

```bash
sc init annotations --lang go
```
This will generate the SDK files directly into a `supercargo_sdk/` directory in your current path, keeping your project free of external dependencies.
