function risky(value) {
  if (value < 0) {
    throw new Error("negative");
  }
  return value;
}

function main(args) {
  try {
    return risky(1) === 1 ? 0 : 1;
  } catch (error) {
    return 2;
  } finally {
    debugger;
  }
}
