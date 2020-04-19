package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"path"
	"reflect"
	"regexp"
	"strings"
)

type tpl struct {
	Name        string
	FindSQL     string
	DeleteSQL   string
	UpdateSQL   string
	CreateSQL   string
	ColumnCount int
	PlaceHolder string
	Value       string
	Scan        string
	Fields      []*tplField
}

type tplField struct {
	Name   string
	Type   string
	Column string
}

var (
	src    string
	suffix = "_gorm.go"
	name   string
)

func init() {
	flag.StringVar(&src, "src", ".", "-src=testdata/testdata.go")
	flag.StringVar(&name, "name", ".", "-name=User")
	flag.Parse()
}

func main() {
	if name == "" {
		flag.PrintDefaults()
		return
	}
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, src, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("failed to parse src:%s, err:%#v", src, err)
	}
	baseDir := path.Dir(src)
	p, err := build.ImportDir(baseDir, 0)
	if err != nil {
		return
	}
	// name := path.Base(src)

	buf := bytes.NewBuffer(nil)

	fullPath := strings.Replace(src, ".go", suffix, -1)
	var lastGen *ast.GenDecl

	var tpl *tpl
	ast.Walk(walker(func(node ast.Node) bool {
		switch v := node.(type) {
		case *ast.GenDecl:
			if v.Tok == token.IMPORT {
				return false
			}
			lastGen = v
			return true
		case *ast.TypeSpec:
			structName := v.Name.Name

			tableName := getTableName(structName, v.Doc, lastGen)
			lastGen = nil
			if tableName == "" {
				return false
			}
			if structName != name {
				return true
			}

			st, ok := v.Type.(*ast.StructType)
			if !ok {
				return true
			}
			tpl = gen(structName, tableName, buf, st)
			return false
		case *ast.ValueSpec:
			return false

		default:
			return true
		}
	}), file)

	if tpl == nil {
		return
	}

	importStr := fmt.Sprintf(`package %s
	import(
		"database/sql"
		"context"
		"fmt"
		"strings"
	)`, p.Name)
	io.WriteString(buf, importStr)

	t, err := template.New("gorm").Funcs(template.FuncMap{
		"raw": raw,
	}).Parse(tplStr)
	if err != nil {
		return
	}
	if err := t.Execute(buf, tpl); err != nil {
		return
	}
	if buf.Len() != 0 {
		ioutil.WriteFile(fullPath, buf.Bytes(), 0644)
	}
}

func getTableName(structName string, doc *ast.CommentGroup, lastGen *ast.GenDecl) string {
	if doc == nil && lastGen != nil {
		doc = lastGen.Doc
	}
	var comment string
	if doc != nil && len(doc.List) > 0 {
		comment = doc.List[0].Text
	}
	reg := regexp.MustCompile(fmt.Sprintf(`// %s (\w+)`, structName))
	subMatches := reg.FindStringSubmatch(comment)
	if len(subMatches) != 0 {
		return subMatches[1]
	}
	panic("no table comment")
}

func gen(structName, tableName string, buf *bytes.Buffer, st *ast.StructType) *tpl {
	fields := st.Fields.List
	if len(fields) == 0 {
		return nil
	}
	scan := make([]string, 0, len(fields))
	column := make([]string, 0, len(fields))
	value := make([]string, 0, len(fields))
	placeHolder := make([]string, 0, len(fields))
	tplFields := make([]*tplField, 0, len(fields))
	for _, field := range fields {
		if field.Tag == nil {
			continue
		}
		ident, ok := field.Type.(*ast.Ident)
		if !ok {
			continue
		}

		trimedValue := strings.Trim(field.Tag.Value, "`")
		curColumn := reflect.StructTag(trimedValue).Get("gorm")
		column = append(column, curColumn)
		name := field.Names[0].Name
		value = append(value, "obj."+name)
		scan = append(scan, `&result.`+name)
		placeHolder = append(placeHolder, "?")
		tplFields = append(tplFields, &tplField{
			Name:   name,
			Type:   ident.Name,
			Column: curColumn,
		})
	}
	return &tpl{
		Name:        structName,
		FindSQL:     fmt.Sprintf("select %s from %s", strings.Join(column, ","), tableName),
		DeleteSQL:   fmt.Sprintf("delete from %s", tableName),
		UpdateSQL:   fmt.Sprintf("update %s set", tableName),
		CreateSQL:   fmt.Sprintf("insert into %s(%s)", tableName, strings.Join(column, ",")),
		ColumnCount: len(column),
		PlaceHolder: strings.Join(placeHolder, ","),
		Value:       strings.Join(value, ","),
		Scan:        strings.Join(scan, ","),
		Fields:      tplFields,
	}

}

type walker func(ast.Node) bool

func (w walker) Visit(node ast.Node) ast.Visitor {
	if w(node) {
		return w
	}
	return nil
}

func raw(prev string) template.HTML {
	return template.HTML(prev)
}
