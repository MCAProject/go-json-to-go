# JSON-to-Go, converts JSON to a Go struct

Based on original code from ["JSON-to-Go"></a>](https://mholt.github.io/json-to-go) from [Matt Holt](https://github.com/mholt)

Translates JSON into a Go type definition. 

Things to note:

- Ths code sometimes has to make some assumptions, so give the output a once-over.
- In an array of objects, it is assumed that the first object is representative of the rest of them.

Contributions are welcome! Open a pull request to fix a bug, or open an issue to discuss a new feature or change.

### Building

```
# go build -o json-to-go main.go
```

### Usage

- Read JSON file:

  ```sh
  json-to-go sample.json
  ```

- Read JSON file from stdin:

  ```sh
  json-to-go < sample.json
  cat sample.json | json-to-go
  ```

### Credits

Original javascript JSON-to-Go is brought to you by Matt Holt ([mholt6](https://twitter.com/mholt6)).
