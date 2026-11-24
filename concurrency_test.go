package easycel

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/google/cel-go/cel"
)

// uniqueStructType returns a struct type that the caches have not seen: it has a field of its own.
// The prefix keeps the types of one test apart from those of another, because reflect returns the same type for the same fields.
func uniqueStructType(prefix string, i int) reflect.Type {
	return reflect.StructOf([]reflect.StructField{
		{Name: "Name", Type: reflect.TypeOf(""), Tag: `json:"name"`},
		{Name: fmt.Sprintf("%s%d", prefix, i), Type: reflect.TypeOf(0), Tag: `json:"extra"`},
	})
}

// TestConcurrentStructEnvironments checks that registries and environments that are independent of each other
// can be created and used from several goroutines. They share the caches of types and of struct fields.
// Each goroutine brings a struct type that is new to the process, because a cache is written the first time it sees a type.
// It uses only the exported functions of the package.
// Without a lock it fails with "fatal error: concurrent map writes", or with a data race under -race.
// It can also fail with an error for a valid expression, when a goroutine reads a type that another one has not finished.
func TestConcurrentStructEnvironments(t *testing.T) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 16; i++ {
		typ := uniqueStructType("Extra", i)
		wg.Add(1)
		// The type is an argument: before Go 1.22 the goroutines would share the variable of the loop.
		go func(typ reflect.Type) {
			defer wg.Done()
			<-start

			val := reflect.New(typ).Interface()
			registry := NewRegistry("example", WithTagName("json"))
			if err := registry.RegisterType(reflect.New(typ).Elem().Interface()); err != nil {
				t.Error(err)
				return
			}
			if err := registry.RegisterVariable("self", val); err != nil {
				t.Error(err)
				return
			}
			env, err := NewEnvironment(cel.Lib(registry))
			if err != nil {
				t.Error(err)
				return
			}
			// Checking the expression asks for the type of the field "name".
			prog, err := env.Program(`self.name == "x"`)
			if err != nil {
				t.Error(err)
				return
			}
			if _, _, err := prog.Eval(map[string]any{"self": val}); err != nil {
				t.Error(err)
			}
		}(typ)
	}
	close(start)
	wg.Wait()
}

// TestCachesConcurrentUse checks that all four caches can be used from several goroutines:
// typeValueMap, typeTraitMap, structFieldMap and structFields.
// It calls the functions that use them directly, so it covers them whatever the exported API reaches.
// Without a lock it fails with "fatal error: concurrent map writes", or with a data race under -race.
func TestCachesConcurrentUse(t *testing.T) {
	// a type that all goroutines read, while each of them writes the types that are new to the caches
	type shared struct {
		Name string `json:"name"`
	}
	sharedType := reflect.TypeOf(shared{})

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 32; i++ {
		typ := uniqueStructType("Cache", i)
		wg.Add(1)
		go func(typ reflect.Type) {
			defer wg.Done()
			<-start
			// getTypeValue calls itself for the element type of a pointer, and calls getTrait for a struct.
			getTypeValue(reflect.PointerTo(typ))
			getTrait(typ)
			getStructFieldMap(typ, "json")
			getStructFields(typ, "json")

			getTypeValue(sharedType)
			getStructFieldMap(sharedType, "json")
			getStructFields(sharedType, "json")
		}(typ)
	}
	close(start)
	wg.Wait()
}
