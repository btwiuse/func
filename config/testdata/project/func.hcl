resource "aws_lambda_function" "func" {
  source = "./src"

  handler = "index.handler"
  memory  = 512
}

