module "fail" {
  source        = "../../"
  condition     = false
  error_message = "sample error"
}

// Force the module and conditions to be evaluated
// by using the module output in a config output.
output "result" {
  value = module.fail.checked
}
