//go:build wasm && go1.23

package main

import (
	"log"
	"neemek.com/anglais/core"
	"syscall/js"
)

func jsError(err error) interface{} {
	errorConstructor := js.Global().Get("Error")
	errorObject := errorConstructor.New(err.Error())

	return errorObject
}

func jsErrorOfString(err string) interface{} {
	errorConstructor := js.Global().Get("Error")
	errorObject := errorConstructor.New(err)

	return errorObject
}

func run(this js.Value, args []js.Value) interface{} {
	source := args[0].String()
	outputHandler := args[1]
	doRun := args[2]
	log.Printf("got source: %s", source)

	lexer := core.NewLexer(source)
	tokens, err := lexer.Tokenize()

	if err != nil {
		return jsError(err)
	}

	log.Printf("got tokens: %v", tokens)

	parser := core.NewParser(tokens)

	tree, err := parser.Parse()

	if err != nil {
		return jsErrorOfString(err.Error())
	}

	log.Printf("Parsed tree: %s", tree.String())

	compiler := core.NewCompiler()

	compiler.Compile(tree)

	log.Printf("Compiled tree (into %i instructions)", len(compiler.Chunk.Bytecode))

	vm := core.NewVM(compiler.Chunk, 256, 256)

	// overwrite output
	vm.SetGlobal("write", core.BuiltinFunctionValue{
		Name:       "write",
		Parameters: []string{"value"},
		F: func(v map[string]core.Value) core.Value {
			log.Printf("Writing value: %s", v["value"].String())
			outputHandler.Invoke(js.ValueOf(v["value"].String()))
			return nil
		},
	})
	vm.SetGlobal("print", core.BuiltinFunctionValue{
		Name:       "print",
		Parameters: []string{"value"},
		F: func(v map[string]core.Value) core.Value {
			log.Printf("Printing value: %s", v["value"].String())
			outputHandler.Invoke(js.ValueOf(v["value"].String() + "\n"))
			return nil
		},
	})

	for doRun.Invoke().Bool() && vm.HasNext() {
		vm.Next()
	}

	log.Println("Finished executing")

	return js.Null()
}

func main() {
	log.Println("Initializing Anglais WASM module")

	js.Global().Set("run", js.FuncOf(run))

	log.Println("Initialized Anglais WASM module")

	// keep alive so run func can be used
	select {}
}
