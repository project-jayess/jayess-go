function main(args) {
  var count = 0;
outer:
  for (var i = 0; i < 3; i++) {
    for (var j = 0; j < 3; j++) {
      if (i === 1 && j === 1) {
        continue outer;
      }
      count++;
      if (count > 5) {
        break outer;
      }
    }
  }
  return count === 6 ? 0 : 1;
}
