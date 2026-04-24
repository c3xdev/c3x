provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}


module "mod" {
  for_each = { test = "1", test2 = "2" }

  source = "./modules/test"

  range = { foo = "bar", baz = "bat" }
}

module "mod2" {
  for_each = { test = "1", test2 = "2" }

  source = "./modules/test"

  range = { foo = "bar", baz = "bat" }
}

module "mod3" {
  for_each = { test = "1", test2 = "2" }

  source = "./modules/test"

  range = { foo = "bar", baz = "bat" }
}

resource "aws_lambda_function" "test" {
  function_name = "event_handler"
  role          = "arn:aws:lambda:us-east-1:aws:resource-id"
  handler       = "index.handler"
  runtime       = "nodejs20.x"
  filename      = "function.zip"
  memory_size   = 1024
}
