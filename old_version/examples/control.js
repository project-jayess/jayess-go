function main(args) {
  var enabled = true;
  const disabled = false;
  var count = 0;

  if (disabled) {
    print("disabled");
  } else if (args[0] == "run") {
    print("run");
  } else {
    print("other");
  }

  while (enabled) {
    if (count < 3) {
      print(count);
      count = count + 1;
    } else {
      enabled = false;
    }
  }

  return count;
}
