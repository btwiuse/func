package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func TestIAMPolicyDocument_generate(t *testing.T) {
	strptr := func(str string) *string { return &str }

	tests := []struct {
		name  string
		input *IAMPolicyDocument
		want  string
	}{
		{
			"Lambda",
			&IAMPolicyDocument{
				Statements: []IAMPolicyStatement{{
					ID:         strptr("test"),
					Effect:     "Allow",
					Actions:    &[]string{"sts:AssumeRole"},
					Principals: &map[string][]string{"Service": {"lambda.amazonaws.com"}},
				}},
			},
			minify(`{
				"Version": "2012-10-17",
					"Statement": [{
						"Sid": "test",
						"Effect": "Allow",
						"Action": "sts:AssumeRole",
						"Principal": {
							"Service": "lambda.amazonaws.com"
						}
					}
				]
			}`),
		},
		{
			"Logs",
			&IAMPolicyDocument{
				Version: strptr("2012-10-17"),
				Statements: []IAMPolicyStatement{{
					Effect:    "Allow",
					Actions:   &[]string{"logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"},
					Resources: &[]string{"*"},
				}},
			},
			minify(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": [
						"logs:CreateLogGroup",
						"logs:CreateLogStream",
						"logs:PutLogEvents"
					],
					"Resource": "*"
				}]
			}`),
		},
		{
			// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_examples_aws-dates.html
			"AccessSpecificDates",
			&IAMPolicyDocument{
				Statements: []IAMPolicyStatement{{
					Effect:    "Allow",
					Actions:   &[]string{"service-prefix:action-name"},
					Resources: &[]string{"*"},
					Conditions: &map[string]map[string]string{
						"DateGreaterThan": {
							"aws:CurrentTime": "2017-07-01T00:00:00Z",
						},
						"DateLessThan": {
							"aws:CurrentTime": "2017-12-31T23:59:59Z",
						},
					},
				}},
			},
			minify(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "service-prefix:action-name",
					"Resource": "*",
					"Condition": {
						"DateGreaterThan": {"aws:CurrentTime": "2017-07-01T00:00:00Z"},
						"DateLessThan": {"aws:CurrentTime": "2017-12-31T23:59:59Z"}
					}
				}]
			}`),
		},
		{
			// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_examples_s3_deny-except-bucket.html
			"LimitS3Management",
			&IAMPolicyDocument{
				Statements: []IAMPolicyStatement{{
					Effect:    "Allow",
					Actions:   &[]string{"s3:*"},
					Resources: &[]string{"arn:aws:s3:::bucket1", "arn:aws:s3:::bucket1/*"},
				}, {
					Effect:       "Deny",
					NotActions:   &[]string{"s3:*"},
					NotResources: &[]string{"arn:aws:s3:::bucket2", "arn:aws:s3:::bucket2/*"},
				}},
			},
			minify(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "s3:*",
					"Resource": [
						"arn:aws:s3:::bucket1",
						"arn:aws:s3:::bucket1/*"
					]
				}, {
					"Effect": "Deny",
					"NotAction": "s3:*",
					"NotResource": [
						"arn:aws:s3:::bucket2",
						"arn:aws:s3:::bucket2/*"
					]
				}]
			}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.generate()
			if err != nil {
				t.Fatalf("generate() error = %v", err)
			}
			got := tt.input.JSON
			if got != tt.want {
				t.Errorf("Generated doc does not match\n\nGot:\n%s\n\nWant:\n%s", pretty(got), pretty(tt.want))
			}
		})
	}
}

func minify(jsonDoc string) string {
	var b bytes.Buffer
	if err := json.Compact(&b, []byte(jsonDoc)); err != nil {
		return fmt.Sprintf("minify error: %v", err)
	}
	return b.String()
}

func pretty(jsonDoc string) string {
	var b bytes.Buffer
	if err := json.Indent(&b, []byte(jsonDoc), "", "  "); err != nil {
		return fmt.Sprintf("minify error: %v", err)
	}
	return b.String()
}
