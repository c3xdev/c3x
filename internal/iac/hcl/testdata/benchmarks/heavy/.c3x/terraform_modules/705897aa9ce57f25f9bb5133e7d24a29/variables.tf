variable "error_message" {
  description = "The error message to display if the condition evaluates to `false`."
  type        = string
  nullable    = true
  default     = null
}

variable "condition" {
  description = "The condition to ensure is `true`."
  type        = bool
  nullable    = true
  default     = null
  validation {
    // We have to use var.error_message == "" to force the evaluation to wait
    // until var.error_message is known. Otherwise, it can fail during the validation
    // phase but won't output the proper error message.
    // https://github.com/hashicorp/terraform/issues/35397
    condition = var.condition == null ? true : (var.error_message == null ? true : (var.error_message == "" ? var.condition : var.condition))
    error_message = var.condition == null ? "" : (
      var.condition ? "" : (
        var.error_message == null ? "" : "ASSERTION FAILURE: ${var.error_message}"
      )
    )
  }
}

variable "assertions" {
  description = "A list of conditions and error message pairs to check. This is for use when you want to make multiple assertions with a single instance of this module."
  type = list(object({
    condition     = bool
    error_message = string
  }))
  default  = null
  nullable = true

  // Verify that either the single assertion variables are used, or this variable is used. Not both.
  validation {
    // Ternaries allow lazy evaluation, which reduces the chance that a value
    // not known at plan time will force this validation to wait for apply time.
    condition = (
      var.condition == null ? (
        var.error_message == null ? var.assertions != null : false
      ) : false
      ) ? true : (
      var.condition != null ? (
        var.error_message != null ? var.assertions == null : false
      ) : false
    )
    error_message = "Either the `condition` and `error_message` variables must be provided and the `assertions` variable must be `null`, or the `assertions` variable must be provided and the `condition` and `error_message` variables must be `null`. To check a single assertion, use `condition` and `error_message`; to check multiple assertions, use `assertions`."
  }

  // Check that either there is no assertions list, or if there is,
  // none of the conditions in it are false.
  validation {
    // We pass this validation if `condition` or `error_message` are not null,
    // so it throws the validation error about multiple vars being provided instead
    // of actually checking the assertions in this variable.
    condition = var.condition != null ? true : (
      var.error_message != null ? true : (
        var.assertions == null ? true : (
          length([
            for v in var.assertions :
            false
            if !v.condition
          ]) == 0
        )
      )
    )
    error_message = var.assertions == null ? "" : join("\n\n", [
      for v in var.assertions :
      "ASSERTION FAILURE: ${v.error_message}"
      if !v.condition
    ])
  }
}
