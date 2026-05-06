# Contributing

Keep changes small, focused, and reviewable.

## File Organization

Do not keep adding unrelated logic to already-large files. Prefer a new focused
file in the relevant package when a feature naturally separates into its own
responsibility.

## Protected Directories

Do not modify `refs/` or `old_version/` during normal compiler work. They are
reference or external material, not active source.

## Temporary Files

Place generated files, build experiments, and large temporary fixtures under
`temp/`. Place test sources and Go tests under `test/` or the relevant package
test files.

## Checklist

Use `jayess-feature-checklist.md` to track planned work. Do not add new scope
when the current request is to finish existing unchecked items.

## Example Temporary Path

```text
temp/
  jayess-build/
  dist-check/
  generated-fixtures/
```
