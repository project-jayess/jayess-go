const defaults = { retries: 2, verbose: false };
const config = { ...defaults, verbose: true };
const values = [1, 2, 3, 4];

function main(args) {
  var [first, , third] = values;
  var { retries, verbose: enabled } = config;
  var record = {
    first,
    retries,
    get ready() {
      return enabled && this.retries > 0;
    },
    set ready(value) {
      this.retries = value ? this.retries : 0;
    },
    total() {
      return first + third + this.retries;
    }
  };
  return record.ready && record.total() === 6 ? 0 : 1;
}
