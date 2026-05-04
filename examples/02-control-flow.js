function classify(value) {
  if (value < 0) {
    return "negative";
  } else if (value === 0) {
    return "zero";
  }
  return "positive";
}

function main(args) {
  var sum = 0;
  for (var i = 0; i < 4; i++) {
    sum += i;
  }
  while (sum < 10) {
    sum++;
  }
  do {
    sum--;
  } while (sum > 9);

  switch (classify(sum)) {
    case "positive":
      return 0;
    default:
      return 1;
  }
}
