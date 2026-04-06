function main(args) {
  var maybe = undefined;
  maybe ??= "kimchi";
  var name = "";
  name ||= "jjigae";
  var ready = true;
  ready &&= false;
  print(maybe, name, ready);
  return 0;
}
