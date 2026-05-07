# Internal Storage

Jayess internal storage is a small dependency-free key/value API for simple
apps. It is not a SQLite clone and does not provide SQL, joins, indexes,
transactions, or query planning.

## API Shape

- `storage.open(path)` opens or creates a persistent store.
- `storage.put(store, key, value)` writes a string value.
- `storage.get(store, key)` reads a string value.
- `storage.delete(store, key)` removes a key.
- `storage.scan(store, prefix)` returns key/value entries sorted by key.
- `storage.close(store)` flushes pending state and closes the store.

## Persistence

The current runtime helper persists a JSON-backed map. This keeps the default
storage surface reviewable and fully internal while giving CLI tools and small
apps a simple local persistence option.

## Example

```js
function main() {
  const db = storage.open("./data/app.store.json");
  storage.put(db, "user:1", "Ada");
  storage.put(db, "user:2", "Grace");

  const users = storage.scan(db, "user:");
  console.log(users.length);

  storage.delete(db, "user:1");
  storage.close(db);
  return 0;
}
```

Use the optional SQLite package only when an app needs SQL compatibility.
