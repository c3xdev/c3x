provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}

resource "aws_lambda_function" "event_handler" {
  function_name = "event_handler"
  role          = "arn:aws:lambda:us-east-1:aws:resource-id"
  handler       = "index.handler"
  runtime       = "nodejs20.x"
  filename      = "function.zip"
  memory_size   = 1024 # <<<<< Try changing this to 512 to compare costs
}
