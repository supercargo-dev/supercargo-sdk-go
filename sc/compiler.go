package sc

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

func buildValidator(typ reflect.Type, progRules []Rule, compiling map[reflect.Type]*Validator) (*Validator, error) {
	v := &Validator{
		typ:      typ,
		metadata: make(map[string]FieldMetadata),
	}
	// we must add 'v' to compiling right away so recursive calls find it
	compiling[typ] = v

	// Scan contract-level tags (e.g., supercargo.contract on anonymous _ struct{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if tag := f.Tag.Get("supercargo.contract"); tag != "" {
			parts := parseTagKeyValue(tag)
			for _, part := range parts {
				kv := strings.SplitN(part, "=", 2)
				key := strings.TrimSpace(kv[0])
				val := ""
				if len(kv) > 1 {
					val = strings.TrimSpace(kv[1])
				}
				switch strings.ToLower(key) {
				case "urn":
					v.contract.URN = val
				case "version":
					v.contract.Version = val
				case "owner_team", "owner":
					v.contract.OwnerTeam = val
				case "data_asset":
					v.contract.DataAsset = val
				case "description", "desc":
					v.contract.Description = val
				}
			}
		}
	}

	fieldsByName := make(map[string][]int)
	ruleMap := make(map[string]*FieldRuleBuilder)
	var structFieldsToRecurse [][]int

	var buildFields func(t reflect.Type, indexPrefix []int, visited []reflect.Type) error
	buildFields = func(t reflect.Type, indexPrefix []int, visited []reflect.Type) error {
		for _, vT := range visited {
			if vT == t {
				return nil // Cycle detected in this path
			}
		}

		branchVisited := make([]reflect.Type, len(visited)+1)
		copy(branchVisited, visited)
		branchVisited[len(visited)] = t

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			currentIndex := make([]int, len(indexPrefix)+1)
			copy(currentIndex, indexPrefix)
			currentIndex[len(indexPrefix)] = i

			tagRule, tagErr := parseTagRules(f)
			if tagErr != nil {
				return tagErr
			}

			// Prevent shadowing (only register the shallowest field)
			if _, exists := fieldsByName[f.Name]; !exists {
				fieldsByName[f.Name] = currentIndex
				if tagRule != nil {
					ruleMap[f.Name] = tagRule
				}
			}

			if f.Anonymous {
				if f.Type.Kind() == reflect.Struct {
					if err := buildFields(f.Type, currentIndex, branchVisited); err != nil {
						return err
					}
				} else if f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
					if err := buildFields(f.Type.Elem(), currentIndex, branchVisited); err != nil {
						return err
					}
				}
			} else {
				if containsStruct(f.Type) {
					structFieldsToRecurse = append(structFieldsToRecurse, currentIndex)
				}
			}
		}
		return nil
	}
	if err := buildFields(typ, nil, nil); err != nil {
		return nil, err
	}

	// Merge programmatic rules (override tag rules) safely using Clone()
	for _, r := range progRules {
		if fRule, ok := r.(*FieldRuleBuilder); ok && fRule != nil {
			// Ensure typed nil pointer isn't dereferenced
			if reflect.ValueOf(fRule).Kind() == reflect.Ptr && reflect.ValueOf(fRule).IsNil() {
				continue
			}
			clonedRule := fRule.Clone()
			if existing, exists := ruleMap[clonedRule.fieldName]; exists {
				existing.mustNotBeEmpty = clonedRule.mustNotBeEmpty || existing.mustNotBeEmpty
				if clonedRule.maxLen != nil {
					existing.maxLen = clonedRule.maxLen
				}
				if clonedRule.minLen != nil {
					existing.minLen = clonedRule.minLen
				}
				if clonedRule.regexPattern != "" {
					existing.regexPattern = clonedRule.regexPattern
				}
				if clonedRule.greater != nil {
					existing.greater = clonedRule.greater
				}
				if clonedRule.less != nil {
					existing.less = clonedRule.less
				}
				if clonedRule.greaterOrEqual != nil {
					existing.greaterOrEqual = clonedRule.greaterOrEqual
				}
				if clonedRule.lessOrEqual != nil {
					existing.lessOrEqual = clonedRule.lessOrEqual
				}
				if len(clonedRule.oneof) > 0 {
					existing.oneof = clonedRule.oneof
				}
				if clonedRule.isPII != nil {
					existing.isPII = clonedRule.isPII
				}
				if clonedRule.ctxID != "" {
					existing.ctxID = clonedRule.ctxID
				}
				if clonedRule.identityDomainURI != "" {
					existing.identityDomainURI = clonedRule.identityDomainURI
				}
				if clonedRule.identifierRank != nil {
					existing.identifierRank = clonedRule.identifierRank
				}
				if clonedRule.isPrimaryKey != nil {
					existing.isPrimaryKey = clonedRule.isPrimaryKey
				}
				if clonedRule.sortRank != nil {
					existing.sortRank = clonedRule.sortRank
				}
				if clonedRule.description != "" {
					existing.description = clonedRule.description
				}
			} else {
				ruleMap[clonedRule.fieldName] = clonedRule
			}
		}
	}

	for _, fRule := range ruleMap {
		idx, ok := fieldsByName[fRule.fieldName]
		if !ok {
			return nil, fmt.Errorf("supercargo: rule defined for unknown field %q on type %q", fRule.fieldName, typ.Name())
		}

		// Record metadata (PII is true if explicitly tagged or if context_id/identity_domain is present)
		isPIIBool := (fRule.isPII != nil && *fRule.isPII) || fRule.ctxID != "" || fRule.identityDomainURI != ""
		v.metadata[fRule.fieldName] = FieldMetadata{
			IsPII:             &isPIIBool,
			ContextID:         fRule.ctxID,
			IdentityDomainURI: fRule.identityDomainURI,
			IdentifierRank:    fRule.identifierRank,
			IsPrimaryKey:      fRule.isPrimaryKey,
			SortRank:          fRule.sortRank,
			Description:       fRule.description,
		}

		// Navigate type to find target kind
		fTyp := typ
		for _, i := range idx {
			for fTyp.Kind() == reflect.Ptr {
				fTyp = fTyp.Elem()
			}
			fTyp = fTyp.Field(i).Type
		}
		for fTyp.Kind() == reflect.Ptr {
			fTyp = fTyp.Elem()
		}

		closure, err := compileFieldRule(fRule, idx, fTyp)
		if err != nil {
			return nil, err
		}
		if closure != nil {
			v.checks = append(v.checks, closure)
		}
	}

	for _, idx := range structFieldsToRecurse {
		// Figure out the element type and prefetch its validator
		fTyp := typ
		for _, i := range idx {
			for fTyp.Kind() == reflect.Ptr {
				fTyp = fTyp.Elem()
			}
			fTyp = fTyp.Field(i).Type
		}
		for fTyp.Kind() == reflect.Ptr {
			fTyp = fTyp.Elem()
		}

		closure, err := generateRecursiveValidationClosure(idx, fTyp, compiling)
		if err != nil {
			return nil, err
		}
		v.checks = append(v.checks, closure)
	}

	return v, nil
}

func containsStruct(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return true
	}
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		elem := t.Elem()
		for elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Struct {
			return true
		}
	}
	return false
}

func generateRecursiveValidationClosure(idx []int, fieldTyp reflect.Type, compiling map[reflect.Type]*Validator) (func(reflect.Value, *ptrStack) error, error) {
	var elemType reflect.Type
	if fieldTyp.Kind() == reflect.Slice || fieldTyp.Kind() == reflect.Array || fieldTyp.Kind() == reflect.Map {
		elemType = fieldTyp.Elem()
	} else {
		elemType = fieldTyp
	}
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// Pre-resolve dependency at compile time!
	targetV, err := getValidatorInternal(elemType, compiling)
	if err != nil {
		return nil, err
	}

	return func(val reflect.Value, visited *ptrStack) error {
		fVal := val
		currVisited := visited

		for _, i := range idx {
			var isCycle bool
			fVal, currVisited, isCycle = derefAndCheckCycle(fVal, currVisited)
			if fVal.Kind() == reflect.Ptr && fVal.IsNil() || isCycle {
				return nil
			}
			if fVal.Kind() != reflect.Struct {
				return nil
			}
			fVal = fVal.Field(i)
		}
		return validateDeep(fVal, currVisited, targetV)
	}, nil
}

func validateDeep(val reflect.Value, visited *ptrStack, targetV *Validator) error {
	var isCycle bool
	val, currVisited, isCycle := derefAndCheckCycle(val, visited)
	if val.Kind() == reflect.Ptr && val.IsNil() || isCycle {
		return nil
	}

	switch val.Kind() {
	case reflect.Struct:
		return targetV.check(val, currVisited)

	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			var isCycle bool
			ev, evVisited, isCycle := derefAndCheckCycle(val.Index(i), currVisited)
			if ev.Kind() == reflect.Ptr && ev.IsNil() || isCycle {
				continue
			}
			if ev.Kind() == reflect.Struct {
				if err := targetV.check(ev, evVisited); err != nil {
					return &ValidationError{Field: fmt.Sprintf("[%d]", i), Reason: "element validation failed", Err: err}
				}
			}
		}

	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			var isCycle bool
			ev, evVisited, isCycle := derefAndCheckCycle(iter.Value(), currVisited)
			if ev.Kind() == reflect.Ptr && ev.IsNil() || isCycle {
				continue
			}
			if ev.Kind() == reflect.Struct {
				if err := targetV.check(ev, evVisited); err != nil {
					return &ValidationError{Field: fmt.Sprintf("[%v]", iter.Key().Interface()), Reason: "map value validation failed", Err: err}
				}
			}
		}
	}
	return nil
}

func navigateField(val reflect.Value, idx []int, notEmpty bool, name string, isPII bool, visited *ptrStack) (reflect.Value, bool, error) {
	fVal := val
	currVisited := visited

	for _, i := range idx {
		var isCycle bool
		fVal, currVisited, isCycle = derefAndCheckCycle(fVal, currVisited)
		if fVal.Kind() == reflect.Ptr && fVal.IsNil() {
			if notEmpty {
				return reflect.Value{}, true, newValidationError(name, "must not be empty", nil, isPII, nil)
			}
			return reflect.Value{}, true, nil
		}
		if isCycle {
			return reflect.Value{}, true, nil
		}
		if fVal.Kind() != reflect.Struct {
			return reflect.Value{}, true, nil
		}
		fVal = fVal.Field(i)
	}

	isNilPtr := false
	for fVal.Kind() == reflect.Ptr {
		if fVal.IsNil() {
			isNilPtr = true
			break
		}
		fVal = fVal.Elem()
	}

	if notEmpty && isNilPtr {
		return reflect.Value{}, true, newValidationError(name, "must not be empty", nil, isPII, nil)
	}
	if isNilPtr {
		return reflect.Value{}, true, nil
	}
	return fVal, false, nil
}

func compileFieldRule(fRule *FieldRuleBuilder, idx []int, fTyp reflect.Type) (func(reflect.Value, *ptrStack) error, error) {
	kind := fTyp.Kind()

	if kind == reflect.String {
		return buildStringRule(fRule, idx)
	}
	if kind >= reflect.Int && kind <= reflect.Int64 {
		return buildIntRule(fRule, idx)
	}
	if kind >= reflect.Uint && kind <= reflect.Uint64 {
		return buildUintRule(fRule, idx)
	}
	if kind == reflect.Float32 || kind == reflect.Float64 {
		return buildFloatRule(fRule, idx)
	}
	if kind == reflect.Slice || kind == reflect.Array || kind == reflect.Map {
		return buildCollectionRule(fRule, idx)
	}

	// For unsupported types, if there are specific scalar constraints requested, fail fast.
	if fRule.regexPattern != "" || fRule.maxLen != nil || fRule.minLen != nil || fRule.greater != nil || fRule.less != nil || fRule.greaterOrEqual != nil || fRule.lessOrEqual != nil || len(fRule.oneof) > 0 {
		return nil, fmt.Errorf("supercargo: unsupported type %s for scalar constraints on field %s", fTyp.Name(), fRule.fieldName)
	}

	if fRule.mustNotBeEmpty {
		name := fRule.fieldName
		isPII := (fRule.isPII != nil && *fRule.isPII) || fRule.ctxID != "" || fRule.identityDomainURI != ""
		return func(val reflect.Value, visited *ptrStack) error {
			_, earlyRet, err := navigateField(val, idx, fRule.mustNotBeEmpty, name, isPII, visited)
			if earlyRet {
				return err
			}
			return nil
		}, nil
	}

	return nil, nil
}

func buildStringRule(fRule *FieldRuleBuilder, idx []int) (func(reflect.Value, *ptrStack) error, error) {
	var regex *regexp.Regexp
	if fRule.regexPattern != "" {
		compiled, err := regexp.Compile(fRule.regexPattern)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid regex pattern for field %q: %w", fRule.fieldName, err)
		}
		regex = compiled
	}

	var greaterBound, lessBound, greaterOrEqualBound, lessOrEqualBound *numericBound
	var err error
	if fRule.greater != nil {
		gb, errFloat := parseNumericBound(*fRule.greater)
		if errFloat != nil {
			err = fmt.Errorf("supercargo: invalid numeric greater_than bounds %q for string field %q", *fRule.greater, fRule.fieldName)
		} else {
			greaterBound = gb
		}
	}
	if fRule.less != nil {
		lb, errFloat := parseNumericBound(*fRule.less)
		if errFloat != nil {
			err = fmt.Errorf("supercargo: invalid numeric less_than bounds %q for string field %q", *fRule.less, fRule.fieldName)
		} else {
			lessBound = lb
		}
	}
	if fRule.greaterOrEqual != nil {
		geb, errFloat := parseNumericBound(*fRule.greaterOrEqual)
		if errFloat != nil {
			err = fmt.Errorf("supercargo: invalid numeric greater_than_or_equal_to bounds %q for string field %q", *fRule.greaterOrEqual, fRule.fieldName)
		} else {
			greaterOrEqualBound = geb
		}
	}
	if fRule.lessOrEqual != nil {
		leb, errFloat := parseNumericBound(*fRule.lessOrEqual)
		if errFloat != nil {
			err = fmt.Errorf("supercargo: invalid numeric less_than_or_equal_to bounds %q for string field %q", *fRule.lessOrEqual, fRule.fieldName)
		} else {
			lessOrEqualBound = leb
		}
	}
	if err != nil {
		return nil, err
	}

	name := fRule.fieldName
	notEmpty := fRule.mustNotBeEmpty
	maxLen := fRule.maxLen
	minLen := fRule.minLen
	oneof := fRule.oneof
	isPII := (fRule.isPII != nil && *fRule.isPII) || fRule.ctxID != "" || fRule.identityDomainURI != ""

	return func(val reflect.Value, visited *ptrStack) error {
		fVal, earlyRet, err := navigateField(val, idx, notEmpty, name, isPII, visited)
		if earlyRet {
			return err
		}

		str := fVal.String()

		// Optional short-circuiting
		if !notEmpty && str == "" {
			return nil
		}

		if notEmpty && str == "" {
			return newValidationError(name, "must not be empty", nil, isPII, nil)
		}
		if maxLen != nil && utf8.RuneCountInString(str) > *maxLen {
			return newValidationError(name, fmt.Sprintf("length exceeds maximum of %d", *maxLen), str, isPII, nil)
		}
		if minLen != nil && utf8.RuneCountInString(str) < *minLen {
			return newValidationError(name, fmt.Sprintf("length is less than minimum of %d", *minLen), str, isPII, nil)
		}
		if regex != nil && !regex.MatchString(str) {
			return newValidationError(name, "does not match pattern", str, isPII, nil)
		}
		if greaterBound != nil {
			cmpRes, err := greaterBound.cmp(str)
			if err != nil {
				return newValidationError(name, "failed to parse as numeric for greater_than bounds check", str, isPII, nil)
			}
			if cmpRes <= 0 {
				return newValidationError(name, fmt.Sprintf("must be strictly greater than %s", greaterBound.str), str, isPII, nil)
			}
		}
		if lessBound != nil {
			cmpRes, err := lessBound.cmp(str)
			if err != nil {
				return newValidationError(name, "failed to parse as numeric for less_than bounds check", str, isPII, nil)
			}
			if cmpRes >= 0 {
				return newValidationError(name, fmt.Sprintf("must be strictly less than %s", lessBound.str), str, isPII, nil)
			}
		}
		if greaterOrEqualBound != nil {
			cmpRes, err := greaterOrEqualBound.cmp(str)
			if err != nil {
				return newValidationError(name, "failed to parse as numeric for greater_than_or_equal_to bounds check", str, isPII, nil)
			}
			if cmpRes < 0 {
				return newValidationError(name, fmt.Sprintf("must be greater than or equal to %s", greaterOrEqualBound.str), str, isPII, nil)
			}
		}
		if lessOrEqualBound != nil {
			cmpRes, err := lessOrEqualBound.cmp(str)
			if err != nil {
				return newValidationError(name, "failed to parse as numeric for less_than_or_equal_to bounds check", str, isPII, nil)
			}
			if cmpRes > 0 {
				return newValidationError(name, fmt.Sprintf("must be less than or equal to %s", lessOrEqualBound.str), str, isPII, nil)
			}
		}
		if len(oneof) > 0 {
			matched := false
			for _, allowed := range oneof {
				if str == allowed {
					matched = true
					break
				}
			}
			if !matched {
				return newValidationError(name, fmt.Sprintf("must be one of [%s]", strings.Join(oneof, ", ")), str, isPII, nil)
			}
		}
		return nil
	}, nil
}

func buildIntRule(fRule *FieldRuleBuilder, idx []int) (func(reflect.Value, *ptrStack) error, error) {
	var greaterInt, lessInt, greaterOrEqualInt, lessOrEqualInt *int64
	if fRule.greater != nil {
		gi, err := strconv.ParseInt(*fRule.greater, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid int for greater_than on field %q: %w", fRule.fieldName, err)
		}
		greaterInt = &gi
	}
	if fRule.less != nil {
		li, err := strconv.ParseInt(*fRule.less, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid int for less_than on field %q: %w", fRule.fieldName, err)
		}
		lessInt = &li
	}
	if fRule.greaterOrEqual != nil {
		gei, err := strconv.ParseInt(*fRule.greaterOrEqual, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid int for greater_than_or_equal_to on field %q: %w", fRule.fieldName, err)
		}
		greaterOrEqualInt = &gei
	}
	if fRule.lessOrEqual != nil {
		lei, err := strconv.ParseInt(*fRule.lessOrEqual, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid int for less_than_or_equal_to on field %q: %w", fRule.fieldName, err)
		}
		lessOrEqualInt = &lei
	}

	var allowedInts []int64
	if len(fRule.oneof) > 0 {
		allowedInts = make([]int64, len(fRule.oneof))
		for i, s := range fRule.oneof {
			parsed, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("supercargo: invalid int for oneof on field %q: %w", fRule.fieldName, err)
			}
			allowedInts[i] = parsed
		}
	}

	name := fRule.fieldName
	notEmpty := fRule.mustNotBeEmpty
	isPII := (fRule.isPII != nil && *fRule.isPII) || fRule.ctxID != "" || fRule.identityDomainURI != ""

	return func(val reflect.Value, visited *ptrStack) error {
		fVal, earlyRet, err := navigateField(val, idx, notEmpty, name, isPII, visited)
		if earlyRet {
			return err
		}

		num := fVal.Int()
		if greaterInt != nil && num <= *greaterInt {
			return newValidationError(name, fmt.Sprintf("must be strictly greater than %v", *greaterInt), num, isPII, nil)
		}
		if lessInt != nil && num >= *lessInt {
			return newValidationError(name, fmt.Sprintf("must be strictly less than %v", *lessInt), num, isPII, nil)
		}
		if greaterOrEqualInt != nil && num < *greaterOrEqualInt {
			return newValidationError(name, fmt.Sprintf("must be greater than or equal to %v", *greaterOrEqualInt), num, isPII, nil)
		}
		if lessOrEqualInt != nil && num > *lessOrEqualInt {
			return newValidationError(name, fmt.Sprintf("must be less than or equal to %v", *lessOrEqualInt), num, isPII, nil)
		}
		if len(allowedInts) > 0 {
			matched := false
			for _, allowed := range allowedInts {
				if num == allowed {
					matched = true
					break
				}
			}
			if !matched {
				return newValidationError(name, fmt.Sprintf("must be one of [%s]", strings.Join(fRule.oneof, ", ")), num, isPII, nil)
			}
		}
		return nil
	}, nil
}

func buildUintRule(fRule *FieldRuleBuilder, idx []int) (func(reflect.Value, *ptrStack) error, error) {
	var greaterUint, lessUint, greaterOrEqualUint, lessOrEqualUint *uint64
	if fRule.greater != nil {
		gu, err := strconv.ParseUint(*fRule.greater, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid uint for greater_than on field %q: %w", fRule.fieldName, err)
		}
		greaterUint = &gu
	}
	if fRule.less != nil {
		lu, err := strconv.ParseUint(*fRule.less, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid uint for less_than on field %q: %w", fRule.fieldName, err)
		}
		lessUint = &lu
	}
	if fRule.greaterOrEqual != nil {
		geu, err := strconv.ParseUint(*fRule.greaterOrEqual, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid uint for greater_than_or_equal_to on field %q: %w", fRule.fieldName, err)
		}
		greaterOrEqualUint = &geu
	}
	if fRule.lessOrEqual != nil {
		leu, err := strconv.ParseUint(*fRule.lessOrEqual, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid uint for less_than_or_equal_to on field %q: %w", fRule.fieldName, err)
		}
		lessOrEqualUint = &leu
	}

	var allowedUints []uint64
	if len(fRule.oneof) > 0 {
		allowedUints = make([]uint64, len(fRule.oneof))
		for i, s := range fRule.oneof {
			parsed, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("supercargo: invalid uint for oneof on field %q: %w", fRule.fieldName, err)
			}
			allowedUints[i] = parsed
		}
	}

	name := fRule.fieldName
	notEmpty := fRule.mustNotBeEmpty
	isPII := (fRule.isPII != nil && *fRule.isPII) || fRule.ctxID != "" || fRule.identityDomainURI != ""

	return func(val reflect.Value, visited *ptrStack) error {
		fVal, earlyRet, err := navigateField(val, idx, notEmpty, name, isPII, visited)
		if earlyRet {
			return err
		}

		num := fVal.Uint()
		if greaterUint != nil && num <= *greaterUint {
			return newValidationError(name, fmt.Sprintf("must be strictly greater than %v", *greaterUint), num, isPII, nil)
		}
		if lessUint != nil && num >= *lessUint {
			return newValidationError(name, fmt.Sprintf("must be strictly less than %v", *lessUint), num, isPII, nil)
		}
		if greaterOrEqualUint != nil && num < *greaterOrEqualUint {
			return newValidationError(name, fmt.Sprintf("must be greater than or equal to %v", *greaterOrEqualUint), num, isPII, nil)
		}
		if lessOrEqualUint != nil && num > *lessOrEqualUint {
			return newValidationError(name, fmt.Sprintf("must be less than or equal to %v", *lessOrEqualUint), num, isPII, nil)
		}
		if len(allowedUints) > 0 {
			matched := false
			for _, allowed := range allowedUints {
				if num == allowed {
					matched = true
					break
				}
			}
			if !matched {
				return newValidationError(name, fmt.Sprintf("must be one of [%s]", strings.Join(fRule.oneof, ", ")), num, isPII, nil)
			}
		}
		return nil
	}, nil
}

func buildFloatRule(fRule *FieldRuleBuilder, idx []int) (func(reflect.Value, *ptrStack) error, error) {
	var greaterFloat, lessFloat, greaterOrEqualFloat, lessOrEqualFloat *float64
	if fRule.greater != nil {
		gf, err := strconv.ParseFloat(*fRule.greater, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid float for greater_than on field %q: %w", fRule.fieldName, err)
		}
		greaterFloat = &gf
	}
	if fRule.less != nil {
		lf, err := strconv.ParseFloat(*fRule.less, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid float for less_than on field %q: %w", fRule.fieldName, err)
		}
		lessFloat = &lf
	}
	if fRule.greaterOrEqual != nil {
		gef, err := strconv.ParseFloat(*fRule.greaterOrEqual, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid float for greater_than_or_equal_to on field %q: %w", fRule.fieldName, err)
		}
		greaterOrEqualFloat = &gef
	}
	if fRule.lessOrEqual != nil {
		lef, err := strconv.ParseFloat(*fRule.lessOrEqual, 64)
		if err != nil {
			return nil, fmt.Errorf("supercargo: invalid float for less_than_or_equal_to on field %q: %w", fRule.fieldName, err)
		}
		lessOrEqualFloat = &lef
	}

	var allowedFloats []float64
	if len(fRule.oneof) > 0 {
		allowedFloats = make([]float64, len(fRule.oneof))
		for i, s := range fRule.oneof {
			parsed, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, fmt.Errorf("supercargo: invalid float for oneof on field %q: %w", fRule.fieldName, err)
			}
			allowedFloats[i] = parsed
		}
	}

	name := fRule.fieldName
	notEmpty := fRule.mustNotBeEmpty
	isPII := (fRule.isPII != nil && *fRule.isPII) || fRule.ctxID != "" || fRule.identityDomainURI != ""

	return func(val reflect.Value, visited *ptrStack) error {
		fVal, earlyRet, err := navigateField(val, idx, notEmpty, name, isPII, visited)
		if earlyRet {
			return err
		}

		num := fVal.Float()
		if greaterFloat != nil && num <= *greaterFloat {
			return newValidationError(name, fmt.Sprintf("must be strictly greater than %v", *greaterFloat), num, isPII, nil)
		}
		if lessFloat != nil && num >= *lessFloat {
			return newValidationError(name, fmt.Sprintf("must be strictly less than %v", *lessFloat), num, isPII, nil)
		}
		if greaterOrEqualFloat != nil && num < *greaterOrEqualFloat {
			return newValidationError(name, fmt.Sprintf("must be greater than or equal to %v", *greaterOrEqualFloat), num, isPII, nil)
		}
		if lessOrEqualFloat != nil && num > *lessOrEqualFloat {
			return newValidationError(name, fmt.Sprintf("must be less than or equal to %v", *lessOrEqualFloat), num, isPII, nil)
		}
		if len(allowedFloats) > 0 {
			matched := false
			for _, allowed := range allowedFloats {
				if num == allowed {
					matched = true
					break
				}
			}
			if !matched {
				return newValidationError(name, fmt.Sprintf("must be one of [%s]", strings.Join(fRule.oneof, ", ")), num, isPII, nil)
			}
		}
		return nil
	}, nil
}

func buildCollectionRule(fRule *FieldRuleBuilder, idx []int) (func(reflect.Value, *ptrStack) error, error) {
	name := fRule.fieldName
	notEmpty := fRule.mustNotBeEmpty
	maxLen := fRule.maxLen
	minLen := fRule.minLen
	isPII := (fRule.isPII != nil && *fRule.isPII) || fRule.ctxID != "" || fRule.identityDomainURI != ""

	return func(val reflect.Value, visited *ptrStack) error {
		fVal, earlyRet, err := navigateField(val, idx, notEmpty, name, isPII, visited)
		if earlyRet {
			return err
		}

		length := fVal.Len()

		// Optional short-circuiting
		if !notEmpty && length == 0 {
			return nil
		}

		if notEmpty && length == 0 {
			return newValidationError(name, "must not be empty", nil, isPII, nil)
		}
		if maxLen != nil && length > *maxLen {
			return newValidationError(name, fmt.Sprintf("length exceeds maximum of %d", *maxLen), length, isPII, nil)
		}
		if minLen != nil && length < *minLen {
			return newValidationError(name, fmt.Sprintf("length is less than minimum of %d", *minLen), length, isPII, nil)
		}
		return nil
	}, nil
}
