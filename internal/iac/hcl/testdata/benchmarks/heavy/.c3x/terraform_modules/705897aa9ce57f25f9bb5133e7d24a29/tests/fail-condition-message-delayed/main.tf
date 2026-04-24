module "fail" {
  source = "../../"
  // Use a condition that isn't known until the apply step
  condition = uuid() == ""
  // Use a message that isn't known until the apply step
  error_message = "sample error: ${uuid()}"
}
