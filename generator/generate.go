package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// ServiceGenerator - агрегатор данных для установки параметров в шаблоне
type ServiceGenerator struct {
	FileIdent          *ast.Ident           // Шапка файла
	TypeSpec           *ast.TypeSpec        // Полная спецификация типа для интерфейса сервиса
	Methods            []*ast.Field         // Набор методов интерфейса сервиса
	OutFiles           map[string]*ast.File // Набор выходных файлов с подготовленной шапкой
	PackagePath        string               // Относительный путь к исходному интерфейсу
	ServicePackageName string               //пакэдж исходного файла
	ModuleName         string               // имя модуля
}

func (r ServiceGenerator) Generate(outFile *ast.File, fileName string) error {

	//Аллокация и установка параметров для template
	serviceFunctions, err := r.convertFunctions()
	if err != nil {
		return err
	}
	params := templateParams{
		//Параметры извлекаем из ресивера метода
		ServiceName:      r.TypeSpec.Name.Name,
		ServicePackage:   r.FileIdent.Name,
		Functions:        serviceFunctions,
		PackagePath:      r.PackagePath,
		TransportPackage: TransportPackage,
		ModuleName:       r.ModuleName,
	}

	packageName := outFile.Name.Name
	//Аллокация буфера,
	//куда будем заливать выполненный шаблон
	var buf bytes.Buffer
	//Процессинг шаблона с подготовленными параметрами
	//в подготовленный буфер
	err = r.ExecuteTemplate(&buf, packageName, fileName, params)
	if err != nil {
		return err
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
