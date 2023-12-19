package vdtgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

func traverseBinaryExpr(binaryExpr *ast.BinaryExpr) (idents []string) {
	for _, curr := range []ast.Expr{binaryExpr.X, binaryExpr.Y} {
		switch expr := curr.(type) {
		case *ast.BinaryExpr:
			idents = append(idents, traverseBinaryExpr(expr)...)
		case *ast.ArrayType:
			if expr.Len != nil {
				idents = append(
					idents,
					fmt.Sprintf(
						"[%s]%s",
						expr.Len.(*ast.BasicLit).Value,
						expr.Elt.(*ast.Ident).Name),
				)
			} else {
				idents = append(
					idents,
					fmt.Sprintf(
						"[]%s",
						expr.Elt.(*ast.Ident).Name),
				)
			}
		case *ast.StarExpr:
			idents = append(
				idents,
				fmt.Sprintf(
					"*%s",
					expr.X.(*ast.Ident).Name),
			)
		case *ast.Ident:
			ident := expr
			switch ident.Obj {
			case nil:
				// primitive types
				idents = append(idents, ident.Name)
			default:
				switch decl := ident.Obj.Decl.(type) {
				case *ast.TypeSpec:
					ts := decl
					switch t := ts.Type.(type) {
					case *ast.Ident:
						if t.Name == "any" {
							break
						}
						idents = append(idents, ts.Name.Name)
					case *ast.StructType:
						idents = append(idents, ts.Name.Name)
					case *ast.InterfaceType:
						if len(t.Methods.List) != 1 {
							break
						}
						field := t.Methods.List[0]
						switch t := field.Type.(type) {
						case *ast.BinaryExpr:
							idents = append(idents, traverseBinaryExpr(t)...)
						}
					}

				}
			}
		}
	}
	return idents
}

func run() {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "dummy.go", src, parser.ParseComments)

	typeName := "VDTValues"

	ast.Inspect(f, func(n ast.Node) bool {
		// Called recursively.
		switch n := n.(type) {
		case *ast.TypeSpec:
			if n.Name.Name != typeName {
				break
			}
			ast.Print(fset, n)
			inType := n.Type.(*ast.InterfaceType)
			field := inType.Methods.List[0]

			expr := field.Type.(*ast.BinaryExpr)
			idents := traverseBinaryExpr(expr)
			fmt.Println(idents)
		}
		return true
	})
}

var src = `package hello

type myStruct struct{}

type someOtherOtherconstraint interface {
	[]byte | [4]byte | rune | *uint | *myStruct | [][]myStruct
}

type someOtherConstraint interface {
	uint64 | uint32 | bool | someOtherOtherconstraint
}

type customInt64 int64

type customAny any

type VDTValues interface {
	int | string | myStruct | customInt64 | someOtherConstraint
}
`

type myStruct struct{}

type someOtherOtherconstraint interface {
	[]byte | [4]byte | rune | *uint | *myStruct
}

type someOtherConstraint interface {
	uint64 | uint32 | bool | someOtherOtherconstraint
}

type customInt64 int64

type customAny any

type VDTValues interface {
	// int | string | myStruct | customInt64 | ~uint | customAny | someOtherConstraint
	int | string | myStruct | customInt64 | someOtherConstraint
}
