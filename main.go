package main

import (
	"github.com/alecthomas/kong"
	"log"
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

	var chunk *Chunk
	if !cmd.Bytecode {
		src := string(f)

		if ctx.Debug {
			log.Println("Initialized lexer")
		}
		l := NewLexer(src)

		if ctx.Debug {
			log.Println("Lexing all tokens")
		}
		tokens, err := l.Tokenize()

		if err != nil {
			log.Fatal(err)
		}

		if ctx.Debug {
			log.Println("Initialized parser")
		}
		p := NewParser(tokens)

		if ctx.Debug {
			log.Println("Parsed tree")
		}

		defer func() {
			if r := recover(); r != nil {
				for _, e := range p.errors {
					e.Print(src)
				}
			}
		}()

		tree := p.Parse()

		if ctx.Debug {
			log.Println("Initialized compiler")
		}
		c := NewCompiler()

		if ctx.Debug {
			log.Println("Compiling parse tree")
		}
		c.Compile(tree)

		chunk = c.chunk
	} else {
		if ctx.Debug {
			log.Println("Registering GOB types")
		}

		RegisterGOBTypes()

		if ctx.Debug {
			log.Println("Deserializing file")
		}

		chunk = DeserializeChunk(f)
	}

	if ctx.Debug {
		log.Println("Printing chunk")

		print(chunk.String())

		log.Println("Initialized VM")
	}
	vm := NewVM(chunk, 256, 256)

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

func (cmd *CompileCmd) Run(ctx *Context) error {
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
	l := NewLexer(src)

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
	p := NewParser(tokens)

	if ctx.Debug {
		log.Println("Parsing tree")
	}
	tree := p.Parse()

	if ctx.Debug {
		log.Println("Initialized compiler")
	}
	c := NewCompiler()

	if ctx.Debug {
		log.Println("Compiling parse tree")
	}

	c.Compile(tree)

	if ctx.Debug {
		log.Println("Registering GOB types")
	}

	RegisterGOBTypes()

	if ctx.Debug {
		log.Println("Serializing chunk")
	}

	serialized := c.chunk.Serialize()

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
