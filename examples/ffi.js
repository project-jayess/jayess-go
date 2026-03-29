native import "./native/math.c";
extern function jayess_add(a, b);
extern function jayess_greet(name);
extern function jayess_toggle(value);
extern function jayess_make_profile(name, score);

function main(args)
{
  var total = jayess_add(3, 4);
  print(total);
  print(jayess_greet("Kimchi"));
  print(jayess_toggle(true));
  print(jayess_make_profile("Kimchi", total));
  return total;
}
