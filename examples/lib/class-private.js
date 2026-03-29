export class BaseCounter {
  value = 1;
  #secret = 9;

  constructor(step) {
    this.step = step;
  }

  readSecret() {
    return this.#secret;
  }
}

export class DeepCounter extends BaseCounter {
  label = "deep";
  #code = 7;
  static family = "counter";
  static #hiddenFamily = "shadow";

  static familyName() {
    return this.family;
  }

  static #hiddenName() {
    return this.#hiddenFamily;
  }

  constructor(step, boost) {
    super(step);
    this.boost = boost;
  }

  total() {
    return super.value + this.boost + this.#code;
  }

  reveal() {
    return super.readSecret() + this.#code;
  }

  static shadow() {
    return this.#hiddenName();
  }
}
