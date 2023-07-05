// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/loader"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	alib "go.opentelemetry.io/contrib/instrgen/lib"
	"go.opentelemetry.io/contrib/instrgen/rewriters"
)

const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
)

func usage() {
	fmt.Printf(InfoColor, "\nusage driver --command [file pattern] replace entrypoint")
	fmt.Println()
	fmt.Printf(InfoColor, "\tcommand:")
	fmt.Println()
	fmt.Printf(InfoColor, "\t\tinject                                 (injects open telemetry calls into project code)")
	fmt.Println()
	fmt.Printf(InfoColor, "\t\tprune                                  (prune open telemetry calls")
	fmt.Println()
}

// Entry point function.
type EntryPoint struct {
	Pkg     string
	FunName string
}

// Command passed to the compiler toolchain.
type InstrgenCmd struct {
	ProjectPath string
	FilePattern string
	Cmd         string
	Replace     string
	EntryPoint  EntryPoint
}

// CommandExecutor.
type CommandExecutor interface {
	Execute(cmd string, args []string)
	Run() error
}

// ToolExecutor.
type ToolExecutor struct {
	cmd *exec.Cmd
}

// Wraps Execute.
func (executor *ToolExecutor) Execute(cmd string, args []string) {
	executor.cmd = exec.Command(cmd, args...)
	executor.cmd.Stdin = os.Stdin
	executor.cmd.Stdout = os.Stdout
	executor.cmd.Stderr = os.Stderr
}

// Wraps Run.
func (executor *ToolExecutor) Run() error {
	return executor.cmd.Run()
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), err
}

func LoadProgram(projectPath string, ginfo *types.Info) (*loader.Program, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	conf := loader.Config{ParserMode: parser.ParseComments}
	conf.Build = &build.Default
	conf.Build.CgoEnabled = false
	conf.Build.Dir = filepath.Join(cwd, projectPath)
	conf.Import(projectPath)
	var mutex = &sync.RWMutex{}
	conf.AfterTypeCheck = func(info *loader.PackageInfo, files []*ast.File) {
		for k, v := range info.Defs {
			mutex.Lock()
			ginfo.Defs[k] = v
			mutex.Unlock()
		}
		for k, v := range info.Uses {
			mutex.Lock()
			ginfo.Uses[k] = v
			mutex.Unlock()
		}
		for k, v := range info.Selections {
			mutex.Lock()
			ginfo.Selections[k] = v
			mutex.Unlock()
		}
	}
	return conf.Load()
}

func executeCommand(command string, projectPath string, packagePattern string, replaceSource string, entryPoint string, executor CommandExecutor) error {
	isDir, err := isDirectory(projectPath)
	if !isDir {
		return errors.New("[path to go project] argument must be directory")
	}
	if err != nil {
		return err
	}
	if command == "--prune" {
		replaceSource = "yes"
	}

	switch command {
	case "--inject", "--prune":
		entry := strings.Split(entryPoint, ".")
		data := InstrgenCmd{projectPath, packagePattern, command[2:], replaceSource,
			EntryPoint{entry[0], entry[1]}}
		file, _ := json.MarshalIndent(data, "", " ")
		err := os.WriteFile("instrgen_cmd.json", file, 0644)
		if err != nil {
			return err
		}
		executor.Execute("go", []string{"build", "-work", "-a", "-toolexec", "driver"})
		//fmt.Println("invoke : " + executor.cmd.String())
		if err := executor.Run(); err != nil {
			return err
		}
		return nil
	default:
		return errors.New("unknown command")
	}
}

func checkArgs(args []string) error {
	if len(args) != 4 {
		return errors.New("wrong arguments")
	}
	return nil
}

func executePass(args []string, executor CommandExecutor) error {
	path := args[0]
	args = args[1:]
	executor.Execute(path, args)
	return executor.Run()
}

// GetCommandName extracts command name from args.
func GetCommandName(args []string) string {
	if len(args) == 0 {
		return ""
	}

	cmd := filepath.Base(args[0])
	if ext := filepath.Ext(cmd); ext != "" {
		cmd = strings.TrimSuffix(cmd, ext)
	}
	return cmd
}

func analyzePackage(rewriter alib.PackageRewriter,
	pkg string, filePaths map[string]int,
	trace *os.File, destPath string,
	args []string,
	remappedFilePaths map[string]string) []string {
	fset := token.NewFileSet()
	// TODO handle trace
	_ = trace
	extraFilesWritten := false

	removedFilePaths := make(map[string]int)
	for filePath, index := range filePaths {
		trace.WriteString(rewriter.Id() + ":" + filePath)
		trace.WriteString("\n")
		file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		if rewriter.Inject(pkg, filePath) {
			rewriter.Rewrite(pkg, file, fset, trace)

			if rewriter.ReplaceSource(pkg, filePath) {
				var out *os.File
				out, err = alib.CreateFile(fset.File(file.Pos()).Name() + "tmp")
				if err != nil {
					continue
				}
				err = printer.Fprint(out, fset, file)
				if err != nil {
					continue
				}
				oldFileName := fset.File(file.Pos()).Name() + "tmp"
				newFileName := fset.File(file.Pos()).Name()
				err = os.Rename(oldFileName, newFileName)
				if err != nil {
					continue
				}
			} else {
				filename := filepath.Base(filePath)
				out, err := alib.CreateFile(destPath + "/" + filename + "tmp")

				if err != nil {
					continue
				}
				err = printer.Fprint(out, fset, file)
				if err != nil {
					continue
				}
				oldFileName := destPath + "/" + filename + "tmp"
				newFileName := destPath + "/" + filename
				err = os.Rename(oldFileName, newFileName)
				if err != nil {
					continue
				}
				out.Close()
				args[index] = destPath + "/" + filename
				removedFilePaths[filePath] = index
				remappedFilePaths[args[index]] = filePath
			}
			if !extraFilesWritten {
				files := rewriter.WriteExtraFiles(pkg, destPath)
				if len(files) > 0 {
					args = append(args, files...)
				}
				extraFilesWritten = true
			}
		}
	}
	for k, v := range removedFilePaths {
		delete(filePaths, k)
		filePaths[args[v]] = v
	}
	return args
}

func analyze(args []string, rewriterS []alib.PackageRewriter, remappedFilePaths map[string]string) []string {
	trace, _ := os.OpenFile("args", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	argsLen := len(args)
	var destPath string
	var pkg string

	for i, a := range args {
		// output directory
		if a == "-o" {
			destPath = filepath.Dir(string(args[i+1]))
		}
		// package
		if a == "-p" {
			pkg = string(args[i+1])
		}
		// source files
		if a == "-pack" {
			files := make(map[string]int)
			for j := i + 1; j < argsLen; j++ {
				// omit -asmhdr switch + following header+
				if string(args[j]) == "-asmhdr" {
					j = j + 2
				}
				if !strings.HasSuffix(args[j], ".go") {
					continue
				}
				filePath := args[j]
				files[filePath] = j
			}
			for _, rewriter := range rewriterS {
				args = analyzePackage(rewriter, pkg, files, trace, destPath, args, remappedFilePaths)
			}
		}
	}
	return args
}

func toolExecMain(args []string, rewriterS []alib.PackageRewriter, executor CommandExecutor, remappedFilePaths map[string]string) error {
	args = analyze(args, rewriterS, remappedFilePaths)
	if len(args) == 0 {
		usage()
		return errors.New("wrong command")
	}

	err := executePass(args[0:], executor)
	if err != nil {
		return err
	}
	return nil
}

func printStack(stack []*ast.CallExpr) {
	for len(stack) > 0 {
		n := len(stack) - 1 // Top element
		if sel, ok := stack[n].Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				fmt.Print(ident.Name)
			}
			fmt.Print(".")
			fmt.Print(sel.Sel.Name)
		}
		stack = stack[:n] // Pop
	}
}

func makeRewriters(instrgenCfg InstrgenCmd, remappedFilePaths map[string]string) []alib.PackageRewriter {
	var rewriterS []alib.PackageRewriter
	switch instrgenCfg.Cmd {
	case "inject":
		rewriterS = append(rewriterS, rewriters.RuntimeRewriter{
			FilePattern: instrgenCfg.FilePattern})
		rewriterS = append(rewriterS, rewriters.BasicRewriter{
			FilePattern: instrgenCfg.FilePattern, Replace: instrgenCfg.Replace,
			Pkg: instrgenCfg.EntryPoint.Pkg, Fun: instrgenCfg.EntryPoint.FunName, RemappedFilePaths: remappedFilePaths})
	case "prune":
		rewriterS = append(rewriterS, rewriters.OtelPruner{
			FilePattern: instrgenCfg.FilePattern, Replace: true})
	}
	return rewriterS
}

func goModTidy(projectPath string, replace string, prog *loader.Program, ginfo *types.Info) {
	for _, pkg := range prog.AllPackages {
		if len(pkg.Files) > 0 {
			path := prog.Fset.File(pkg.Files[0].Pos()).Name()
			if !strings.Contains(path, projectPath) {
				continue
			}
			if !alib.FileExists(filepath.Dir(path) + "/instrgen_imports.go") {
				f, err := alib.CreateFile(filepath.Dir(path) + "/instrgen_imports.go")
				if err != nil {
					fmt.Println(err)
					return
				}
				imports :=
					`package ` + pkg.Pkg.Name() + `
import (
	_ "go.opentelemetry.io/contrib/instrgen/rtlib"
	_ "go.opentelemetry.io/otel"
	_ "context"
	_ "runtime"
	_ "go.opentelemetry.io/otel/trace"
	_ "go.opentelemetry.io/otel/sdk/trace"
)
`
				_, err = f.WriteString(imports)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}
	executor := &ToolExecutor{}
	executor.Execute("go", []string{"mod", "tidy"})
	fmt.Printf(InfoColor, "invoke : "+executor.cmd.String()+"\n")
	if err := executor.Run(); err != nil {
		fmt.Println(err)
	}
}

func driverMain(args []string, executor CommandExecutor) error {
	cmdName := GetCommandName(args)
	if cmdName != "compile" {
		// do semantic check before injecting
		if cmdName == "--inject" {
			ginfo := &types.Info{
				Defs:       make(map[*ast.Ident]types.Object),
				Uses:       make(map[*ast.Ident]types.Object),
				Selections: make(map[*ast.SelectorExpr]*types.Selection),
			}
			fmt.Printf(InfoColor, "instrgen semantic analysis...\n")
			prog, err := LoadProgram(".", ginfo)
			if err != nil {
				err = errors.New("Load failed : " + err.Error())
				return err
			}
			goModTidy(args[1], args[2], prog, ginfo)
		}
		switch cmdName {
		case "--inject", "--prune":
			fmt.Printf(InfoColor, "instrgen compiler\n")
			err := checkArgs(args)
			if err != nil {
				usage()
				return err
			}
			replace := "no"
			if len(args) > 2 {
				replace = args[2]
			}
			err = executeCommand(args[0], ".", args[1], replace, args[3], executor)
			if err != nil {
				return err
			}
			return nil
		}
		if len(args) > 0 {
			err := executePass(args[0:], executor)
			if err != nil {
				return err
			}
		} else {
			usage()
		}
		return nil
	}
	content, err := os.ReadFile("./instrgen_cmd.json")
	if err != nil {
		return err
	}

	var instrgenCfg InstrgenCmd
	err = json.Unmarshal(content, &instrgenCfg)
	if err != nil {
		return err
	}
	remappedFilePaths := make(map[string]string)
	rewriterS := makeRewriters(instrgenCfg, remappedFilePaths)
	return toolExecMain(args, rewriterS, executor, remappedFilePaths)
}

func main() {
	executor := &ToolExecutor{}
	err := driverMain(os.Args[1:], executor)
	if err != nil {
		fmt.Println()
		fmt.Printf(ErrorColor, err.Error())
		fmt.Println()
	}
}
