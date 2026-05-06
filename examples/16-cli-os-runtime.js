import "fs";
import "process";
import "stream";
import "child_process";
import "terminal";

function main() {
  fs.mkdirp("temp/jayess-cli-example");
  fs.writeFile("temp/jayess-cli-example/input.txt", "hello");
  const input = fs.createReadStream("temp/jayess-cli-example/input.txt");
  const output = fs.createWriteStream("temp/jayess-cli-example/output.txt");
  stream.pipe(input, output);
  const result = childProcess.exec("jayess", ["--version"]);
  if (terminal.supportsColor(process.stdout)) {
    process.stdout.write(result.stdout);
  } else {
    process.stdout.write("jayess\n");
  }
  return 0;
}
