package main

import (
	"github.com/alecthomas/kong"
	"log"
	"neemek.com/anglais/core"
	"os"
)

type Context struct {
	Debug bool
}

type RunCmd struct {
	Bytecode bool   `name:"bytecode" short:"c" help:"Run file as if it's bytecode"`
	File     string `arg:"" name:"file" help:"File to read program from" type:"existingfile"`
}

func (cmd *RunCmd) Run(ctx *Context) error {
	if ctx.Debug {
		log.Println("Reading file")
	}

	f, err := os.ReadFile(cmd.File)

	if err != nil {
		return err
	}

	var chunk *core.Chunk
	if !cmd.Bytecode {
		src := string(f)

		if ctx.Debug {
			log.Println("Initialized lexer")
		}
		l := core.NewLexer(src)

		if ctx.Debug {
			log.Println("Lexing all tokens")
		}
		tokens, err := l.Tokenize()

		if err != nil {
			log.Fatal(err)
		}

		if len(tokens) <= 1 {
			log.Fatal("Empty file")
		}

		if ctx.Debug {
			log.Printf("Lexed %d tokens", len(tokens))

		}
		p := core.NewParser(tokens)

		if ctx.Debug {
			log.Println("Initialized parser")
		}

		defer func() {
			if r := recover(); r != nil {
				if r != "no more tokens" { // if the panic was not caused by the parser, it should not be recovered.
					panic(r)
				}
				for _, e := range p.Errors {
					print(e.Format([]rune(src)))
				}
			}
		}()

		tree := p.Parse()

		// if there were parsing errors, print them out
		if len(p.Errors) > 0 {
			for _, e := range p.Errors {
				print(e.Format([]rune(src)))
			}
			log.Fatal("Parsing had errors")
		}

		if ctx.Debug {
			log.Println("Initialized compiler")
		}
		c := core.NewCompiler()

		if ctx.Debug {
			log.Println("Compiling parse tree")
		}
		c.Compile(tree)

		chunk = c.Chunk
	} else {
		if ctx.Debug {
			log.Println("Registering GOB types")
		}

		core.RegisterGOBTypes()

		if ctx.Debug {
			log.Println("Deserializing file")
		}

		chunk = core.DeserializeChunk(f)
	}

	if ctx.Debug {
		log.Println("Printing chunk")

		print(chunk.String())

		log.Println("Initialized VM")
	}
	vm := core.NewVM(chunk, 256, 256)

	if ctx.Debug {
		log.Println("Executing bytecode")
		log.Println("=v= output =v=")
	}
	// execute order 66
	for vm.HasNext() && vm.Next() {
	}

	return nil
}

type CompileCmd struct {
	File   string `arg:"" name:"file" help:"File to compile program from" type:"existingfile"`
	Output string `arg:"" name:"output" optional:"" help:"File path to output bytecode to" type:"path"`
}

func (cmd *CompileCmd) Compile(ctx *Context) error {
	if ctx.Debug {
		log.Println("Reading file")
	}

	f, err := os.ReadFile(cmd.File)

	if err != nil {
		return err
	}

	src := string(f)

	if ctx.Debug {
		log.Println("Initializing lexer")
	}
	l := core.NewLexer(src)

	if ctx.Debug {
		log.Println("Lexing all tokens")
	}
	tokens, err := l.Tokenize()

	if err != nil {
		log.Fatal(err)
	}

	if ctx.Debug {
		log.Println("Initializing parser")
	}
	p := core.NewParser(tokens)

	if ctx.Debug {
		log.Println("Parsing tree")
	}
	tree := p.Parse()

	if ctx.Debug {
		log.Println("Initialized compiler")
	}
	c := core.NewCompiler()

	if ctx.Debug {
		log.Println("Compiling parse tree")
	}

	c.Compile(tree)

	if ctx.Debug {
		log.Println("Registering GOB types")
	}

	core.RegisterGOBTypes()

	if ctx.Debug {
		log.Println("Serializing chunk")
	}

	serialized := c.Chunk.Serialize()

	if ctx.Debug {
		log.Println("Writing file")
	}

	err = os.WriteFile(cmd.Output, serialized, 0666)

	if err != nil {
		return err
	}

	return nil
}

var cli struct {
	Debug bool `short:"d" help:"Enable debug mode."`

	Run        RunCmd     `cmd:"" name:"run" help:"Run program."`
	CompileCmd CompileCmd `cmd:"" name:"compile" help:"Compile program to bytecode."`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
