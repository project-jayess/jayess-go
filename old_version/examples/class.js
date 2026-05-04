import Person from "./lib/person.js";

function main(args) {
  var person = new Person("Kimchi", 12);
  print(person.name);
  print(person.currentAge());
  print(person.birthday());
  return person.currentAge();
}
