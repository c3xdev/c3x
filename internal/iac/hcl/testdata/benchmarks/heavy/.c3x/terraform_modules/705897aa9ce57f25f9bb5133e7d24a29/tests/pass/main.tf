module "pass" {
  source        = "../../"
  condition     = true
  error_message = "sample error"
}
