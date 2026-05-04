import { jayess_add, jayess_greet, jayess_toggle, jayess_make_profile as makeProfile } from "./native/math.c";

function main(args)
{
  var total = jayess_add(3, 4);
  print(total);
  print(jayess_greet("Kimchi"));
  print(jayess_toggle(true));
  print(makeProfile("Kimchi", total));
  return total;
}
