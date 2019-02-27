<!-- This file was generated by structdoc. DO NOT EDIT. -->
<!-- For changes modify iam_role.go instead. -->

# aws_iam_role

IAMRole creates a new role for your AWS account. For more information about
roles, go to [IAM Roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/WorkingWithRoles.html).

For information about limitations on role names and the number of roles you
can create, go to Limitations on
[IAM Entities](http://docs.aws.amazon.com/IAM/latest/UserGuide/LimitationsOnEntities.html)
in the IAM User Guide.

## Overview

| i/o | name | type | required |
| --- | ---- | ---- | -------: |
| input | [`assume_role_policy_document`](#assume_role_policy_document) | `string` | required |
| input | [`description`](#description) | `string` |  |
| input | [`max_session_duration`](#max_session_duration) | `int64` |  |
| input | [`path`](#path) | `string` |  |
| input | [`permission_boundary`](#permission_boundary) | `string` |  |
| input | [`role_name`](#role_name) | `string` | required |
| output | [`arn`](#arn) | `string` ||
| output | [`create_date`](#create_date) | `time` ||
| output | [`role_id`](#role_id) | `string` ||


## Inputs

### assume_role_policy_document

`string`

The trust relationship policy document that grants an entity permission to
assume the role.

The [regex pattern](http://wikipedia.org/wiki/regex) used to validate this
parameter is a string of characters consisting of the following:

* Any printable ASCII character ranging from the space character (\u0020)
  through the end of the ASCII character range

* The printable characters in the Basic Latin and Latin-1 Supplement character
  set (through \u00FF)

* The special characters tab (\u0009), line feed (\u000A), and carriage
  return (\u000D)

### description

`optional string`

A description of the role.

### max_session_duration

`optional int64`

The maximum session duration (in seconds) that you want to set for the
specified role. If you do not specify a value for this setting, the
default maximum of one hour is applied. This setting can have a value
from 1 hour to 12 hours.

Anyone who assumes the role from the AWS CLI or API can use the
DurationSeconds API parameter or the duration-seconds CLI parameter to
request a longer session. The MaxSessionDuration setting determines the
maximum duration that can be requested using the DurationSeconds
parameter. If users don't specify a value for the DurationSeconds
parameter, their security credentials are valid for one hour by default.
This applies when you use the AssumeRole* API operations or the
assume-role* CLI operations but does not apply when you use those
operations to create a console URL. For more information, see
[Using IAM Roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html)
in the IAM User Guide.

### path

`optional string`

The path to the role. For more information about paths, see
[IAM Identifiers](http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
in the IAM User Guide.

This parameter is optional. If it is not included, it defaults to a
slash (/).

This parameter allows (per its
[regex pattern](http://wikipedia.org/wiki/regex) a string of characters
consisting of either a forward slash (/) by itself or a string that must
begin and end with forward slashes. In addition, it can contain any
ASCII character from the ! (\u0021) through the DEL character (\u007F),
including most punctuation characters, digits, and upper and lowercased
letters.

### permission_boundary

`optional string`

The ARN of the policy that is used to set the permissions boundary for
the role.

### role_name

`string`

The name of the role to create.

This parameter allows (per its
[regex pattern](http://wikipedia.org/wiki/regex) a string of characters
consisting of upper and lowercase alphanumeric characters with no
spaces. You can also include any of the following characters: _+=,.@-

Role names are not distinguished by case. For example, you cannot create
roles named both "PRODROLE" and "prodrole".

## Outputs

### arn

`string`

The Amazon Resource Name (ARN) specifying the role.

For more information about ARNs and how to use them in policies, see
[IAM Identifiers](http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
in the IAM User Guide guide.
### create_date

`time`


### role_id

`string`

