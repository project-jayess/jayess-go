function main(args)
{
  const delay = 500;
  const kimchi = {};
  kimchi.a = "kimchi";
  print(kimchi.a);
  print(args);
  print("Jayess says hello.");
  sleep(delay);
  var name = readLine("What is your name? ");
  print(name);
  print(args[0]);
  readKey("Press any key to continue");
  return 0;
}
