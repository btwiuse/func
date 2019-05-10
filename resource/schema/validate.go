package schema

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"gopkg.in/go-playground/validator.v9"
)

var check = validator.New()

func mustRegister(err error) {
	if err != nil {
		panic(fmt.Sprintf("Register custom validator: %v", err))
	}
}

func init() {
	mustRegister(check.RegisterValidation("div", func(fl validator.FieldLevel) bool {
		param := fl.Param()
		if param == "" {
			panic("div validator must have a param (div=64 for dividable by 64)")
		}
		mod, err := strconv.Atoi(param)
		if err != nil {
			panic(fmt.Sprintf("Parse divider: %v", err))
		}
		if fl.Field().Int()%int64(mod) == 0 {
			return true
		}
		return false
	}))
	mustRegister(check.RegisterValidation("arn", func(fl validator.FieldLevel) bool {
		str := fl.Field().String()
		_, err := arn.Parse(str)
		return err == nil
	}))
}

var once sync.Once
var formats map[string]string

func validate(v interface{}, tag string) error {
	err := check.Var(v, tag)
	if err == nil {
		return nil
	}
	once.Do(initFormatters)
	errs := err.(validator.ValidationErrors)
	fe := errs[0]
	format, ok := formats[fe.Tag()]
	if !ok {
		return err
	}
	if !strings.Contains(format, "%") {
		return fmt.Errorf(format)
	}
	return fmt.Errorf(format, fe.Param())
}

func initFormatters() {
	formats = map[string]string{
		"gte":   "must be %v or more",
		"gt":    "must be more than %v",
		"lte":   "must be %v or less",
		"lt":    "must be less than %v",
		"oneof": "must be one of: [%v]",

		// custom
		"div": "must be divisible by %v",
		"arn": "must be a valid arn (https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html)",
	}
}
