// 版权 @2019 凹语言 作者。保留所有权利。

package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/wa-lang/wa/internal/ast"
	"github.com/wa-lang/wa/internal/backends/compiler_c"
	"github.com/wa-lang/wa/internal/backends/compiler_llvm"
	"github.com/wa-lang/wa/internal/backends/compiler_wat"
	"github.com/wa-lang/wa/internal/backends/target_spec"
	"github.com/wa-lang/wa/internal/config"
	"github.com/wa-lang/wa/internal/format"
	"github.com/wa-lang/wa/internal/loader"
	"github.com/wa-lang/wa/internal/logger"
	"github.com/wa-lang/wa/internal/parser"
	"github.com/wa-lang/wa/internal/scanner"
	"github.com/wa-lang/wa/internal/ssa"
	"github.com/wa-lang/wa/internal/token"
	"github.com/wa-lang/wa/internal/waroot"
)

// 命令行选项
type Option struct {
	Debug      bool
	TargetArch string
	TargetOS   string
	Clang      string
}

// 命令行程序对象
type App struct {
	opt  Option
	path string
	src  string
}

// 构建命令行程序对象
func NewApp(opt *Option) *App {
	logger.Tracef(&config.EnableTrace_app, "opt: %+v", opt)

	p := &App{}
	if opt != nil {
		p.opt = *opt
	}
	if p.opt.Clang == "" {
		if runtime.GOOS == "windows" {
			p.opt.Clang, _ = exec.LookPath("clang.exe")
		} else {
			p.opt.Clang, _ = exec.LookPath("clang")
		}
		if p.opt.Clang == "" {
			p.opt.Clang = "clang"
		}
	}
	if p.opt.TargetOS == "" {
		p.opt.TargetOS = runtime.GOOS
	}
	if p.opt.TargetArch == "" {
		p.opt.TargetArch = runtime.GOARCH
	}

	return p
}

func (p *App) InitApp(name, pkgpath string, update bool) error {
	if name == "" {
		return fmt.Errorf("init failed: <%s> is empty", name)
	}

	if !update {
		if _, err := os.Lstat(name); err == nil {
			return fmt.Errorf("init failed: <%s> exists", name)
		}
	}

	var info = struct {
		Name    string
		Pkgpath string
		Year    int
	}{
		Name:    name,
		Pkgpath: pkgpath,
		Year:    time.Now().Year(),
	}

	appFS := waroot.GetExampleAppFS()
	err := fs.WalkDir(appFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}

		data, err := fs.ReadFile(appFS, path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(path).Parse(string(data))
		if err != nil {
			return err
		}

		dstpath := filepath.Join(name, path)
		os.MkdirAll(filepath.Dir(dstpath), 0777)

		f, err := os.Create(dstpath)
		if err != nil {
			return err
		}
		defer f.Close()

		err = tmpl.Execute(f, &info)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	vendorFS := waroot.GetExampleVendorFS()
	err = fs.WalkDir(vendorFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}

		data, err := fs.ReadFile(vendorFS, path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(path).Parse(string(data))
		if err != nil {
			return err
		}

		dstpath := filepath.Join(name, "vendor", path)
		os.MkdirAll(filepath.Dir(dstpath), 0777)

		f, err := os.Create(dstpath)
		if err != nil {
			return err
		}
		defer f.Close()

		err = tmpl.Execute(f, &info)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *App) Fmt(path string) error {
	if path == "" {
		path, _ = os.Getwd()
	}

	if strings.HasSuffix(path, "...") {
		panic("TODO: fmt dir/...")
	}

	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		panic("TODO: fmt dir")
	}

	code, err := format.File(nil, path, nil)
	if err != nil {
		return err
	}

	return os.WriteFile(path, code, 0666)
}

func (p *App) Lex(filename string) error {
	src, err := p.readSource(filename, nil)
	if err != nil {
		return err
	}

	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile(filename, fset.Base(), len(src))
	s.Init(file, src, nil, scanner.ScanComments)

	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		fmt.Printf("%s\t%s\t%q\n", fset.Position(pos), tok, lit)
	}

	return nil
}

func (p *App) AST(filename string) error {
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(nil, fset, filename, nil, 0)
	if err != nil {
		return err
	}

	return ast.Print(fset, f)
}

func (p *App) SSA(filename string) error {
	cfg := config.DefaultConfig()
	prog, err := loader.LoadProgram(cfg, filename)
	if err != nil {
		return err
	}

	prog.SSAMainPkg.WriteTo(os.Stdout)

	var funcNames []string
	for name, x := range prog.SSAMainPkg.Members {
		if _, ok := x.(*ssa.Function); ok {
			funcNames = append(funcNames, name)
		}
	}
	sort.Strings(funcNames)
	for _, s := range funcNames {
		prog.SSAMainPkg.Func(s).WriteTo(os.Stdout)
	}

	return nil
}

func (p *App) CIR(filename string) error {
	cfg := config.DefaultConfig()
	prog, err := loader.LoadProgram(cfg, filename)
	if err != nil {
		return err
	}

	var c compiler_c.CompilerC
	c.CompilePackage(prog.SSAMainPkg)
	print("\n\n")
	print(c.String())

	return nil
}

func (p *App) LLVM(infile string, outfile string, target string) error {
	cfg := config.DefaultConfig()
	prog, err := loader.LoadProgram(cfg, infile)
	if err != nil {
		return err
	}
	output, err := compiler_llvm.New(target).Compile(prog)
	if err != nil {
		return err
	}
	if err := os.WriteFile(infile+".ll", []byte(output), 0644); err != nil {
		return err
	}

	// Invoke command `llc infile.ll -mtriple=xxx`.
	llc := []string{infile + ".ll"}
	if target != "" {
		llc = append(llc, "-mtriple", target)
	}
	cmd0 := exec.Command("llc", llc...)
	cmd0.Stderr = os.Stderr
	if err := cmd0.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "**** failed to invoke LLVM ****\n")
		return err
	}

	// Invoke command `clang infile.s -o outfile --target=xxx`.
	clang := []string{infile + ".s", "-static", "-o", outfile}
	if target != "" {
		clang = append(clang, "-target", target)
	}
	cmd1 := exec.Command("clang", clang...)
	cmd1.Stderr = os.Stderr
	if err := cmd1.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "**** failed to invoke CLANG ****\n")
		return err
	}

	return nil
}

func (p *App) WASM(filename string, target target_spec.Machine) ([]byte, error) {
	cfg := config.DefaultConfig()
	prog, err := loader.LoadProgram(cfg, filename)
	if err != nil {
		return nil, err
	}

	output, err := compiler_wat.New().Compile(prog, target)
	if err != nil {
		return nil, err
	}

	return []byte(output), nil
}

func (p *App) readSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch s := src.(type) {
		case string:
			return []byte(s), nil
		case []byte:
			return s, nil
		case *bytes.Buffer:
			if s != nil {
				return s.Bytes(), nil
			}
		case io.Reader:
			d, err := io.ReadAll(s)
			return d, err
		}
		return nil, errors.New("invalid source")
	}

	d, err := os.ReadFile(filename)
	return d, err
}

func (p *App) isWaFile(path string) bool {
	if fi, err := os.Lstat(path); err == nil && fi.Mode().IsRegular() {
		return strings.HasSuffix(strings.ToLower(path), ".wa")
	}
	return false
}
