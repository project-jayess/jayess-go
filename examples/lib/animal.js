export class Animal {
  static kingdom = "animal";

  static kind() {
    return Animal.kingdom;
  }

  constructor(name) {
    this.name = name;
  }

  sound() {
    return this.name;
  }
}

export class Dog extends Animal {
  static label() {
    return Dog.kind();
  }

  constructor(name, age) {
    super(name);
    this.age = age;
  }

  birthday() {
    this.age = this.age + 1;
    return this.age;
  }

  sound() {
    return super.sound();
  }
}
