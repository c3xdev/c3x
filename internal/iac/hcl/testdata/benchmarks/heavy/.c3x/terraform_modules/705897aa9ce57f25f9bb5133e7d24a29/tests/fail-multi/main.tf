module "fail_multi" {
  source = "../../"
  assertions = [
    {
      condition     = false
      error_message = "sample error"
    },
    {
      condition     = false
      error_message = "sample error"
    }
  ]
}
