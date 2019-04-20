<!-- This file was generated by structdoc. DO NOT EDIT. -->
<!-- For changes modify iam_role_policy.go instead. -->

# aws_iam_role_policy

IAMRolePolicy is an inline role policy, attached to a role.

## Overview

| i/o | name | type | required |
| --- | ---- | ---- | -------: |
| input | [`policy_document`](#policy_document) | `string` | required |
| input | [`policy_name`](#policy_name) | `string` | required |
| input | [`role_name`](#role_name) | `string` | required |


## Inputs

### policy_document

`string`

The policy document.

### policy_name

`string`

The name of the policy document.

### role_name

`string`

The name of the role to associate the policy with.

## Outputs
