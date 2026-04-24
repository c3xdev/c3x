module "pass_multi_message_delayed" {
  source = "../../"
  assertions = [
    {
      condition     = true
      error_message = "sample error: ${uuid()}"
    },
    {
      condition     = true
      error_message = "sample error: ${uuid()}"
    }
  ]
}
