import { Animal, Dog } from "./lib/animal.js";

function main(args) {
  print(Animal.kind());
  print(Dog.label());
  var dog = new Dog("Bori", 2);
  print(dog.sound());
  print(dog.birthday());
  return dog.birthday();
}
