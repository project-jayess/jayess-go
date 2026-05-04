function main(args) {
  var enabled = true;
  const disabled = false;
  var count = 0;

  if (!disabled && (args[0] == "run" || args[1] == "go")) {
    print("active");
  } else if (disabled) {
    print("disabled");
  } else {
    print("idle");
  }

  while (enabled) {
    count = count + 1;

    if (count == 2) {
      continue;
    }

    if (count > 3) {
      break;
    }

    print(count);
  }

  return count;
}
