function makeAdder(base) {
  return function (value) {
    return base + value;
  };
}

const double = (value) => value * 2;

function collect(prefix, ...items) {
  return prefix + items.length;
}

function main(args) {
  var addFive = makeAdder(5);
  var bound = addFive.bind(null, 7);
  var value = bound.call(null);
  return double(value) === 24 && collect("n", 1, 2) === "n2" ? 0 : 1;
}
