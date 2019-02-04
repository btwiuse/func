resource "aws_lambda_function" "func" {
  source  = "." # not processed as the compressor is not set
  handler = "index.handler"
}

