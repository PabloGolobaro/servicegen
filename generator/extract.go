package generator

import (
	"errors"
	"fmt"
	"github.com/pablogolobaro/servicegen/utils"
	"go/ast"
	"strings"
)

type ServiceFunction struct {
	Name                string      //Имя функции
	Signature           string      // Полная сигнатура
	Arguments           []parameter // Список аргументов
	ResultFullSignature string      // Полный набор возвращаемых значений в виде строки
	Results             []parameter // Список возвращаемых значений
}

type parameter struct {
	Name string
	Type string
}

func (r ServiceGenerator) convertFunctions() ([]ServiceFunction, error) {
	ret := []ServiceFunction{}
	for _, method := range r.Methods {
		arguments, err := extractArguments(method.Type)
		if err != nil {
			return ret, err
		}

		result, err := extractFullResultSignature(method.Type)
		if err != nil {
			return nil, err
		}

		resultParameters, err := extractResults(method.Type)
		if err != nil {
			return nil, err
		}
		signSting, err := utils.Expr2string(method.Type)
		if err != nil {
			return nil, err
		}
		//log.Fatalf("error print expression to string: #{err}")
		f := ServiceFunction{
			Name:                method.Names[0].Name,
			Signature:           strings.TrimPrefix(signSting, "func"),
			Arguments:           arguments,
			ResultFullSignature: result,
			Results:             resultParameters,
		}
		ret = append(ret, f)
	}
	return ret, nil
}

func extractArguments(spec ast.Expr) ([]parameter, error) {
	ret := []parameter{}
	funcType, ok := spec.(*ast.FuncType)
	if !ok {
		return nil, errors.New("type is not *ast.FuncType")
	}
	for _, param := range funcType.Params.List {
		switch paramType := param.Type.(type) {
		case *ast.Ident:
			for _, name := range param.Names {
				ret = append(ret, parameter{Name: name.Name, Type: paramType.Name})
			}
		case *ast.SelectorExpr:
			x := paramType.X.(*ast.Ident)
			sel := paramType.Sel
			for _, name := range param.Names {
				ret = append(ret, parameter{Name: name.Name, Type: fmt.Sprintf("%s.%s", x.Name, sel.Name)})
			}
		}

	}
	return ret, nil
}

func extractFullResultSignature(spec ast.Expr) (string, error) {
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

func extractResults(spec ast.Expr) ([]parameter, error) {
	ret := []parameter{}
	funcType, ok := spec.(*ast.FuncType)
	if !ok {
		return nil, errors.New("type is not *ast.FuncType")
	}

	for _, resultField := range funcType.Results.List {

		switch resultType := resultField.Type.(type) {

		case *ast.Ident:
			param := parameter{Type: resultType.Name}
			ret = append(ret, param)

		case *ast.SelectorExpr:
			x := resultType.X.(*ast.Ident)
			sel := resultType.Sel
			param := parameter{Type: fmt.Sprintf("%s.%s", x.Name, sel.Name)}
			ret = append(ret, param)

		default:
			return nil, fmt.Errorf("unknown type of Results.Field")
		}

	}

	return ret, nil
}
