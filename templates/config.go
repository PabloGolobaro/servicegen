package templates

import (
	"strings"
	"text/template"
)

var ConfigTemplate *template.Template

func init() {
	ConfigStr = strings.Replace(ConfigStr, "^", "`", -1)
	ConfigTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(ConfigStr))
}

var ConfigStr = `package config

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
)

type ServiceConfig struct {
	NATS struct {
		Endpoint string ^mapstructure:"NATS_ENDPOINT"^
	}
	Production bool
}

var MainConfig ServiceConfig

var Banner = ""
var ApplicationDesription = "Boilerplate service v0.0.1"

func init() {
	fmt.Printf("%s\n %s %s\n", color.GreenString(Banner), emoji.Sprint(":clinking_beer_mugs:"), color.RedString(ApplicationDesription))
}
`
