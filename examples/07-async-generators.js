async function loadValue() {
  return 7;
}

function* range(max) {
  for (var i = 0; i < max; i++) {
    yield i;
  }
}

const service = {
  async fetch() {
    return await loadValue();
  },
  *ids() {
    yield* range(3);
  }
};

function main(args) {
  return service ? 0 : 1;
}
