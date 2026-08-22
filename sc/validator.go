package sc

import (
	"fmt"
	"reflect"
)

// Validator holds the precompiled rules for a struct type.
type Validator struct {
	typ      reflect.Type
	checks   []func(reflect.Value, *ptrStack) error
	metadata map[string]FieldMetadata
	contract ContractMetadata
}

// Contract returns the extracted contract-level metadata.
func (v *Validator) Contract() ContractMetadata {
	return v.contract
}

// Schema returns the collected Schema combining contract metadata and field metadata.
func (v *Validator) Schema() Schema {
	return Schema{
		Contract: v.contract,
		Fields:   v.Metadata(),
	}
}

// GetValidator returns the compiled Validator for the given struct type or pointer to struct.
func GetValidator(typ reflect.Type) (*Validator, error) {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("supercargo: type %s is not a struct", typ.Name())
	}
	return getValidator(typ)
}

// Metadata returns the collected metadata for fields as a copy to prevent cache poisoning.
func (v *Validator) Metadata() map[string]FieldMetadata {
	cloned := make(map[string]FieldMetadata, len(v.metadata))
	for k, m := range v.metadata {
		c := m
		if m.IsPII != nil {
			b := *m.IsPII
			c.IsPII = &b
		}
		if m.IdentifierRank != nil {
			i := *m.IdentifierRank
			c.IdentifierRank = &i
		}
		if m.IsPrimaryKey != nil {
			pk := *m.IsPrimaryKey
			c.IsPrimaryKey = &pk
		}
		if m.SortRank != nil {
			sr := *m.SortRank
			c.SortRank = &sr
		}
		cloned[k] = c
	}
	return cloned
}

// Validate payload against its defined ContractConstraints and struct tags.
func Validate(payload any) error {
	if payload == nil {
		return nil
	}

	val := reflect.ValueOf(payload)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil // We only validate structs
	}

	v, err := getValidator(val.Type())
	if err != nil {
		return err
	}

	return v.check(val, nil)
}

func getValidator(typ reflect.Type) (*Validator, error) {
	return getValidatorInternal(typ, make(map[reflect.Type]*Validator))
}

func getValidatorInternal(typ reflect.Type, compiling map[reflect.Type]*Validator) (*Validator, error) {
	// 1. Recursive lookup: returns the partially built Validator at compile time
	if v, ok := compiling[typ]; ok {
		return v, nil
	}

	// 2. Fast path for globally cached types
	if futInter, ok := globalValidatorCache.Load(typ); ok {
		fut := futInter.(*validatorFuture)
		<-fut.done
		return fut.v, fut.err
	}

	futNew := &validatorFuture{
		done: make(chan struct{}),
	}
	futInter, loaded := globalValidatorCache.LoadOrStore(typ, futNew)
	fut := futInter.(*validatorFuture)

	if loaded {
		<-fut.done
		return fut.v, fut.err
	}

	fut.once.Do(func() {
		defer close(fut.done)

		// Execute ContractConstraints safely inside sync.Once lock
		var progRules []Rule
		var initErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					initErr = fmt.Errorf("supercargo: panic while executing ContractConstraints on zero-value struct %s (ensure method is safe on nil/zero values): %v", typ.Name(), r)
				}
			}()

			var cv ContractValidator
			ptrToZero := reflect.New(typ)
			if cvZero, ok := ptrToZero.Interface().(ContractValidator); ok {
				cv = cvZero
			} else if cvZero, ok := ptrToZero.Elem().Interface().(ContractValidator); ok {
				cv = cvZero
			}
			if cv != nil {
				progRules = cv.ContractConstraints()
			}
		}()

		if initErr != nil {
			fut.err = initErr
			fut.v = nil
			return
		}

		// Compile rules
		compiledV, buildErr := buildValidator(typ, progRules, compiling)
		if buildErr != nil {
			fut.err = buildErr
			fut.v = nil
			return
		}

		fut.v = compiledV
	})

	return fut.v, fut.err
}

func (v *Validator) check(val reflect.Value, visited *ptrStack) error {
	for _, chk := range v.checks {
		if err := chk(val, visited); err != nil {
			return err
		}
	}
	return nil
}

func newValidationError(field, reason string, val any, isPII bool, err error) *ValidationError {
	if isPII {
		val = nil
	}
	return &ValidationError{
		Field:  field,
		Reason: reason,
		Value:  val,
		Err:    err,
		IsPII:  isPII,
	}
}
