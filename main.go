package main

import (
	"bytes"
	"fmt"
	"github.com/jinzhu/gorm"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"golang.org/x/tools/go/ast/inspector"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const implementation = "implementation"

var serviceTemplate = template.Must(template.New("").Parse(`
package implementation

import (
	"context"
	"log"
	
)

// {{ .ServiceName }}Service implements the {{ .ServicePackage }}.{{ .ServiceName }}
type {{ .ServiceName }}Service struct {
	logger *log.Logger
}

func New{{ .ServiceName }}Service(logger *log.Logger) {{ .ServicePackage }}.{{ .ServiceName }} {
	return &{{ .ServiceName }}Service{
		logger: logger,
	}
}

{{ range .Functions}}
// {{ .Name }} implements {{ $.ServicePackage }}.{{ $.ServiceName }}
func (s *{{ $.ServiceName }}Service){{ .Name }} {{ .Signature }} {

	panic("Not implemented yet")
}

{{end}}

`))

func expr2string(expr ast.Expr) string {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, token.NewFileSet(), expr)
	if err != nil {
		log.Fatalf("error print expression to string: #{err}")

	}
	return buf.String()
}

// Агрегатор данных для установки параметров в шаблоне
type serviceGenerator struct {
	fileIdent *ast.Ident
	typeSpec  *ast.TypeSpec
	methods   []*ast.Field
}

type serviceFunction struct {
	Name      string
	Signature string
}

func (r serviceGenerator) convertFunctions() []serviceFunction {
	ret := []serviceFunction{}
	for _, method := range r.methods {
		f := serviceFunction{
			Name:      method.Names[0].Name,
			Signature: strings.TrimPrefix(expr2string(method.Type), "func"),
		}
		ret = append(ret, f)
	}
	return ret
}

func (r serviceGenerator) Generate(outFile *ast.File) error {
	//Аллокация и установка параметров для template
	params := struct {
		ServiceName    string
		ServicePackage string
		Functions      []serviceFunction
	}{
		//Параметры извлекаем из ресивера метода
		r.typeSpec.Name.Name,
		r.fileIdent.Name,
		r.convertFunctions(),
	}

	//Аллокация буфера,
	//куда будем заливать выполненный шаблон
	var buf bytes.Buffer
	//Процессинг шаблона с подготовленными параметрами
	//в подготовленный буфер
	err := serviceTemplate.Execute(&buf, params)
	if err != nil {
		return fmt.Errorf("execute template: %v", err)
	}

	//Теперь сделаем парсинг обработанного шаблона,
	//который уже стал валидным кодом Go,
	//в дерево разбора,
	//получаем AST этого кода
	templateAst, err := parser.ParseFile(
		token.NewFileSet(),
		//Источник для парсинга лежит не в файле,
		"",
		//а в буфере
		buf.Bytes(),
		//mode парсинга, нас интересуют в основном комментарии
		parser.ParseComments,
	)
	if err != nil {
		return fmt.Errorf("parse template: %v", err)
	}

	//Добавляем декларации из полученного дерева
	//в результирующий outFile *ast.File,
	//переданный нам аргументом
	for _, decl := range templateAst.Decls {
		outFile.Decls = append(outFile.Decls, decl)
	}
	return nil
}

func main() {
	_ = gorm.DB{}
	//Цель генерации передаётся переменной окружения
	path := os.Getenv("GOFILE")
	if path == "" {
		path = "./calc/service.go"
	}
	//Разбираем целевой файл в AST
	astInFile, err := parser.ParseFile(
		token.NewFileSet(),
		path,
		nil,
		//Нас интересуют комментарии
		parser.ParseComments,
	)
	if err != nil {
		log.Fatalf("parse file: %v", err)
	}
	//Для выбора интересных нам деклараций
	//используем Inspector из golang.org/x/tools/go/ast/inspector
	i := inspector.New([]*ast.File{astInFile})
	//Подготовим фильтр для этого инспектора
	iFilter := []ast.Node{
		//Нас интересуют декларации
		&ast.GenDecl{},
	}
	//Выделяем список заданий генерации
	var genTasks []serviceGenerator
	//Запускаем инспектор с подготовленным фильтром
	//и литералом фильтрующей функции
	i.Nodes(iFilter, func(node ast.Node, push bool) (proceed bool) {
		genDecl := node.(*ast.GenDecl)
		//Код без комментариев не нужен,
		if genDecl.Doc == nil {
			return false
		}
		//интересуют спецификации типов,
		typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
		if !ok {
			return false
		}
		//а конкретно интерфейсы
		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			return false
		}
		//Из оставшегося
		for _, comment := range genDecl.Doc.List {
			switch comment.Text {
			//выделяем структуры, помеченные комментарием repogen:entity,
			case "//servicegen:service":
				//и добавляем в список заданий генерации
				genTasks = append(genTasks, serviceGenerator{
					fileIdent: astInFile.Name,
					typeSpec:  typeSpec,
					methods:   interfaceType.Methods.List,
				})
			}
		}
		return false
	})
	//Аллокация результирующего дерева разбора
	astOutFile := &ast.File{
		Name: &ast.Ident{
			Name: "implementation",
		},
	}

	//Запускаем список заданий генерации
	for _, task := range genTasks {
		//Для каждого задания вызываем написанный нами генератор
		//как метод этого задания
		//Сгенерированные декларации помещаются в результирующее дерево разбора
		err = task.Generate(astOutFile)
		if err != nil {
			log.Fatalf("generate: %v", err)
		}
	}

	//astOutFile.Name.Name = "implementation"

	err = os.Mkdir(implementation, 0660)
	if err != nil {
		log.Fatalf("create dir: %v", err)
	}

	filename := filepath.Base(path)
	//Подготовим файл конечного результата всей работы,
	//назовем его созвучно файлу модели, добавим только суффикс _gen
	outFile, err := os.Create(filepath.Join(implementation, strings.TrimSuffix(filename, ".go")+"_gen.go"))
	if err != nil {
		log.Fatalf("create file: %v", err)
	}
	//Не забываем прибраться
	defer outFile.Close()
	//Печатаем результирующий AST в результирующий файл исходного кода
	//«Печатаем» не следует понимать буквально,
	//дерево разбора нельзя просто переписать в файл исходного кода,
	//это совершенно разные форматы
	//Мы здесь воспользуемся специализированным принтером из пакета ast/printer
	err = printer.Fprint(outFile, token.NewFileSet(), astOutFile)
	if err != nil {
		log.Fatalf("print file: %v", err)
	}
}
