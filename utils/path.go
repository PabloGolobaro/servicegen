package utils

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func GetPackagePath(mod string) string {
	var packageDir string
	cwd, _ := os.Getwd()

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

func Expr2string(expr ast.Expr) (string, error) {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, token.NewFileSet(), expr)
	if err != nil {
		return "", err

	}
	return buf.String(), nil
}
