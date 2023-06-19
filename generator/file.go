package generator

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func (r ServiceGenerator) GenerateFile(OutFile *ast.File, fileName string) error {
	var packageName = OutFile.Name.Name
	var filePath string
	var dir string
	var subDir string

	switch packageName {
	case r.ServicePackageName:
		dir = ""
	case CmdPackage:
		dir = packageName
	case OtelTracingPackage:
		dir = packageName
	case ConfigPackage:
		dir = packageName
	case ImplementationPackage:
		dir = packageName
	case TransportPackage:
		dir = packageName
	case MiddlewarePackage:
		dir = packageName
	case HttpPackage:
		dir = filepath.Join(TransportPackage, packageName)
		subDir = TransportPackage
	case NatsPackage:
		dir = filepath.Join(TransportPackage, packageName)
		subDir = TransportPackage
	}

	filePath = filepath.Join(dir, fileName) + "_gen.go"

	if subDir != "" {
		if err := createDir(subDir); err != nil {
			return fmt.Errorf("create subDir: %v", err)
		}
	}

	if dir != "" {
		if err := createDir(dir); err != nil {
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

func createDir(dir string) error {
	err := os.Mkdir(dir, 0660)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists.") {
			return err
		}
	}
	return nil
}
