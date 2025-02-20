package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/asdine/genji/value"
)

const recordsTmpl = `
{{- define "records" }}
  {{- range .Records }}
    {{- template "record" . }}
  {{- end }}
{{- end }}
`

const recordTmpl = `
{{- define "record" }}
{{- template "record-GetField" . }}
{{- template "record-Iterate" . }}
{{- template "record-ScanRecord" . }}
{{- template "record-Scan" . }}
{{- template "record-Pk" . }}
{{- end }}
`

const recordGetFieldTmpl = `
{{ define "record-GetField" }}
{{- $fl := .FirstLetter -}}
{{- $structName := .Name -}}

// GetField implements the field method of the record.Record interface.
func ({{$fl}} *{{$structName}}) GetField(name string) (record.Field, error) {
	switch name {
	{{- range .Fields }}
	case "{{.Name}}":
		return record.New{{.Type}}Field("{{.Name}}", {{$fl}}.{{.Name}}), nil
	{{- end}}
	}

	return record.Field{}, errors.New("unknown field")
}
{{ end }}
`

const recordIterateTmpl = `
{{ define "record-Iterate" }}
{{- $fl := .FirstLetter -}}
{{- $structName := .Name -}}

// Iterate through all the fields one by one and pass each of them to the given function.
// It the given function returns an error, the iteration is interrupted.
func ({{$fl}} *{{$structName}}) Iterate(fn func(record.Field) error) error {
	var err error

	{{range .Fields}}
	err = fn(record.New{{.Type}}Field("{{.Name}}", {{$fl}}.{{.Name}}))
	if err != nil {
		return err
	}
	{{end}}

	return nil
}
{{ end }}
`

const recordScanRecordTmpl = `
{{ define "record-ScanRecord" }}
{{- $fl := .FirstLetter -}}
{{- $structName := .Name -}}

// ScanRecord extracts fields from record and assigns them to the struct fields.
// It implements the record.Scanner interface.
func ({{$fl}} *{{$structName}}) ScanRecord(rec record.Record) error {
	return rec.Iterate(func(f record.Field) error {
		var err error

		switch f.Name {
		{{- range .Fields}}
		case "{{.Name}}":
		{{$fl}}.{{.Name}}, err = f.DecodeTo{{.Type}}()
		{{- end}}
		}
		return err
	})
}
{{ end }}
`

const recordScanTmpl = `
{{ define "record-Scan" }}
{{- $fl := .FirstLetter -}}
{{- $structName := .Name -}}

// Scan extracts fields from src and assigns them to the struct fields.
// It implements the driver.Scanner interface.
func ({{$fl}} *{{$structName}}) Scan(src interface{}) error {
	r, ok := src.(record.Record)
	if !ok {
		return errors.New("unable to scan record from src")
	}

	return {{$fl}}.ScanRecord(r)
}
{{ end }}
`

const recordPkTmpl = `
{{ define "record-Pk" }}
{{- $fl := .FirstLetter -}}
{{- $structName := .Name -}}

{{- if ne .Pk.Name ""}}
// PrimaryKey returns the primary key. It implements the table.PrimaryKeyer interface.
func ({{$fl}} *{{$structName}}) PrimaryKey() ([]byte, error) {
	return value.Encode{{.Pk.Type}}({{$fl}}.{{.Pk.Name}}), nil
}
{{- end}}
{{ end }}
`

type recordContext struct {
	Name   string
	Fields []struct {
		Name, Type, GoType string
	}
	Pk struct {
		Name, Type, GoType string
	}
}

func (rctx *recordContext) lookupRecord(f *ast.File, target string) (bool, error) {
	for _, n := range f.Decls {
		gn, ok := ast.Node(n).(*ast.GenDecl)
		if !ok || gn.Tok != token.TYPE || len(gn.Specs) == 0 {
			continue
		}

		ts, ok := gn.Specs[0].(*ast.TypeSpec)
		if !ok {
			continue
		}

		if ts.Name.Name != target {
			continue
		}

		s, ok := ts.Type.(*ast.StructType)
		if !ok {
			return false, errors.New("invalid object")
		}

		rctx.Name = target

		for _, fd := range s.Fields.List {
			var typeName string

			typ, ok := fd.Type.(*ast.Ident)
			if !ok {
				atyp, ok := fd.Type.(*ast.ArrayType)
				if !ok {
					return false, errors.New("struct must only contain supported fields")
				}

				typ, ok = atyp.Elt.(*ast.Ident)
				if !ok || typ.Name != "byte" {
					return false, errors.New("struct must only contain supported fields")
				}

				typeName = "[]byte"
			} else {
				typeName = typ.Name
			}

			if len(fd.Names) == 0 {
				return false, errors.New("embedded fields are not supported")
			}

			if value.TypeFromGoType(typeName) == 0 {
				return false, fmt.Errorf("unsupported type %s", typeName)
			}

			for _, name := range fd.Names {
				rctx.Fields = append(rctx.Fields, struct {
					Name, Type, GoType string
				}{
					name.String(), value.TypeFromGoType(typeName).String(), typeName,
				})
			}

			if fd.Tag != nil {
				err := handleGenjiTag(rctx, fd)
				if err != nil {
					return false, err
				}
			}
		}

		return true, nil
	}

	return false, nil
}

func (rctx *recordContext) IsExported() bool {
	return unicode.IsUpper(rune(rctx.Name[0]))
}

func (rctx *recordContext) FirstLetter() string {
	return strings.ToLower(rctx.Name[0:1])
}

func (rctx *recordContext) UnexportedName() string {
	if !rctx.IsExported() {
		return rctx.Name
	}

	return rctx.Unexport(rctx.Name)
}

func (rctx *recordContext) ExportedName() string {
	if rctx.IsExported() {
		return rctx.Name
	}

	return rctx.Export(rctx.Name)
}

func (rctx *recordContext) NameWithPrefix(prefix string) string {
	n := prefix + rctx.ExportedName()
	if rctx.IsExported() {
		return rctx.Export(n)
	}

	return rctx.Unexport(n)
}

func (rctx *recordContext) Export(n string) string {
	name := []byte(n)
	name[0] = byte(unicode.ToUpper(rune(n[0])))
	return string(name)
}

func (rctx *recordContext) Unexport(n string) string {
	name := []byte(n)
	name[0] = byte(unicode.ToLower(rune(n[0])))
	return string(name)
}

func handleGenjiTag(ctx *recordContext, fd *ast.Field) error {
	unquoted, err := strconv.Unquote(fd.Tag.Value)
	if err != nil {
		return err
	}

	v, ok := reflect.StructTag(unquoted).Lookup("genji")
	if !ok {
		return nil
	}

	gtags := strings.Split(v, ",")

	for _, gtag := range gtags {
		switch gtag {
		case "pk":
			if ctx.Pk.Name != "" {
				return errors.New("only one pk field is allowed")
			}

			ctx.Pk.Name = fd.Names[0].Name
			ctx.Pk.Type = value.TypeFromGoType(fd.Type.(*ast.Ident).Name).String()
			ctx.Pk.GoType = fd.Type.(*ast.Ident).Name
		default:
			return fmt.Errorf("unsupported genji tag '%s'", gtag)
		}
	}

	return nil
}
