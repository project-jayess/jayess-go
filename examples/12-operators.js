function main(args) {
  var mask = (3 << 2) | 1;
  var hasBit = (mask & 4) !== 0;
  var power = 2 ** 3;
  var value = +power;
  var nothing = void value;
  var record = { name: "jayess" };
  var hasName = "name" in record;
  delete record.name;
  return hasBit && hasName && nothing === undefined ? 0 : 1;
}
