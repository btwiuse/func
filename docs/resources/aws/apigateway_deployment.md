<!-- This file was generated by structdoc. DO NOT EDIT. -->
<!-- For changes modify apigateway_deployment.go instead. -->

# aws_apigateway_deployment

APIGatewayDeployment provides a Serverless REST API.

## Overview

| i/o | name | type | required |
| --- | ---- | ---- | -------: |
| input | [`cache_cluster_enabled`](#cache_cluster_enabled) | `bool` |  |
| input | [`cache_cluster_size`](#cache_cluster_size) | `string` |  |
| input | [`canary_settings`](#canary_settings) | `APIGatewayCanarySettings` |  |
| input | [`description`](#description) | `string` |  |
| input | [`rest_api_id`](#rest_api_id) | `string` | required |
| input | [`state_description`](#state_description) | `string` |  |
| input | [`stage_name`](#stage_name) | `string` | required |
| input | [`tracing_enabled`](#tracing_enabled) | `bool` |  |
| input | [`variables`](#variables) | `map` |  |
| input | [`region`](#region) | `string` | required |
| input | [`change_trigger`](#change_trigger) | `string` | required |
| output | [`api_summary`](#api_summary) | `map` ||
| output | [`created_date`](#created_date) | `time` ||


## Inputs

### cache_cluster_enabled

`optional bool`

Enables a cache cluster for the Stage resource specified in the input.

### cache_cluster_size

`optional string`

Specifies the cache cluster size for the Stage resource specified in the
input, if a cache cluster is enabled.

Value must be one of:
- "0.5"
- "1.6"
- "6.1"
- "13.5"
- "28.4"
- "58.2"
- "118"
- "237"

### canary_settings

`optional APIGatewayCanarySettings`

The input configuration for the canary deployment when the deployment is
a canary release deployment.

### description

`optional string`

The description for the Deployment resource to create.

### rest_api_id

`string`

The string identifier of the associated RestApi.

### state_description

`optional string`

The description of the Stage resource for the Deployment resource to
create.

### stage_name

`string`

The name of the Stage resource for the Deployment resource to create.

### tracing_enabled

`optional bool`

Specifies whether active tracing with X-ray is enabled for the Stage.

### variables

`optional map`

A map that defines the stage variables for the Stage resource that is
associated with the new deployment. Variable names can have alphanumeric
and underscore characters, and the values must match
`[A-Za-z0-9-._~:/?#&=,]+`.

### region

`string`



### change_trigger

`string`

ChangeTrigger causes a new deployment to be executed when the value has
changed, even if other inputs have not changed.

## Outputs

### api_summary

`map`

A summary of the RestApi at the date and time that the deployment resource
was created.
### created_date

`time`

The date and time that the deployment resource was created.