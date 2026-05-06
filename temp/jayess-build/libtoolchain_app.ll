; ModuleID = 'toolchain_app'
target triple = "x86_64-pc-linux-gnu"

%jayess.value = type { i64, i64 }

declare %jayess.value @jayess_value_from_number(double)
; legacy declare %jayess.value (double) @jayess_value_from_number

declare i32 @jayess_value_to_exit_code(%jayess.value)
; legacy declare i32 (%jayess.value) @jayess_value_to_exit_code

define i32 @main() {
  %jayess.result = call %jayess.value @__jayess_user_main()
  %jayess.exit_code = call i32 @jayess_value_to_exit_code(%jayess.value %jayess.result)
  ret i32 %jayess.exit_code
}

define %jayess.value @__jayess_user_main() {
  %v0 = call %jayess.value @jayess_value_from_number(double 0.0)
  ; legacy %v0 = call %jayess.value @jayess_value_from_number(double 0)
  %local.0 = alloca %jayess.value
  store %jayess.value %v0, %jayess.value* %local.0
  br label %return.0
  ; legacy ret %jayess.value %v0
  return.0:
  %v1 = load %jayess.value, %jayess.value* %local.0
  ret %jayess.value %v1
}
