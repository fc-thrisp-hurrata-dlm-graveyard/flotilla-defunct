package flotilla

import (
	"path/filepath"
	"reflect"
)

func existsIn(s string, l []string) bool {
	for _, x := range l {
		if s == x {
			return true
		}
	}
	return false
}

func dirAdd(dir string, envDirs []string) []string {
	adddir := dirAbs(dir)
	if dirAppendable(adddir, envDirs) {
		envDirs = append(envDirs, adddir)
	}
	return envDirs
}

func dirAbs(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	} else {
		return filepath.Join(workingPath, dir)
	}
}

func dirAppendable(dir string, envDirs []string) bool {
	for _, d := range envDirs {
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
