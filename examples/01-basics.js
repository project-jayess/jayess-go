const scale = 3;
var total = 0;

function add(a, b) {
  return a + b;
}

function main(args) {
  var first = add(2, 4);
  var second = (first * scale) - 1;
  total = second / 2;
  return total >= 8 ? 0 : 1;
}
