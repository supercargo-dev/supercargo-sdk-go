package sc

import (
	"reflect"
	"sync"
)

var globalValidatorCache sync.Map // map[reflect.Type]*validatorFuture

type validatorFuture struct {
	once sync.Once
	v    *Validator
	err  error
	done chan struct{}
}

// ptrStack is an immutable linked list used for path-scoped cycle detection
type ptrStack struct {
	ptr  uintptr
	prev *ptrStack
}

func derefAndCheckCycle(val reflect.Value, visited *ptrStack) (reflect.Value, *ptrStack, bool) {
	currVisited := visited
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return val, currVisited, false
		}
		ptr := val.Pointer()
		curr := currVisited
		isCycle := false
		for curr != nil {
			if curr.ptr == ptr {
				isCycle = true
				break
			}
			curr = curr.prev
		}
		if isCycle {
			return val, currVisited, true
		}
		currVisited = &ptrStack{ptr: ptr, prev: currVisited}
		val = val.Elem()
	}
	return val, currVisited, false
}
