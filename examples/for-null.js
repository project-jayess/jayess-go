function main(args) {
  var label = null;
  var fallback = undefined;
  var total = 0;

  for (var i = 0; i < 4; i = i + 1) {
    if (i == 2) {
      continue;
    }
    total = total + i;
  }

  if (label == null) {
    print(label);
  }

  if (!fallback) {
    print(fallback);
  }

  return total;
}
