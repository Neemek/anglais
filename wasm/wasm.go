//go:build wasm && go1.23

package main

import (
	"errors"
	"log"
	"neemek.com/anglais/core"
	"syscall/js"
)

type JsResolver struct {
	jsResolver js.Value
}

func (r *JsResolver) Resolve(name string) (core.Node, error) {
	jsv := r.jsResolver.Invoke(name)

	if jsv.Type() == js.TypeUndefined {
		return nil, errors.New("cannot find import with name " + name)
	}

	if jsv.Type() != js.TypeString {
		return nil, errors.New("invalid value for source: " + jsv.String())
	}

	source := jsv.String()

	l := core.NewLexer(source)
	tokens, err := l.Tokenize()
	if err != nil {
		return nil, err
	}

	p := core.NewParser(tokens)
	tree, err := p.Parse()
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func jsError(err error) interface{} {
	return jsErrorOfString(err.Error())
}

func jsErrorOfString(err string) interface{} {
	errorConstructor := js.Global().Get("Error")
	errorObject := errorConstructor.New(err)

	return errorObject
}

func run(_ js.Value, args []js.Value) interface{} {
	source := args[0].String()
	outputHandler := args[1]
	errorHandler := args[2]
	resolver := args[3]
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
		var e core.ParsingError
		if errors.As(err, &e) {
			errorHandler.Invoke(e.Format([]rune(source)))
			return nil
		}
		errorHandler.Invoke(err.Error())
		return nil
	}

	log.Printf("Parsed tree: %s", tree.String())

	compiler := core.NewCompiler([]rune(source))

	log.Printf("Set imports resolver", tree.String())

	compiler.SetImportsResolver(&JsResolver{
		resolver,
	})

	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic recovered: %v", err)
		}
	}()

	err = compiler.Compile(tree)
	if err != nil {
		var e core.CompilerError
		if errors.As(err, &e) {
			errorHandler.Invoke(e.Format())
			return nil
		}

		errorHandler.Invoke(err.Error())
		return nil
	}

	log.Printf("Compiled tree (into %v instructions)", len(compiler.Chunk.Bytecode))

	vm := core.NewVM(compiler.Chunk, 256, 256)

	// overwrite output
	vm.SetGlobal("write", &core.BuiltinFunctionValue{
		Name: "write",
		Signature: &core.FunctionSignature{
			[]core.TypeSignature{&core.StringSignature{}},
			&core.NilSignature{},
		},
		F: func(vm *core.VM, this core.Value, args []core.Value) (core.Value, error) {
			s := args[0].String()
			log.Printf("Writing value: %s", s)
			outputHandler.Invoke(js.ValueOf(s + "\n"))
			return nil, nil
		},
	})
	vm.SetGlobal("print", &core.BuiltinFunctionValue{
		Name: "print",
		Signature: &core.FunctionSignature{
			In:  []core.TypeSignature{&core.StringSignature{}},
			Out: &core.NilSignature{},
		},
		F: func(vm *core.VM, this core.Value, args []core.Value) (core.Value, error) {
			s := args[0].String()
			log.Printf("Printing value: %s", s)
			outputHandler.Invoke(js.ValueOf(s))
			return nil, nil
		},
	})

	for vm.Next() {
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
