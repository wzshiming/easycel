package easycel_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/cel-go/cel"

	"github.com/wzshiming/easycel"
)

func Example() {
	registry := easycel.NewRegistry("example")
	err := registry.RegisterFunction("greet", func(name string) string {
		return "Hello, " + name + "!"
	})
	if err != nil {
		log.Fatal(err)
	}
	// The value only sets the type; Eval supplies the actual value.
	err = registry.RegisterVariable("name", "")
	if err != nil {
		log.Fatal(err)
	}

	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		log.Fatal(err)
	}
	program, err := env.Program(`greet(name)`)
	if err != nil {
		log.Fatal(err)
	}
	out, _, err := program.Eval(map[string]any{"name": "CEL"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Value())
	// Output:
	// Hello, CEL!
}

func ExampleRegistry_RegisterType() {
	type User struct {
		Name string `easycel:"name"`
		Age  int    `easycel:"age"`
	}

	registry := easycel.NewRegistry("example")
	err := registry.RegisterType(User{})
	if err != nil {
		log.Fatal(err)
	}
	err = registry.RegisterVariable("user", User{})
	if err != nil {
		log.Fatal(err)
	}

	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		log.Fatal(err)
	}
	program, err := env.Program(`user.name + " is " + string(user.age)`)
	if err != nil {
		log.Fatal(err)
	}
	out, _, err := program.Eval(map[string]any{"user": User{Name: "Alice", Age: 30}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Value())
	// Output:
	// Alice is 30
}

func ExampleRegistry_RegisterMethod() {
	registry := easycel.NewRegistry("example")
	err := registry.RegisterMethod("repeat", func(s string, n int) string {
		return strings.Repeat(s, n)
	})
	if err != nil {
		log.Fatal(err)
	}

	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		log.Fatal(err)
	}
	program, err := env.Program(`"ab".repeat(3)`)
	if err != nil {
		log.Fatal(err)
	}
	out, _, err := program.Eval(cel.NoVars())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Value())
	// Output:
	// ababab
}
