package schema

import (
	"reflect"
	"testing"
)

func TestFieldName(t *testing.T) {
	tests := []struct {
		field reflect.StructField
		want  string
	}{
		{
			reflect.StructField{Name: "DeadLetterConfig"},
			"dead_letter_config",
		},
		{
			reflect.StructField{Name: "KMSKeyArn"},
			"kms_key_arn",
		},
		{
			// Without custom tag
			reflect.StructField{Name: "RestAPIID"},
			"rest_apiid",
		},
		{
			// Custom tag
			reflect.StructField{Name: "RestAPIID", Tag: reflect.StructTag(`name:"rest_api_id"`)},
			"rest_api_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := fieldName(tt.field); got != tt.want {
				t.Errorf("fieldName() = %v, want %v", got, tt.want)
			}
		})
	}
}
