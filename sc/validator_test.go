package sc_test

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supercargo-dev/supercargo-sdk-go/sc"
	"github.com/supercargo-dev/supercargo-sdk-go/supercargotest"
)

type TestUser struct {
	_              struct{}              `supercargo.contract:"urn=urn:supercargo:hub:contract:test_user,version=1.0.0"`
	Name           string                `json:"name" supercargo.field:"not_empty,max_length=50"`
	Email          *string               `json:"email" supercargo.field:"pii=true,context_id=user,rank=1,not_empty,pattern=^[^@]+@[^@]+\\.[^@]+$"`
	Age            string                `json:"age" supercargo.field:"greater_than=18,less_than=99"`
	Tags           []string              `json:"tags" supercargo.field:"not_empty,max_length=5"`
	Department     string                `json:"department"`
	Score          float64               `json:"score" supercargo.field:"greater_than=0,less_than=100"`
	LoginCount     uint                  `json:"loginCount" supercargo.field:"greater_than=0"`
	HugeID         int64                 `json:"hugeId" supercargo.field:"greater_than=9007199254740992"`
	*RecursiveNode `supercargo.field:""` // Ensure nil traversal doesn't panic
	Addresses      []Address             `json:"addresses"`
}

type Address struct {
	City string `json:"city" supercargo.field:"not_empty"`
}

type RecursiveNode struct {
	*RecursiveNode
	NodeVal int
}

// Branch tests
type GraphRoot struct {
	Left  BranchNode
	Right BranchNode
}

type BranchNode struct {
	Leaf LeafNode
}

type LeafNode struct {
	Value string `supercargo.field:"not_empty"`
}

func (u TestUser) ContractConstraints() []sc.Rule {
	return []sc.Rule{} // Test purely struct tags
}

func TestValidation_Success(t *testing.T) {
	email := "alice@example.com"
	validUser := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "30",
		Tags:       []string{"admin", "engineering"},
		Department: "Engineering",
		Score:      95.5,
		LoginCount: 5,
		HugeID:     9007199254740993, // strictly greater than 2^53
		Addresses: []Address{
			{City: "New York"},
		},
	}

	supercargotest.AssertValidates(t, validUser)
	supercargotest.AssertValidates(t, &validUser)
}

func TestValidation_Failure_NotEmpty(t *testing.T) {
	invalidUser := TestUser{
		Name:       "",  // should fail NotEmpty
		Email:      nil, // should fail NotEmpty on pointer
		Age:        "30",
		Tags:       []string{"admin"},
		Department: "Engineering",
		Score:      50,
		LoginCount: 5,
		HugeID:     9007199254740995,
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"Name\": must not be empty")
}

func TestValidation_Failure_Pattern_Pointer(t *testing.T) {
	email := "not-an-email"
	invalidUser := TestUser{
		Name:       "Alice",
		Email:      &email, // should fail Pattern (dereferencing)
		Age:        "30",
		Tags:       []string{"admin"},
		Score:      50,
		LoginCount: 5,
		HugeID:     9007199254740995,
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"Email\": does not match pattern (got: [REDACTED PII])")
}

func TestValidation_Failure_GreaterLess_StringNumeric(t *testing.T) {
	email := "alice@example.com"
	invalidUser := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "2", // Lexicographically "2" > "18", but numerically 2 < 18, so should fail GreaterThan
		Tags:       []string{"admin"},
		Score:      50,
		LoginCount: 5,
		HugeID:     9007199254740995,
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"Age\": must be strictly greater than 18")
}

func TestValidation_Failure_GreaterLess_StringParseFail(t *testing.T) {
	email := "alice@example.com"
	invalidUser := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "not_a_number", // Fails ParseFloat and should trigger validation error
		Tags:       []string{"admin"},
		Score:      50,
		LoginCount: 5,
		HugeID:     9007199254740995,
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"Age\": failed to parse as numeric")
}

func TestValidation_Failure_GreaterLess_Float64(t *testing.T) {
	email := "alice@example.com"
	invalidUser := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "30",
		Tags:       []string{"admin"},
		Score:      105.0, // should fail LessThan 100
		LoginCount: 5,
		HugeID:     9007199254740995,
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"Score\": must be strictly less than 100")
}

func TestValidation_Failure_Uint(t *testing.T) {
	email := "alice@example.com"
	invalidUser := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "30",
		Tags:       []string{"admin"},
		Score:      50.0,
		LoginCount: 0, // should fail GreaterThan 0
		HugeID:     9007199254740995,
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"LoginCount\": must be strictly greater than 0")
}

func TestValidation_Failure_Int64_Precision(t *testing.T) {
	email := "alice@example.com"
	invalidUser := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "30",
		Tags:       []string{"admin"},
		Score:      50.0,
		LoginCount: 10,
		HugeID:     9007199254740992, // should fail GreaterThan 9007199254740992 since it's equal
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"HugeID\": must be strictly greater than 9007199254740992")
}

func TestValidation_Failure_SliceRecursion(t *testing.T) {
	email := "alice@example.com"
	invalidUser := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "30",
		Tags:       []string{"admin"},
		Score:      50.0,
		LoginCount: 10,
		HugeID:     9007199254740995,
		Addresses: []Address{
			{City: ""}, // Should fail NotEmpty in slice
		},
	}

	supercargotest.AssertFailsValidation(t, invalidUser, "validation failed on field \"[0]\": element validation failed (validation failed on field \"City\": must not be empty)")
}

func TestValidation_NilPointer(t *testing.T) {
	var u *TestUser = nil
	// Should not panic, should return nil since there's no data to validate
	supercargotest.AssertValidates(t, u)
}

func TestValidation_GraphTraversal(t *testing.T) {
	invalidGraph := GraphRoot{
		Left:  BranchNode{Leaf: LeafNode{Value: "OK"}},
		Right: BranchNode{Leaf: LeafNode{Value: ""}}, // Should fail since it's in the right branch
	}
	supercargotest.AssertFailsValidation(t, invalidGraph, "validation failed on field \"Value\": must not be empty")
}

func TestValidation_CycleDetection(t *testing.T) {
	node := &RecursiveNode{NodeVal: 1}
	node.RecursiveNode = node // create infinite cycle

	email := "alice@example.com"
	user := TestUser{
		Name:          "Alice",
		Email:         &email,
		Age:           "30",
		Tags:          []string{"admin", "engineering"},
		Department:    "Engineering",
		Score:         99.9,
		LoginCount:    5,
		HugeID:        9007199254740995,
		RecursiveNode: node, // The validator must not stack overflow
	}

	// Should successfully terminate without panicking, and pass validation
	// because there are no invalid constraints in this valid object.
	supercargotest.AssertValidates(t, user)
}

type Phase6TestUser struct {
	// Sibling pointers inside an embedded struct
	*Phase6Node `supercargo.field:""`
	// Optional field falling through to float parse
	OptionalAge string `json:"optionalAge" supercargo.field:"greater_than=18"`
	// Regex with comma
	RegexWithComma string `json:"regex" supercargo.field:"pattern='^[a-z,A-Z]+$'"`
	// PII Leakage
	SecretPII string `json:"secret" supercargo.field:"pii=true,not_empty,pattern=^[a-z]+$"`
	// Large precision
	HugeID string `json:"hugeId" supercargo.field:"greater_than=9007199254740995"`
}

type Phase6Node struct {
	SiblingA string `supercargo.field:"not_empty"`
	SiblingB string `supercargo.field:"not_empty"`
}

func TestValidation_Failure_RegexComma(t *testing.T) {
	node := &Phase6Node{SiblingA: "a", SiblingB: "b"}
	user := Phase6TestUser{
		Phase6Node:     node,
		OptionalAge:    "30",
		RegexWithComma: "123", // fails regex because it doesn't match a-z,A-Z
		SecretPII:      "abc",
		HugeID:         "9007199254740996",
	}
	supercargotest.AssertFailsValidation(t, user, "validation failed on field \"RegexWithComma\": does not match pattern")
}

func TestValidation_Failure_PII_Redaction(t *testing.T) {
	node := &Phase6Node{SiblingA: "a", SiblingB: "b"}
	user := Phase6TestUser{
		Phase6Node:     node,
		OptionalAge:    "30",
		RegexWithComma: "a,b",
		SecretPII:      "123", // fails regex
		HugeID:         "9007199254740996",
	}
	// The value "123" must not be present in the error string!
	supercargotest.AssertFailsValidation(t, user, "[REDACTED PII]")
}

func TestValidation_Optional_Empty_ShortCircuit(t *testing.T) {
	node := &Phase6Node{SiblingA: "a", SiblingB: "b"}
	user := Phase6TestUser{
		Phase6Node:     node,
		OptionalAge:    "", // Optional string should short-circuit and NOT fail float parsing
		RegexWithComma: "a,B",
		SecretPII:      "abc",
		HugeID:         "9007199254740996",
	}
	supercargotest.AssertValidates(t, user)
}

func TestValidation_SiblingPointers(t *testing.T) {
	node := &Phase6Node{
		SiblingA: "a",
		SiblingB: "", // Should fail! Previous code would mask this due to shared cycle detection map.
	}
	user := Phase6TestUser{
		Phase6Node:     node,
		OptionalAge:    "30",
		RegexWithComma: "a,B",
		SecretPII:      "abc",
		HugeID:         "9007199254740996",
	}
	supercargotest.AssertFailsValidation(t, user, "validation failed on field \"SiblingB\": must not be empty")
}

func TestValidation_PrecisionLoss(t *testing.T) {
	node := &Phase6Node{SiblingA: "a", SiblingB: "b"}
	user := Phase6TestUser{
		Phase6Node:     node,
		OptionalAge:    "30",
		RegexWithComma: "a,B",
		SecretPII:      "abc",
		HugeID:         "9007199254740995", // fails because it must be strictly GREATER THAN 9007199254740995 (equal fails)
	}
	// Previous code would lose precision and parse 9007199254740995 as 9007199254740996 float64, making this pass (false negative).
	supercargotest.AssertFailsValidation(t, user, "validation failed on field \"HugeID\": must be strictly greater than 9007199254740995")
}

func BenchmarkValidation_Overhead(b *testing.B) {
	email := "alice@example.com"
	user := TestUser{
		Name:       "Alice",
		Email:      &email,
		Age:        "30",
		Tags:       []string{"admin", "engineering"},
		Department: "Engineering",
		Score:      99.9,
		LoginCount: 1,
		HugeID:     math.MaxInt64,
	}

	b.Run("Unvalidated_JSON_Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(user)
		}
	})

	b.Run("Validated_And_JSON_Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = sc.Validate(user)
			_, _ = json.Marshal(user)
		}
	})

	b.Run("Pure_Validation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = sc.Validate(user)
		}
	})
}

func TestValidation_ContractSchemaExtraction(t *testing.T) {
	v, err := sc.GetValidator(reflect.TypeOf(TestUser{}))
	if err != nil {
		t.Fatalf("unexpected error getting validator: %v", err)
	}

	schema := v.Schema()
	if schema.Contract.URN != "urn:supercargo:hub:contract:test_user" {
		t.Errorf("expected URN 'urn:supercargo:hub:contract:test_user', got %q", schema.Contract.URN)
	}
	if schema.Contract.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got %q", schema.Contract.Version)
	}

	if fieldMeta, ok := schema.Fields["Email"]; !ok {
		t.Errorf("expected metadata for field Email")
	} else {
		if fieldMeta.IsPII == nil || !*fieldMeta.IsPII {
			t.Errorf("expected Email field to be PII")
		}
		if fieldMeta.ContextID != "user" {
			t.Errorf("expected Email ContextID 'user', got %q", fieldMeta.ContextID)
		}
	}
}

func TestValidation_PII_Unwrap_Redaction(t *testing.T) {
	underlying := fmt.Errorf("sensitive raw value: secret123")
	valErr := &sc.ValidationError{
		Field:  "Email",
		Reason: "invalid email format",
		Value:  "secret123",
		Err:    underlying,
		IsPII:  true,
	}

	unwrapped := valErr.Unwrap()
	if unwrapped == nil {
		t.Fatalf("expected unwrapped error to not be nil")
	}

	if unwrapped.Error() != "underlying error contains redacted PII" {
		t.Errorf("expected unwrapped error string 'underlying error contains redacted PII', got %q", unwrapped.Error())
	}

	if strings.Contains(unwrapped.Error(), "secret123") {
		t.Errorf("PII leaked in Unwrap(): %s", unwrapped.Error())
	}
}

type ThunderingHerdUser struct {
	Name string `supercargo.field:"not_empty"`
}

var thunderingHerdCallCount int64

func (u ThunderingHerdUser) ContractConstraints() []sc.Rule {
	atomic.AddInt64(&thunderingHerdCallCount, 1)
	time.Sleep(10 * time.Millisecond) // Simulate compilation work
	return []sc.Rule{}
}

func TestValidation_ThunderingHerd(t *testing.T) {
	atomic.StoreInt64(&thunderingHerdCallCount, 0)
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u := ThunderingHerdUser{Name: "Alice"}
			if err := sc.Validate(u); err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		}()
	}
	wg.Wait()

	if count := atomic.LoadInt64(&thunderingHerdCallCount); count != 1 {
		t.Errorf("expected ContractConstraints to be called exactly 1 time, got %d", count)
	}
}

type BadSchemaUser struct {
	Email string `supercargo.field:"pattern=["` // Invalid regex syntax
}

func TestValidation_BadSchemaCache(t *testing.T) {
	user := BadSchemaUser{Email: "test@example.com"}
	err1 := sc.Validate(user)
	if err1 == nil {
		t.Fatalf("expected validation error for bad schema regex, got nil")
	}

	// Second call must return cached error instantly
	err2 := sc.Validate(user)
	if err2 == nil {
		t.Fatalf("expected cached validation error on second call, got nil")
	}
	if err1.Error() != err2.Error() {
		t.Errorf("expected identical error messages, got %q vs %q", err1.Error(), err2.Error())
	}
}

type PIIValUser struct {
	SecretData string `supercargo.field:"pii=true,pattern=^[0-9]+$"`
}

func TestValidation_PII_Value_Redaction(t *testing.T) {
	user := PIIValUser{SecretData: "sensitive_user_input"}
	err := sc.Validate(user)
	if err == nil {
		t.Fatalf("expected validation failure, got nil")
	}

	valErr, ok := err.(*sc.ValidationError)
	if !ok {
		t.Fatalf("expected *sc.ValidationError, got %T", err)
	}
	if !valErr.IsPII {
		t.Errorf("expected IsPII to be true")
	}
	if valErr.Value != nil {
		t.Errorf("expected ValidationError.Value to be nil for PII field, got %v", valErr.Value)
	}
}
