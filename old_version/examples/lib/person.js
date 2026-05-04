export default class Person {
  constructor(name, age) {
    this.name = name;
    this.age = age;
  }

  birthday() {
    this.age = this.age + 1;
    return this.age;
  }

  currentAge() {
    return this.age;
  }
}
