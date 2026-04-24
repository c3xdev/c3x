module "pass_multi" {
  source = "../../"
  assertions = [
    {
      condition     = true
      error_message = "Not needed!"
    },
    {
      condition     = true
      error_message = "Not needed!"
    }
  ]
}
