function main() {
  const data = {
    items: [
      { name: "kimchi", spicy: 10 },
      { name: "jjigae", spicy: 7 }
    ]
  };

  print(data.items[0].name);
  print(data.items[1].spicy + 3);
  return data.items[0].spicy + data.items[1].spicy;
}
