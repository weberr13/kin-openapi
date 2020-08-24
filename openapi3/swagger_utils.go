package openapi3

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"
)

type RefOrValue interface {
	Resolved() bool
	ClearRef()
	GetRef() string
	IsRef() bool
}

const rootObjectDepth = 3

func IsExternalRef(rr RefOrValue) bool {
	return strings.Index(rr.GetRef(), "#") >= 0
}

func clearResolvedExternalRef(rr RefOrValue) {
	if rr.IsRef() && IsExternalRef(rr) && rr.Resolved() {
		rr.ClearRef()
	}
}

// ClearResolvedExternalRefs Recursively iterate over the swagger structure, resetting <Type>Ref structs where
// the reference is remote and was resolved
func ClearResolvedExternalRefs(swagger *Swagger) {
	visited := map[reflect.Value]struct{}{}
	resetExternalRef(reflect.ValueOf(swagger), visited)
}

func resetExternalRef(c reflect.Value, visited map[reflect.Value]struct{}) {
	if _, ok := visited[c]; ok {
		return
	}
	visited[c] = struct{}{}
	switch c.Kind() {
	// If it is a struct, check if it's the desired type first before drilling into fields
	// Further if this is a <Type>Ref struct, reset the reference if it's remote and resolved
	case reflect.Struct:
		if c.CanAddr() {
			rov, ok := c.Addr().Interface().(RefOrValue)
			if ok {
				clearResolvedExternalRef(rov)
			}
		}
		for i := 0; i < c.NumField(); i++ {
			resetExternalRef(c.Field(i), visited)
		}

	// If it is a pointer or interface we need to unwrap and call once again
	case reflect.Interface, reflect.Ptr:
		c2 := c.Elem()
		if c2.IsValid() {
			resetExternalRef(c2, visited)
		}

	// If it is a slice we iterate over each each element
	case reflect.Slice:
		for i := 0; i < c.Len(); i++ {
			resetExternalRef(c.Index(i), visited)
		}

	// If it is a map we iterate over each of the key,value pairs
	case reflect.Map:
		mi := c.MapRange()
		for mi.Next() {
			resetExternalRef(mi.Value(), visited)
		}

	// And everything else will simply be ignored
	default:
	}
}

// BuildCompoenentRefMap Recursively iterate over the swagger structure, building map of references to
// components
func BuildCompoenentRefMap(swagger *Swagger) map[interface{}]string {
	refMap := map[interface{}]string{}
	buildComponentRefMap(0, "#", reflect.ValueOf(swagger), refMap)
	return refMap
}

func setRef(c reflect.Value, refMap map[interface{}]string) {
	if !c.CanInterface() {
		return
	}

	// todo: currently only doing this for references
	ci := c.Interface()
	sr, ok := ci.(*Schema)
	if !ok {
		return
	}

	if sr.Metadata.Path.Fragment == "" {
		return
	}

	if _, ok := refMap[sr]; !ok {
		s := fnv.New64a()
		_, _ = s.Write([]byte(sr.Metadata.Path.Host + sr.Metadata.Path.Path))
		sfx := "_" + fmt.Sprintf("%x", s.Sum64())

		sr.Metadata.Path.Fragment += sfx
		sr.Metadata.ID += sfx

		refMap[sr] = "#" + sr.Metadata.Path.Fragment
	}

}

func isRefSet(c reflect.Value, refMap map[interface{}]string) bool {
	if !c.CanInterface() {
		return false
	}

	ci := c.Interface()
	sr, ok := ci.(*Schema)
	if !ok {
		return false
	}

	_, ok = refMap[sr]
	return ok
}

func buildComponentRefMap(level int, name string, c reflect.Value, refMap map[interface{}]string) {
	if isRefSet(c, refMap) {
		return
	}

	// debug
	//switch c.Kind() {
	//case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Struct:
	//	fmt.Printf( "%3d: %-60s %s\n", level, fmt.Sprintf( "%s'%s' (%s %s)", strings.Repeat(" ", level), name, c.Kind().String(), c.Type().Name()), c.String())
	//}

	switch c.Kind() {
	// If it is a struct, check if it's the desired type first before drilling into fields
	case reflect.Struct:
		for i := 0; i < c.NumField(); i++ {
			buildComponentRefMap(level+1, c.Type().Field(i).Name, c.Field(i), refMap)
		}

	// If it is a pointer or interface we need to unwrap and call once again
	// if this happens at the right level, this is a pointer to a component we want to remember (done in setRef)
	case reflect.Interface, reflect.Ptr:
		// 3 because we have to peel : * -> Struct(SchenaRef).Value -> * -> Struct(Schema) as minimum and we avoid root level schemas
		if level > rootObjectDepth {
			setRef(c, refMap)
		}
		c2 := c.Elem()
		if c2.IsValid() {
			buildComponentRefMap(level+1, name, c2, refMap)
		}

	// If it is a slice we iterate over each each element
	case reflect.Slice:
		for i := 0; i < c.Len(); i++ {
			buildComponentRefMap(level+1, name, c.Index(i), refMap)
		}

	// If it is a map we iterate over each of the key,value pairs
	case reflect.Map:
		mi := c.MapRange()
		for mi.Next() {
			//fmt.Printf("visiting element: %v\n", mi.Key())
			buildComponentRefMap(level+1, mi.Key().String(), mi.Value(), refMap)
		}

	// And everything else will simply be ignored
	default:
	}
}
