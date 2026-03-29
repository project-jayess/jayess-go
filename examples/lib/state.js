var counter = 1;
const label = "state";

function bump() {
  counter = counter + 1;
  return counter;
}

export { counter, bump, label as name };

export default function current() {
  return counter;
}
