class Counter {
  static created = 0;
  #value = 0;

  static {
    Counter.created = 1;
  }

  constructor(initial) {
    this.#value = initial;
  }

  increment() {
    this.#value++;
    return this.#value;
  }

  get value() {
    return this.#value;
  }
}

class NamedCounter extends Counter {
  constructor(name, initial) {
    super(initial);
    this.name = name;
  }
}

function main(args) {
  var counter = new NamedCounter("jobs", 2);
  return counter.increment() === 3 && Counter.created === 1 ? 0 : 1;
}
