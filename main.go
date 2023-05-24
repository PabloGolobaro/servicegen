package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pablogolobaro/servicegen/templates"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"golang.org/x/tools/go/ast/inspector"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const implementationPackage = "implementation"
const transportPackage = "transport"

func expr2string(expr ast.Expr) string {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, token.NewFileSet(), expr)
	if err != nil {
		log.Fatalf("error print expression to string: #{err}")

	}
	return buf.String()
}

// Агрегатор данных для установки параметров в шаблоне
type implementationGenerator struct {
	fileIdent *ast.Ident
	typeSpec  *ast.TypeSpec
	methods   []*ast.Field
}

type serviceFunction struct {
	Name      string
	Signature string
	Arguments []argument
}

type argument struct {
	Name string
	Type string
}

func extractArguments(spec ast.Expr) ([]argument, error) {
	ret := []argument{}
	funcType, ok := spec.(*ast.FuncType)
	if !ok {
		return nil, errors.New("type is not *ast.FuncType")
	}
	for _, param := range funcType.Params.List {
		switch paramType := param.Type.(type) {
		case *ast.Ident:
			for _, name := range param.Names {
				ret = append(ret, argument{Name: name.Name, Type: paramType.Name})
			}
		case *ast.SelectorExpr:
			x := paramType.X.(*ast.Ident)
			sel := paramType.Sel
			for _, name := range param.Names {
				if name.Name != "ctx" {
					ret = append(ret, argument{Name: name.Name, Type: fmt.Sprintf("%s.%s", x.Name, sel.Name)})
				}
			}
		}

	}
	return ret, nil
}

func (r implementationGenerator) convertFunctions() ([]serviceFunction, error) {
	ret := []serviceFunction{}
	for _, method := range r.methods {
		arguments, err := extractArguments(method.Type)
		if err != nil {
			return ret, err
		}
		f := serviceFunction{
			Name:      method.Names[0].Name,
			Signature: strings.TrimPrefix(expr2string(method.Type), "func"),
			Arguments: arguments,
		}
		ret = append(ret, f)
	}
	return ret, nil
}

func (r implementationGenerator) Generate(outFile *ast.File) error {
	//Аллокация и установка параметров для template

	serviceFunctions, err := r.convertFunctions()
	if err != nil {
		return err
	}
	params := struct {
		ServiceName    string
		ServicePackage string
		Functions      []serviceFunction
	}{
		//Параметры извлекаем из ресивера метода
		r.typeSpec.Name.Name,
		r.fileIdent.Name,
		serviceFunctions,
	}

	//Аллокация буфера,
	//куда будем заливать выполненный шаблон
	var buf bytes.Buffer
	//Процессинг шаблона с подготовленными параметрами
	//в подготовленный буфер
	switch outFile.Name.Name {
	case implementationPackage:
		err = templates.ServiceTemplate.Execute(&buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case transportPackage:
		err = templates.TransportTemplate.Execute(&buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
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
	//Аллокация результирующих деревьев разбора
	var astOutFiles = []*ast.File{
		&ast.File{
			Name: &ast.Ident{
				Name: implementationPackage,
			}},
		&ast.File{
			Name: &ast.Ident{
				Name: transportPackage,
			}},
	}

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
	var genTasks []implementationGenerator

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
				genTasks = append(genTasks, implementationGenerator{
					fileIdent: astInFile.Name,
					typeSpec:  typeSpec,
					methods:   interfaceType.Methods.List,
				})
			}
		}
		return false
	})

	//Запускаем список заданий генерации
	for _, task := range genTasks {
		//Для каждого задания вызываем написанный нами генератор
		//как метод этого задания
		//Сгенерированные декларации помещаются в результирующее дерево разбора
		for _, astOutFile := range astOutFiles {
			err = task.Generate(astOutFile)
			if err != nil {
				log.Fatalf("generate: %v", err)
			}

			err := generateFile(astOutFile)
			if err != nil {
				log.Fatalf("generate file: %v", err)
			}

		}
	}

}

func generateFile(astOutFile *ast.File) error {
	var packageName string
	var filename string

	switch astOutFile.Name.Name {
	case implementationPackage:
		packageName = implementationPackage
		filename = implementationPackage + "_gen.go"
	case transportPackage:
		packageName = transportPackage
		filename = transportPackage + "_gen.go"
	}
	err := os.Mkdir(packageName, 0660)
	if err != nil {
		return fmt.Errorf("create dir: %v", err)
	}

	//Подготовим файл конечного результата всей работы,
	//назовем его созвучно файлу модели, добавим только суффикс _gen
	outFile, err := os.OpenFile(filepath.Join(packageName, filename), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("create file: %v", err)
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
		return fmt.Errorf("print file: %v", err)
	}

	return nil
}
