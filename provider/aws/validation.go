package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

// AddValidators add AWS specific validator rules.
func AddValidators(validator validator) {
	validator.Add("aws_arn", validARN)
}

func validARN(value interface{}, param string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("input is not a string")
	}
	_, err := arn.Parse(str)
	return err
}
