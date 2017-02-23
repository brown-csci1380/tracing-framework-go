package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/loader"
)

func TestGos(t *testing.T) {
	src := `
package main

func one(a int) {}

func two(a, b int) {}

func three(a, b, c int) {}

func variadic(a ...int) {}

func blank(_ int) {}

func nested(a int) {
	func(a int) {}(a)
}

func main() {
	go one(0)

	go two(0, 1)

	go three(0, 1, 2)

	go variadic()

	go variadic(0)

	go variadic(0, 1)

	go variadic([]int{1, 2, 3}...)

	go blank(0)

	go nested(0)
}
`

	expect := `package main

import xtr "github.com/brown-csci1380/tracing-framework-go/xtrace/client"

func one(a int) {}

func two(a, b int) {}

func three(a, b, c int) {}

func variadic(a ...int) {}

func blank(_ int) {}

func nested(a int) {
	func(a int) {}(a)
}

func main() {
	{
		arg0 :=

			0
		xtr.XGo(func() {

			one(arg0)
		})
	}
	{
		arg0 :=

			0
		arg1 := 1
		xtr.XGo(func() {
			two(arg0, arg1)

		})
	}
	{
		arg0 :=

			0
		arg1 := 1
		arg2 :=

			2
		xtr.XGo(func() {
			three(arg0, arg1, arg2)
		})
	}
	{
		xtr.
			XGo(func() {
				variadic()
			})
	}
	{
		arg0 :=

			0
		xtr.XGo(func() {

			variadic(arg0)
		})
	}
	{
		arg0 :=

			0
		arg1 := 1
		xtr.XGo(func() {
			variadic(arg0, arg1,
			)
		})
	}
	{
		arg0 :=

			[]int{1, 2, 3}
		xtr.
			XGo(func() {
				variadic(arg0...,
				)
			})
	}
	{
		arg0 :=

			0
		xtr.XGo(func() {

			blank(arg0)
		})
	}
	{
		arg0 :=

			0
		xtr.XGo(func() {

			nested(arg0)
		})
	}

}
`

	new := testHelper(t, src, rewriteGos)
	if new != expect {
		var idx int
		for i, c := range []byte(new) {
			if c != expect[i] {
				idx = i
				break
			}
		}
		t.Errorf("unexpected output (see source for expected output) at character %v:\n%v", idx, new)
	}
}

type rewriter func(fset *token.FileSet, info types.Info, qual types.Qualifier, f *ast.File) (bool, error)

func testHelper(t *testing.T, src string, r rewriter) string {
	c := loader.Config{
		Fset:        token.NewFileSet(),
		ParserMode:  parser.ParseComments | parser.DeclarationErrors,
		AllowErrors: true,
	}

	f, err := c.ParseFile("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.CreateFromFiles("main", f)
	p, err := c.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pi := p.Package("main")

	_, err = r(c.Fset, pi.Info, qualifierForFile(pi.Pkg, f), f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	err = format.Node(&buf, c.Fset, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return buf.String()
}
