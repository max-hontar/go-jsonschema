package codegen

import (
	"sort"
	"strings"

	"github.com/sanity-io/litter"
)

type Decl interface {
	Generate(out *Emitter)
}

type Named interface {
	Decl
	GetName() string
}

type File struct {
	FileName string
	Package  Package
}

func (p *File) Generate(out *Emitter) {
	out.Comment("Code generated by github.com/max-hontar/go-jsonschema, DO NOT EDIT.")
	out.Newline()
	p.Package.Generate(out)
}

// Package is a "package <name>; <body>".
type Package struct {
	QualifiedName string
	Comment       string
	Decls         []Decl
	Imports       []Import
}

func (p *Package) AddDecl(t Decl) {
	p.Decls = append(p.Decls, t)
}

func (p *Package) AddImport(qualifiedName, alias string) {
	if !p.hasImport(qualifiedName) {
		p.Imports = append(p.Imports, Import{
			QualifiedName: qualifiedName,
			Name:          alias,
		})
	}
}

func (p *Package) hasImport(q string) bool {
	for _, i := range p.Imports {
		if i.QualifiedName == q {
			return true
		}
	}
	return false
}

func (p *Package) Name() string {
	s := p.QualifiedName
	if i := strings.LastIndex(s, "/"); i != -1 && i < len(s)-1 {
		return s[i+1:]
	}
	return s
}

func (p *Package) Generate(out *Emitter) {
	out.Comment(p.Comment)
	out.Println("package %s", p.Name())
	if len(p.Imports) > 0 {
		for _, i := range p.Imports {
			i.Generate(out)
		}
	}
	out.Newline()

	sorted := make([]Decl, len(p.Decls))
	copy(sorted, p.Decls)
	sort.Slice(sorted, func(i, j int) bool {
		if a, ok := sorted[i].(Named); ok {
			if b, ok := sorted[j].(Named); ok {
				return a.GetName() < b.GetName()
			}
		}
		return false
	})
	for i, t := range sorted {
		if i > 0 {
			out.Newline()
		}
		t.Generate(out)
	}
}

// Var is a "var <name> = <value>".
type Var struct {
	Type  Type
	Name  string
	Value interface{}
}

func (v *Var) GetName() string {
	return v.Name
}

func (v *Var) Generate(out *Emitter) {
	out.Print("var %s ", v.Name)
	if v.Type != nil {
		v.Type.Generate(out)
	}
	out.Print(" = %s", litter.Sdump(v.Value))
}

// Constant is a "const <name> = <value>".
type Constant struct {
	Type  Type
	Name  string
	Value interface{}
}

func (c *Constant) GetName() string {
	return c.Name
}

func (c *Constant) Generate(out *Emitter) {
	out.Print("const %s ", c.Name)
	if c.Type != nil {
		c.Type.Generate(out)
	}
	out.Print(" = %s", litter.Sdump(c.Value))
}

// Fragment is an arbitary piece of code.
type Fragment func(*Emitter)

func (f Fragment) Generate(out *Emitter) {
	f(out)
}

// Method defines a method and how to generate it.
type Method struct {
	Impl func(*Emitter)
}

func (m *Method) Generate(out *Emitter) {
	out.Newline()
	m.Impl(out)
	out.Newline()
}

// Import is a "type <name> = <definition>".
type Import struct {
	Name          string
	QualifiedName string
}

func (i *Import) Generate(out *Emitter) {
	if i.Name != "" {
		out.Println("import %s %q", i.Name, i.QualifiedName)
	} else {
		out.Println("import %q", i.QualifiedName)
	}
}

// TypeDecl is a "type <name> = <definition>".
type TypeDecl struct {
	Name    string
	Type    Type
	Comment string
}

func (td *TypeDecl) GetName() string {
	return td.Name
}

func (td *TypeDecl) Generate(out *Emitter) {
	out.Comment(td.Comment)
	out.Print("type %s ", td.Name)
	td.Type.Generate(out)
	out.Newline()
}

type Type interface {
	Decl
	IsNillable() bool
}

type PointerType struct {
	Type Type
}

func (PointerType) IsNillable() bool { return true }

func (p PointerType) Generate(out *Emitter) {
	out.Print("*")
	p.Type.Generate(out)
}

type ArrayType struct {
	Type Type
}

func (ArrayType) IsNillable() bool { return true }

func (a ArrayType) Generate(out *Emitter) {
	out.Print("[]")
	a.Type.Generate(out)
}

type NamedType struct {
	Package *Package
	Decl    *TypeDecl
}

func (t NamedType) GetName() string {
	return t.Decl.Name
}

func (t NamedType) IsNillable() bool {
	return t.Decl.Type != nil && t.Decl.Type.IsNillable()
}

func (t NamedType) Generate(out *Emitter) {
	if t.Package != nil {
		out.Print(t.Package.Name())
		out.Print(".")
	}
	out.Print(t.Decl.Name)
}

type PrimitiveType struct {
	Type string
}

func (PrimitiveType) IsNillable() bool { return false }

func (p PrimitiveType) Generate(out *Emitter) {
	out.Print(p.Type)
}

type CustomNameType struct {
	Type string
}

func (CustomNameType) IsNillable() bool { return false }

func (p CustomNameType) Generate(out *Emitter) {
	out.Print(p.Type)
}

type MapType struct {
	KeyType, ValueType Type
}

func (MapType) IsNillable() bool { return true }

func (p MapType) Generate(out *Emitter) {
	out.Print("map[")
	p.KeyType.Generate(out)
	out.Print("]")
	p.ValueType.Generate(out)
}

type EmptyInterfaceType struct{}

func (EmptyInterfaceType) IsNillable() bool { return true }

func (EmptyInterfaceType) Generate(out *Emitter) {
	out.Print("interface{}")
}

type NullType struct{}

func (NullType) IsNillable() bool { return true }

func (NullType) Generate(out *Emitter) {
	out.Print("interface{}")
}

type StructType struct {
	Fields             []StructField
	RequiredJSONFields []string
}

func (StructType) IsNillable() bool { return false }

func (s *StructType) AddField(f StructField) {
	s.Fields = append(s.Fields, f)
}

func (s *StructType) Generate(out *Emitter) {
	out.Println("struct {")
	out.Indent(1)
	i := 0
	for _, f := range s.Fields {
		if i > 0 {
			out.Newline()
		}
		f.Generate(out)
		out.Newline()
		i++
	}
	out.Indent(-1)
	out.Print("}")
}

type StructField struct {
	Name         string
	Type         Type
	Comment      string
	Tags         string
	JSONName     string
	DefaultValue interface{}
}

func (f *StructField) GetName() string {
	return f.Name
}

func (f *StructField) Generate(out *Emitter) {
	out.Comment(f.Comment)
	out.Print("%s ", f.Name)
	f.Type.Generate(out)
	if f.Tags != "" {
		out.Print(" `%s`", f.Tags)
	}
}
