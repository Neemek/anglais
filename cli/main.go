package main

import (
	"bufio"
	"errors"
	"github.com/alecthomas/kong"
	"log"
	"neemek.com/anglais/core"
	"os"
	"path"
	"path/filepath"
)

type Context struct {
	Debug bool
}

type RunCmd struct {
	IgnoreWarnings bool   `name:"ignore-warnings" help:"Ignore warning messages"`
	Bytecode       bool   `name:"bytecode" short:"c" help:"Run file as if it's bytecode"`
	File           string `arg:"" name:"file" help:"File to read program from" type:"existingfile"`
}

// WorkingDirectoryResolver resolves imports relative to the working directory
type WorkingDirectoryResolver struct {
	workingDirectory string
}

func (r *WorkingDirectoryResolver) Resolve(path string) (string, error) {
	pth := filepath.Join(r.workingDirectory, path)
	f, err := os.ReadFile(pth)
	if err != nil {
		return "", err
	}

	return string(f), nil
}

func (r *WorkingDirectoryResolver) IsSame(a, b string) bool {
	apath := filepath.Clean(filepath.Join(r.workingDirectory, a))
	bpath := filepath.Clean(filepath.Join(r.workingDirectory, b))

	return apath == bpath
}

func makeChunk(ctx *Context, fpath string, ignoreWarnings bool) (*core.Chunk, error) {
	if ctx.Debug {
		log.Println("Reading file")
	}

	f, err := os.ReadFile(fpath)

	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	if len(tokens) <= 1 {
		return nil, errors.New("empty file")
	}

	if ctx.Debug {
		log.Printf("Lexed %d tokens", len(tokens))

	}
	p := core.NewParser(src, tokens)

	if ctx.Debug {
		log.Println("Initialized parser")
	}

	pathat, err := filepath.Abs(fpath)
	if err != nil {
		return nil, err
	}

	tree, err := p.Parse(pathat)

	if ctx.Debug {
		log.Printf("Parsed tree, meaning:\n%s", tree)
	}

	// if there were parsing errors, print them out
	if err != nil {
		print(err.(core.ParsingError).Format())
		log.Fatal("Parsing had errors")
	}

	if ctx.Debug {
		log.Println("Initialized compiler")
	}
	c := core.NewCompiler([]rune(src))

	if ctx.Debug {
		log.Println("Setting imports resolver")
	}

	dir, _ := path.Split(fpath)
	c.SetImportsResolver(&WorkingDirectoryResolver{
		dir,
	})

	if ctx.Debug {
		log.Println("Compiling parse tree")
	}
	err = c.Compile(tree)
	if err != nil {
		var e core.FormatedError
		if errors.As(err, &e) {
			log.Fatal(e.Format())
		}
		log.Fatal(err)
	}

	// if there were non-critical warnings, report them
	if !ignoreWarnings && len(c.Warnings) != 0 {
		for _, warning := range c.Warnings {
			log.Println(warning.Format())
		}
		log.Fatal("compiler reported warning(s) (ignore warnings with the --ignore-warnings option)")
	}

	return c.Chunk, nil
}

func (cmd *RunCmd) Run(ctx *Context) error {
	var chunk *core.Chunk
	if !cmd.Bytecode {
		c, err := makeChunk(ctx, cmd.File, cmd.IgnoreWarnings)
		if err != nil {
			return err
		}
		chunk = c
	} else {
		if ctx.Debug {
			log.Println("Reading file")
		}

		f, err := os.ReadFile(cmd.File)

		if err != nil {
			return err
		}

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
	File           string `arg:"" name:"file" help:"File to compile program from" type:"existingfile"`
	Output         string `arg:"" name:"output" help:"File path to output bytecode to" type:"path"`
	IgnoreWarnings bool   `name:"ignore-warnings" help:"Ignore warning messages"`
}

func (cmd *CompileCmd) Run(ctx *Context) error {
	c, err := makeChunk(ctx, cmd.File, cmd.IgnoreWarnings)
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

	serialized := c.Serialize()

	if ctx.Debug {
		log.Println("Writing file")
	}

	err = os.WriteFile(cmd.Output, serialized, 0666)

	if err != nil {
		return err
	}

	return nil
}

type ReplCmd struct {
}

func (cmd *ReplCmd) Run(ctx *Context) error {
	c := core.NewCompiler([]rune(""))
	vm := core.NewVM(core.NewChunk([]core.Bytecode{}, []core.Value{}), 256, 256)

	reader := bufio.NewReader(os.Stdin)
	for {
		print("> ")
		src, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		l := core.NewLexer(src)
		tokens, err := l.Tokenize()
		if err != nil {
			log.Println(err)
			continue
		}

		p := core.NewParser(src, tokens)
		prog, err := p.Parse("REPL")
		if err != nil {
			var e core.FormatedError
			if errors.As(err, &e) {
				log.Print(e.Format())
			}
			continue
		}

		c.SetSource(src)
		if err = c.Compile(prog); err != nil {
			var e core.FormatedError
			if errors.As(err, &e) {
				log.Print(e.Format())
			}
			continue
		}

		vm.SetChunk(c.Chunk)
		for vm.Next() {
		}
	}
}

var cli struct {
	Debug bool `short:"D" name:"debug" help:"Enable debug mode."`

	Run     RunCmd     `cmd:"" name:"run" help:"Run program."`
	Compile CompileCmd `cmd:"" name:"compile" help:"Compile program to bytecode."`
	Repl    ReplCmd    `cmd:"" name:"repl" help:"Start a REPL loop."`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
