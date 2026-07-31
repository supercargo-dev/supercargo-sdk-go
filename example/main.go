package main

import (
	"fmt"
	"log"

	"github.com/supercargo-dev/supercargo-sdk-go/sc"
)

// Order represents a fully-annotated DataContract natively in Go.
// The dummy `_` field carries the contract-level metadata.
type Order struct {
	_ struct{} `supercargo.contract:"urn=urn:supercargo:hub:contract:order,version=1.0.0,owner_team=team-sales"`

	BuyerEmail string `supercargo.field:"type=STRING,pii=true,context_id=buyer,identity_domain=urn:supercargo:hub:identity_domain:customer,rank=1"`
	BuyerPhone string `supercargo.field:"type=STRING,pii=true,context_id=buyer,rank=2"`
	SellerID   string `supercargo.field:"type=UUID,pii=true,context_id=seller,identity_domain=urn:supercargo:hub:identity_domain:employee,rank=1"`

	// Notice we can omit struct tags for fields if we prefer using the Builder in ContractConstraints()
	OrderTotal float64
}

// ContractConstraints implements sc.ContractValidator.
// This is AOT-compiled by the SDK on startup to provide < 5% overhead during runtime serialization.
func (o Order) ContractConstraints() []sc.Rule {
	return []sc.Rule{
		sc.Field("BuyerEmail").Pattern(`^[^@]+@[^@]+\.[^@]+$`).MaxLength(50),
		sc.Field("BuyerPhone").Pattern(`^\+?[1-9]\d{1,14}$`),
	}
}

func main() {
	order := Order{
		BuyerEmail: "customer@example.com",
		BuyerPhone: "+15551234567",
		SellerID:   "a3b8d4f2-9c1e-4b6a-8d7e-2f3a1b4c5d6e",
		OrderTotal: 99.99,
	}

	// Validate the contract payload before serialization/publishing.
	// In a real high-throughput pipeline, the reflection lookup is cached, and validation
	// happens incredibly fast using pre-compiled execution plans.
	if err := sc.Validate(order); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Println("DataContract validated successfully!")
}
