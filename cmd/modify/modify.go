// the "modify" command modifies an exisitng Go installation to support x-trace
// by adding goroutine-local variables and a way to access them.
// run "go run modify.go" and then "go install -a std"
package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"runtime"
)

var localFieldName = "local"

func modifyRuntime2dotGo() {
	goroot := runtime.GOROOT()
	path := goroot + "/src/runtime/runtime2.go"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse runtime2.go in ", path, ": ", err)
		os.Exit(1)
	}

	alreadyModified := false
	continueStepping := true

	ast.Inspect(f, func(n ast.Node) bool {
		typeDecl, ok := n.(*ast.TypeSpec)
		if ok {
			if typeDecl.Name.Name == "g" {
				fmt.Print("Found g struct...")
				gStruct := typeDecl.Type.(*ast.StructType)

				for _, field := range gStruct.Fields.List {
					for _, name := range field.Names {
						if name.Name == localFieldName {
							//local already exists
							fmt.Println("...already modified.")
							continueStepping = false
							alreadyModified = true
						}
					}
				}
				if !alreadyModified {
					docComment := &ast.Comment{Text: "\n// Goroutine-local storage"}

					gStruct.Fields.List = append(gStruct.Fields.List, &ast.Field{
						Names: []*ast.Ident{ast.NewIdent(localFieldName)},
						Type: &ast.InterfaceType{
							Methods: &ast.FieldList{},
						},
						Doc: &ast.CommentGroup{
							[]*ast.Comment{
								docComment,
							},
						},
					})

					fmt.Println("...Created modified goroutine structure:")
					printer.Fprint(os.Stdout, fset, gStruct)
					fmt.Println()
					continueStepping = false
				}
			} else {
				return false
			}
		}
		return continueStepping
	})

	if !alreadyModified {
		outfile, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open", path)
			os.Exit(1)
		}

		err = format.Node(outfile, fset, f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to write to", path)
			os.Exit(1)
		}
		outfile.Close()
	}
}

var localMethodsText = `package runtime

func GetLocal() interface{} {
    return getg().local
}

func SetLocal(local interface{}) {
    getg().local = local
}
`

func addMethodsToRuntime() {
	fmt.Print("Creating local access methods...")
	goroot, exists := os.LookupEnv("GOROOT")
	if !exists {
		goroot = "/usr/local/go"
	}
	path := goroot + "/src/runtime/" + localFieldName + ".go"
	newfile, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create", path, err)
		os.Exit(1)
	}
	_, err = newfile.WriteString(localMethodsText)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write to", path, err)
		os.Exit(1)
	}
	err = newfile.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save ", path, err)
		os.Exit(1)
	}
	fmt.Println("...Done")
}

func main() {
	modifyRuntime2dotGo()
	addMethodsToRuntime()
}
