resource "lambda" {
  type   = "aws:lambda_function"

  source = "./src"

  handler = "index.handler"
  memory  = 512
}
