module "fail_multi_condition_message_delayed" {
  source = "../../"
  assertions = [
    {
      condition     = uuid() == ""
      error_message = "sample error: ${uuid()}"
    },
    {
      condition     = uuid() == ""
      error_message = "sample error: ${uuid()}"
    }
  ]
}
