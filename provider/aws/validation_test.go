package aws

import "testing"

func TestValidationARN(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		// Valid ARNs from
		// https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#genref-arns
		{"arn:partition:service:region:account-id:resource", true},
		{"arn:partition:service:region:account-id:resourcetype/resource", true},
		{"arn:partition:service:region:account-id:resourcetype/resource/qualifier", true},
		{"arn:partition:service:region:account-id:resourcetype/resource:qualifier", true},
		{"arn:partition:service:region:account-id:resourcetype:resource", true},
		{"arn:partition:service:region:account-id:resourcetype:resource:qualifier", true},
		{"arn:partition:service:region:account-id", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validARN(tt.input, "")
			if (err == nil) != tt.valid {
				t.Errorf("got err = %v, want err = %t", err, tt.valid)
			}
		})
	}
}
