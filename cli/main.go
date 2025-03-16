package main

import (
	"github.com/alecthomas/kong"
	"log"
	"neemek.com/anglais/core"
	"os"
	"path/filepath"
)

type Context struct {
	Debug bool
}

type RunCmd struct {
	Bytecode bool   `name:"bytecode" short:"c" help:"Run file as if it's bytecode"`
	File     string `arg:"" name:"file" help:"File to read program from" type:"existingfile"`
}

// WorkingDirectoryResolver resolves imports relative to the working directory
type WorkingDirectoryResolver struct {
	workingDirectory string
}

func (r *WorkingDirectoryResolver) Resolve(path string) (core.Node, error) {
	pth := filepath.Join(r.workingDirectory, path)
	f, err := os.ReadFile(pth)
	if err != nil {
		return nil, err
	}

	str := string(f)

	l := core.NewLexer(str)

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

		tree, err := p.Parse()

		// if there were parsing errors, print them out
		if err != nil {
			print(err.(*core.ParsingError).Format([]rune(src)))
			log.Fatal("Parsing had errors")
		}

		if ctx.Debug {
			log.Println("Initialized compiler")
		}
		c := core.NewCompiler()

		if ctx.Debug {
			log.Println("Setting imports resolver")
		}

		dir, _ := filepath.Split(cmd.File)
		c.SetImportsResolver(&WorkingDirectoryResolver{
			dir,
		})

		if ctx.Debug {
			log.Println("Compiling parse tree")
		}
		err = c.Compile(tree)
		if err != nil {
			return err
		}

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
	for vm.Next() {
	}

	return nil
}

type CompileCmd struct {
	File   string `arg:"" name:"file" help:"File to compile program from" type:"existingfile"`
	Output string `arg:"" name:"output" help:"File path to output bytecode to" type:"path"`
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

	tree, err := p.Parse()

	if err != nil {
		print(err.(*core.ParsingError).Format([]rune(src)))
	}

	if ctx.Debug {
		log.Println("Initialized compiler")
	}

	c := core.NewCompiler()

	if ctx.Debug {
		log.Println("Setting import resolver")
	}

	dir, _ := filepath.Split(cmd.File)
	c.SetImportsResolver(&WorkingDirectoryResolver{
		dir,
	})

	if ctx.Debug {
		log.Println("Compiling parse tree")
	}

	err = c.Compile(tree)
	if err != nil {
		return err
	}

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
	Debug bool `short:"D" name:"debug" help:"Enable debug mode."`

	Run        RunCmd     `cmd:"" name:"run" help:"Run program."`
	CompileCmd CompileCmd `cmd:"" name:"compile" help:"Compile program to bytecode."`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
