package main

import (
	"flag"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pablogolobaro/servicegen/generator"
	"github.com/pablogolobaro/servicegen/utils"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/ast/inspector"
	"log"
	"os"
	"strings"
)

func main() {
	//Аллокация результирующих деревьев разбора
	mod := flag.String("mod", "github.com/pablogolobaro/servicegen", "Module name of generate source service")

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

	packagePath := utils.GetPackagePath(*mod)

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

	servicePackageName := astInFile.Name.Name
	//Для выбора интересных нам деклараций
	//используем Inspector из golang.org/x/tools/go/ast/inspector
	i := inspector.New([]*ast.File{astInFile})
	//Подготовим фильтр для этого инспектора
	iFilter := []ast.Node{
		//Нас интересуют декларации
		&ast.GenDecl{},
	}
	//Выделяем список заданий генерации
	var genTasks []generator.ServiceGenerator

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

			var gen generator.ServiceGenerator

			//выделяем структуры, помеченные комментарием servicegen:service,
			if strings.Contains(comment.Text, "servicegen:service") {
				//и добавляем в список заданий генерации
				gen = generator.ServiceGenerator{
					FileIdent:          astInFile.Name,
					TypeSpec:           typeSpec,
					Methods:            interfaceType.Methods.List,
					PackagePath:        packagePath,
					ServicePackageName: servicePackageName,
					ModuleName:         *mod,
					OutFiles: map[string]*ast.File{
						generator.ImplementationPackage: {Name: &ast.Ident{Name: generator.ImplementationPackage}},
						generator.TransportPackage:      {Name: &ast.Ident{Name: generator.TransportPackage}},
						generator.RootFilename:          {Name: &ast.Ident{Name: generator.CmdPackage}},
						generator.ConfigPackage:         {Name: &ast.Ident{Name: generator.ConfigPackage}},
						generator.OtelTracingPackage:    {Name: &ast.Ident{Name: generator.OtelTracingPackage}},
						generator.ErrorFileName:         {Name: &ast.Ident{Name: servicePackageName}},
					},
				}
			}
			if strings.Contains(comment.Text, "http") {
				gen.OutFiles[generator.HttpFileName] = &ast.File{Name: &ast.Ident{Name: generator.HttpPackage}}
				gen.OutFiles[generator.HttpRunFilename] = &ast.File{Name: &ast.Ident{Name: generator.CmdPackage}}

			}
			if strings.Contains(comment.Text, "nats") {
				gen.OutFiles[generator.NatsFileName] = &ast.File{Name: &ast.Ident{Name: generator.NatsPackage}}
			}
			if strings.Contains(comment.Text, "logging") {
				gen.OutFiles[generator.LoggingFileName] = &ast.File{Name: &ast.Ident{Name: generator.MiddlewarePackage}}
			}
			if strings.Contains(comment.Text, "tracing") {
				gen.OutFiles[generator.TracingFileName] = &ast.File{Name: &ast.Ident{Name: generator.MiddlewarePackage}}
			}

			genTasks = append(genTasks, gen)
		}
		return false
	})

	//Запускаем список заданий генерации
	for _, task := range genTasks {
		//Для каждого задания вызываем написанный нами генератор
		//как метод этого задания
		//Сгенерированные декларации помещаются в результирующее дерево разбора
		for fileName, outFile := range task.OutFiles {
			err = task.Generate(outFile, fileName)
			if err != nil {
				log.Fatalf("generate: %v", err)
			}

			err := task.GenerateFile(outFile, fileName)
			if err != nil {
				log.Fatalf("generate file: %v", err)
			}
		}
	}
}
