package flotilla

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

func existsIn(s string, l []string) bool {
	for _, x := range l {
		if s == x {
			return true
		}
	}
	return false
}

func dirAdd(dir string, dirs []string) []string {
	if dirAppendable(dir, dirs) {
		dirs = append(dirs, dir)
	}
	return dirs
}

func dirAppendable(dir string, dirs []string) bool {
	for _, d := range dirs {
		if d == dir {
			return false
		}
	}
	return true
}

func isFunc(fn interface{}) bool {
	return reflect.ValueOf(fn).Kind() == reflect.Func
}

// Compare functions, see https://github.com/zenazn/goji/blob/master/web/func_equal.go
func funcEqual(a, b interface{}) bool {
	if !isFunc(a) || !isFunc(b) {
		panic("funcEqual: type error!")
	}
	av := reflect.ValueOf(&a).Elem()
	bv := reflect.ValueOf(&b).Elem()
	return av.InterfaceData() == bv.InterfaceData()
}

func mapvalueFunc(fns map[string]interface{}) map[string]reflect.Value {
	newmap := make(map[string]reflect.Value)

	for k, v := range fns {
		newmap[k] = valueFunc(v)
	}

	return newmap
}

func valueFunc(fn interface{}) reflect.Value {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		panic(newError("Provided:(%+v, type: %T), but it is not a function", fn, fn))
	}
	if !goodFunc(v.Type()) {
		panic(newError("Cannot add context function %q with %d results\nreturn must be 1 value, or 1 value and 1 error value", fn, v.Type().NumOut()))
	}
	return v
}

// checks that the function or method has signature conforming to either one
// return value or one return value plus one returned error value.
func goodFunc(typ reflect.Type) bool {
	switch {
	case typ.NumOut() == 1:
		return true
	case typ.NumOut() == 2 && typ.Out(1) == errorType:
		return true
	}
	return false
}

// canBeNil reports whether an untyped nil can be assigned to the type. See reflect.Zero.
func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}

// from http://golang.org/src/pkg/text/template/funcs.go
func call(fn reflect.Value, args ...interface{}) (interface{}, error) {
	typ := fn.Type()
	numIn := typ.NumIn()
	var dddType reflect.Type
	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return nil, fmt.Errorf("wrong number of args: got %d want at least %d", len(args), numIn-1)
		}
		dddType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return nil, fmt.Errorf("wrong number of args: got %d want %d", len(args), numIn)
		}
	}
	argv := make([]reflect.Value, len(args))
	for i, arg := range args {
		value := reflect.ValueOf(arg)
		// Compute the expected type. Clumsy because of variadics.
		var argType reflect.Type
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		} else {
			argType = dddType
		}
		if !value.IsValid() && canBeNil(argType) {
			value = reflect.Zero(argType)
		}
		if !value.Type().AssignableTo(argType) {
			return nil, fmt.Errorf("arg %d has type %s; should be %s", i, value.Type(), argType)
		}
		argv[i] = value
	}
	result := fn.Call(argv)
	if len(result) == 2 && !result[1].IsNil() {
		return result[0].Interface(), result[1].Interface().(error)
	}
	return result[0].Interface(), nil
}

func pathDropFilepathSplat(path string) string {
	if fp := strings.Split(path, "/"); fp[len(fp)-1] == "*filepath" {
		return strings.Join(fp[0:len(fp)-1], "/")
	}
	return path
}
