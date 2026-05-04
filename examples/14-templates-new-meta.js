class Box {
  constructor(value) {
    this.value = value;
  }
}

function tag(strings, value) {
  return strings[0] + value;
}

function main(args) {
  var box = new Box(4);
  var text = tag`value:${box.value}`;
  var meta = import.meta;
  return text === "value:4" && meta ? 0 : 1;
}
