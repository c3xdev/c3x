module "pass" {
  source = "../../"
  // Use a condition that isn't known until the apply step
  condition     = uuid() != ""
  error_message = "sample error"
}
