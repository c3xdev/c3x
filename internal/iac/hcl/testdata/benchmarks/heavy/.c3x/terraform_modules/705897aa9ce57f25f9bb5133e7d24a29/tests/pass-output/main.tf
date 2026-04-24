module "pass" {
  source        = "../../"
  condition     = true
  error_message = "sample error"
}

// Force the module and conditions to be evaluated
// by using the module output in a config output.
output "result" {
  value = module.pass.checked
}
