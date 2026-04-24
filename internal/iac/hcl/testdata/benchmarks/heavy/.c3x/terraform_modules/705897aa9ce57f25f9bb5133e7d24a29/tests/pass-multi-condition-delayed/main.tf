module "pass_multi_condition_delayed" {
  source = "../../"
  assertions = [
    {
      condition     = uuid() != ""
      error_message = "sample error"
    },
    {
      condition     = uuid() != ""
      error_message = "sample error"
    }
  ]
}
