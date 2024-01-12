package vdtgen

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

func traverseArrayType(arrayType *ast.ArrayType) (idents []string) {
	if arrayType.Elt == nil {
		panic("in here")
	} else {
		switch elt := arrayType.Elt.(type) {
		case *ast.Ident:
			if arrayType.Len != nil {
				idents = append(
					idents,
					fmt.Sprintf(
						"[%s]%s",
						arrayType.Len.(*ast.BasicLit).Value,
						arrayType.Elt.(*ast.Ident).Name),
				)
			} else {
				idents = append(
					idents,
					fmt.Sprintf(
						"[]%s",
						arrayType.Elt.(*ast.Ident).Name),
				)
			}
		case *ast.ArrayType:
			children := traverseArrayType(elt)
			for i, child := range children {
				if arrayType.Len != nil {
					children[i] = fmt.Sprintf("[%s]%s", arrayType.Len.(*ast.BasicLit).Value, child)
				} else {
					children[i] = fmt.Sprintf("[]%s", child)
				}
			}
			idents = append(idents, children...)
		}

	}
	return idents
}

func parseIdent(ident *ast.Ident) (idents []string) {
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
			case *ast.SelectorExpr:
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
	return
}

func traverseBinaryExpr(binaryExpr *ast.BinaryExpr) (idents []string) {
	for _, curr := range []ast.Expr{binaryExpr.X, binaryExpr.Y} {
		switch expr := curr.(type) {
		case *ast.BinaryExpr:
			idents = append(idents, traverseBinaryExpr(expr)...)
		case *ast.ArrayType:
			idents = append(idents, traverseArrayType(expr)...)
		case *ast.StarExpr:
			idents = append(
				idents,
				fmt.Sprintf(
					"*%s",
					expr.X.(*ast.Ident).Name),
			)
		case *ast.Ident:
			idents = append(idents, parseIdent(expr)...)
		}
	}
	return idents
}

func generateCode(idents []string, vdtName string, typeName string) string {
	code := fmt.Sprintf(`
type %s struct {
	inner any
}

func set%s[Value %s](mvdt *%s, value Value) {
	mvdt.inner = value
}
`,
		vdtName, vdtName, typeName, vdtName)

	code = code + fmt.Sprintf(`
func (mvdt *%s) SetValue(value any) (err error) {
	switch value := value.(type) {`, vdtName)

	for _, ident := range idents {
		code = code + fmt.Sprintf(`
		case %s:
			set%s(mvdt, value)
			return
		`, ident, vdtName)
	}
	code = code + `
		default:
			return fmt.Errorf("unsupported type")
	}
}
`
	code = code + fmt.Sprintf(`
func (mvdt %s) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {`, vdtName)

	for i, ident := range idents {
		code = code + fmt.Sprintf(`
		case %s:
			return %d, mvdt.inner, nil
		`, ident, i)
	}

	code = code + `
		}
		return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
	}
`
	code = code + fmt.Sprintf(`
func (mvdt %s) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}`,
		vdtName)

	code = code + fmt.Sprintf(`
func (mvdt %s) ValueAt(index uint) (value any, err error) {
	switch index {`,
		vdtName)

	for i, ident := range idents {
		code = code + fmt.Sprintf(`
		case %d:
			return *new(%s), nil
		`, i, ident)
	}

	code = code + `
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}`

	formatted, err := format.Source([]byte(code))
	if err != nil {
		panic(err)
	}

	return string(formatted)
}

func run(typeName string, src string) {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "dummy.go", src, parser.ParseComments)

	idents := make([]string, 0)

	ast.Inspect(f, func(n ast.Node) bool {
		// Called recursively.
		switch n := n.(type) {
		case *ast.TypeSpec:
			if n.Name.Name != typeName {
				break
			}
			// ast.Print(fset, n)
			inType, ok := n.Type.(*ast.InterfaceType)
			if !ok {
				panic("not an interface type")
			}
			field := inType.Methods.List[0]

			switch expr := field.Type.(type) {
			case *ast.BinaryExpr:
				idents = traverseBinaryExpr(expr)
			case *ast.Ident:
				idents = parseIdent(expr)
			}
			// fmt.Println(idents)
		}
		return true
	})

	if len(idents) == 0 {
		panic(fmt.Errorf("no interface named %s found", typeName))
	}

	var vdtName string
	if strings.HasSuffix(typeName, "Values") {
		vdtName = strings.TrimSuffix(typeName, "Values")
	} else if strings.HasSuffix(typeName, "s") {
		vdtName = strings.TrimSuffix(typeName, "s")
	} else {
		vdtName = typeName + "VaryingDataType"
	}

	formatted := generateCode(idents, vdtName, typeName)
	fmt.Println(formatted)
}

// var src = `
// package hello

// type myStruct struct{}

// type someOtherOtherconstraint interface {
// 	[]byte | [4]byte | rune | *uint | *myStruct | [][]myStruct | [][4]byte | [4][]myStruct
// }

// type someOtherConstraint interface {
// 	uint64 | uint32 | bool | someOtherOtherconstraint
// }

// type customInt64 int64

// type customAny any

// type VDTValues interface {
// 	int | string | myStruct | customInt64 | someOtherConstraint
// }
// `

var src = `
// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

type BabeDigestValues interface {
	BabePrimaryPreDigest | BabeSecondaryPlainPreDigest | BabeSecondaryVRFPreDigest
}

// NewBabeDigest returns a new VaryingDataType to represent a BabeDigest
// func NewBabeDigest() scale.VaryingDataType {
// 	return scale.MustNewVaryingDataType(BabePrimaryPreDigest{}, BabeSecondaryPlainPreDigest{}, BabeSecondaryVRFPreDigest{})
// }

// DecodeBabePreDigest decodes the input into a BabePreRuntimeDigest
// func DecodeBabePreDigest(in []byte) (scale.VaryingDataTypeValue, error) {
// 	babeDigest := NewBabeDigest()
// 	err := scale.Unmarshal(in, &babeDigest)
// 	if err != nil {
// 		return nil, err
// 	}

// 	babeDigestValue, err := babeDigest.Value()
// 	if err != nil {
// 		return nil, fmt.Errorf("getting babe digest value: %w", err)
// 	}
// 	switch msg := babeDigestValue.(type) {
// 	case BabePrimaryPreDigest, BabeSecondaryPlainPreDigest, BabeSecondaryVRFPreDigest:
// 		return msg, nil
// 	}

// 	return nil, errors.New("cannot decode data with invalid BABE pre-runtime digest type")
// }

// BabePrimaryPreDigest as defined in Polkadot RE Spec, definition 5.10 in section 5.1.4
type BabePrimaryPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
	VRFOutput      [sr25519.VRFOutputLength]byte
	VRFProof       [sr25519.VRFProofLength]byte
}

// NewBabePrimaryPreDigest returns a new BabePrimaryPreDigest
func NewBabePrimaryPreDigest(authorityIndex uint32,
	slotNumber uint64, vrfOutput [sr25519.VRFOutputLength]byte,
	vrfProof [sr25519.VRFProofLength]byte) *BabePrimaryPreDigest {
	return &BabePrimaryPreDigest{
		VRFOutput:      vrfOutput,
		VRFProof:       vrfProof,
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// ToPreRuntimeDigest returns the BabePrimaryPreDigest as a PreRuntimeDigest
// func (d BabePrimaryPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
// 	return toPreRuntimeDigest(d)
// }

// Index returns VDT index
func (BabePrimaryPreDigest) Index() uint { return 1 }

func (d BabePrimaryPreDigest) String() string {
	return fmt.Sprintf("BabePrimaryPreDigest{AuthorityIndex=%d, SlotNumber=%d, "+
		"VRFOutput=0x%x, VRFProof=0x%x}",
		d.AuthorityIndex, d.SlotNumber, d.VRFOutput, d.VRFProof)
}

// BabeSecondaryPlainPreDigest is included in a block built by a secondary slot authorized producer
type BabeSecondaryPlainPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
}

// NewBabeSecondaryPlainPreDigest returns a new BabeSecondaryPlainPreDigest
func NewBabeSecondaryPlainPreDigest(authorityIndex uint32, slotNumber uint64) *BabeSecondaryPlainPreDigest {
	return &BabeSecondaryPlainPreDigest{
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// ToPreRuntimeDigest returns the BabeSecondaryPlainPreDigest as a PreRuntimeDigest
// func (d BabeSecondaryPlainPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
// 	return toPreRuntimeDigest(d)
// }

// Index returns VDT index
func (BabeSecondaryPlainPreDigest) Index() uint { return 2 }

func (d BabeSecondaryPlainPreDigest) String() string {
	return fmt.Sprintf("BabeSecondaryPlainPreDigest{AuthorityIndex=%d, SlotNumber: %d}",
		d.AuthorityIndex, d.SlotNumber)
}

// BabeSecondaryVRFPreDigest is included in a block built by a secondary slot authorized producer
type BabeSecondaryVRFPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
	VrfOutput      [sr25519.VRFOutputLength]byte
	VrfProof       [sr25519.VRFProofLength]byte
}

// NewBabeSecondaryVRFPreDigest returns a new NewBabeSecondaryVRFPreDigest
func NewBabeSecondaryVRFPreDigest(authorityIndex uint32,
	slotNumber uint64, vrfOutput [sr25519.VRFOutputLength]byte,
	vrfProof [sr25519.VRFProofLength]byte) *BabeSecondaryVRFPreDigest {
	return &BabeSecondaryVRFPreDigest{
		VrfOutput:      vrfOutput,
		VrfProof:       vrfProof,
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// ToPreRuntimeDigest returns the BabeSecondaryVRFPreDigest as a PreRuntimeDigest
// func (d BabeSecondaryVRFPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
// 	return toPreRuntimeDigest(d)
// }

// Index returns VDT index
func (BabeSecondaryVRFPreDigest) Index() uint { return 3 }

func (d BabeSecondaryVRFPreDigest) String() string {
	return fmt.Sprintf("BabeSecondaryVRFPreDigest{AuthorityIndex=%d, SlotNumber=%d, "+
		"VrfOutput=0x%x, VrfProof=0x%x",
		d.AuthorityIndex, d.SlotNumber, d.VrfOutput, d.VrfProof)
}

// toPreRuntimeDigest returns the VaryingDataTypeValue as a PreRuntimeDigest
// func toPreRuntimeDigest(value scale.VaryingDataTypeValue) (*PreRuntimeDigest, error) {
// 	digest := NewBabeDigest()
// 	err := digest.Set(value)
// 	if err != nil {
// 		return nil, fmt.Errorf("cannot set varying data type value to babe digest: %w", err)
// 	}

// 	enc, err := scale.Marshal(digest)
// 	if err != nil {
// 		return nil, fmt.Errorf("cannot marshal babe digest: %w", err)
// 	}

// 	return NewBABEPreRuntimeDigest(enc), nil
// }

`
