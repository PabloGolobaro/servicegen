package main

import (
	"bytes"
	"errors"
	"flag"
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
const httpPackage = "httptransport"
const natsPackage = "natstransport"
const middlewarePackage = "middleware"

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
	fileIdent   *ast.Ident
	typeSpec    *ast.TypeSpec
	methods     []*ast.Field
	OutFiles    []*ast.File
	packagePath string
}

type serviceFunction struct {
	Name       string
	Signature  string
	Arguments  []argument
	ResultType string
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
func extractResultType(spec ast.Expr) (string, error) {
	ret := ""
	funcType, ok := spec.(*ast.FuncType)
	if !ok {
		return "", errors.New("type is not *ast.FuncType")
	}
	resultType, ok := funcType.Results.List[0].Type.(*ast.Ident)
	if !ok {
		return "", errors.New("resultType is not *ast.Ident")
	}

	ret = resultType.Name

	return ret, nil
}

func (r serviceGenerator) convertFunctions() ([]serviceFunction, error) {
	ret := []serviceFunction{}
	for _, method := range r.methods {
		arguments, err := extractArguments(method.Type)
		if err != nil {
			return ret, err
		}

		result, err := extractResultType(method.Type)
		if err != nil {
			return nil, err
		}

		f := serviceFunction{
			Name:       method.Names[0].Name,
			Signature:  strings.TrimPrefix(expr2string(method.Type), "func"),
			Arguments:  arguments,
			ResultType: result,
		}
		ret = append(ret, f)
	}
	return ret, nil
}

func (r serviceGenerator) Generate(outFile *ast.File) error {
	//Аллокация и установка параметров для template

	serviceFunctions, err := r.convertFunctions()
	if err != nil {
		return err
	}
	params := struct {
		ServiceName      string
		ServicePackage   string
		Functions        []serviceFunction
		TransportPackage string
		PackagePath      string
	}{
		//Параметры извлекаем из ресивера метода
		ServiceName:      r.typeSpec.Name.Name,
		ServicePackage:   r.fileIdent.Name,
		Functions:        serviceFunctions,
		TransportPackage: transportPackage,
		PackagePath:      r.packagePath,
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
	case httpPackage:
		err = templates.HttpTemplate.Execute(&buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case natsPackage:
		err = templates.NatsTemplate.Execute(&buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case middlewarePackage:
		err = templates.LoggingTemplate.Execute(&buf, params)
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
	mod := flag.String("mod", ".", "Module name of generate source service")

	flag.Parse()

	if *mod == "." {
		panic("No module name is provided")
	}

	fmt.Println(*mod)

	_ = gorm.DB{}

	//Цель генерации передаётся переменной окружения
	path := os.Getenv("GOFILE")
	if path == "" {
		path = "./services/calc/service.go"
	}

	packagePath := getPackagePath(*mod)

	fmt.Println(packagePath)

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

			var generator serviceGenerator

			//выделяем структуры, помеченные комментарием servicegen:service,
			if strings.Contains(comment.Text, "servicegen:service") {
				//и добавляем в список заданий генерации
				generator = serviceGenerator{
					fileIdent: astInFile.Name,
					typeSpec:  typeSpec,
					methods:   interfaceType.Methods.List,
					OutFiles: []*ast.File{
						{Name: &ast.Ident{Name: implementationPackage}},
						{Name: &ast.Ident{Name: transportPackage}},
					},
					packagePath: packagePath,
				}
			}
			if strings.Contains(comment.Text, "http") {
				generator.OutFiles = append(generator.OutFiles,
					&ast.File{Name: &ast.Ident{Name: httpPackage}},
				)
			}
			if strings.Contains(comment.Text, "nats") {
				generator.OutFiles = append(generator.OutFiles,
					&ast.File{Name: &ast.Ident{Name: natsPackage}},
				)
			}
			if strings.Contains(comment.Text, "logging") {
				generator.OutFiles = append(generator.OutFiles,
					&ast.File{Name: &ast.Ident{Name: middlewarePackage}},
				)
			}

			genTasks = append(genTasks, generator)
		}
		return false
	})

	//Запускаем список заданий генерации
	for _, task := range genTasks {
		//Для каждого задания вызываем написанный нами генератор
		//как метод этого задания
		//Сгенерированные декларации помещаются в результирующее дерево разбора
		for _, outFile := range task.OutFiles {
			err = task.Generate(outFile)
			if err != nil {
				log.Fatalf("generate: %v", err)
			}

			err := generateFile(outFile)
			if err != nil {
				log.Fatalf("generate file: %v", err)
			}

		}
	}

}

func generateFile(OutFile *ast.File) error {
	var packageName = OutFile.Name.Name
	var filePath string
	var dir string

	switch packageName {
	case implementationPackage:
		dir = packageName
	case transportPackage:
		dir = packageName
	case middlewarePackage:
		dir = packageName
	case httpPackage:
		dir = filepath.Join(transportPackage, packageName)
	case natsPackage:
		dir = filepath.Join(transportPackage, packageName)
	}

	filePath = filepath.Join(dir, packageName) + "_gen.go"

	err := os.Mkdir(dir, 0660)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists.") {
			return fmt.Errorf("create dir: %v", err)
		}

	}

	//Подготовим файл конечного результата всей работы,
	//назовем его созвучно файлу модели, добавим только суффикс _gen
	outFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
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
	err = printer.Fprint(outFile, token.NewFileSet(), OutFile)
	if err != nil {
		return fmt.Errorf("print file: %v", err)
	}

	return nil
}

func getPackagePath(mod string) string {
	var packageDir string
	cwd, _ := os.Getwd()
	//C:\Users\Professional\GolandProjects\servicegen\services\calc
	//github.com/pablogolobaro/servicegen

	modParts := strings.Split(mod, "/")
	lastModPart := modParts[len(modParts)-1]

	splitedCwd := strings.Split(cwd, lastModPart)
	packageDir = splitedCwd[len(splitedCwd)-1]

	fmt.Println(packageDir)

	packagePath := filepath.Join(mod, packageDir)

	packagePath = cleanPath(packagePath)

	return packagePath
}

func cleanPath(dirPath string) string {
	dirPath = strings.Replace(dirPath, "\\", "/", -1)
	return dirPath
}
