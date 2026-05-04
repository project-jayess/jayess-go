const values = [1, 2, 3];
const record = { a: 1, b: 2 };

function main(args) {
  var total = 0;
  for (var value of values) {
    total += value;
  }
  for (var key in record) {
    total += record[key];
  }
  return total === 9 ? 0 : 1;
}
