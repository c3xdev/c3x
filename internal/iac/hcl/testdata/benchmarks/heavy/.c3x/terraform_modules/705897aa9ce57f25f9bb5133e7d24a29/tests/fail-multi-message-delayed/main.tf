module "fail_multi_message_delayed" {
  source = "../../"
  assertions = [
    {
      condition     = false
      error_message = "sample error: ${uuid()}"
    },
    {
      condition     = false
      error_message = "sample error: ${uuid()}"
    }
  ]
}
