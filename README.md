# Transform

[![Test & Build](https://github.com/zeiss/go-transform/actions/workflows/main.yml/badge.svg)](https://github.com/zeiss/go-transform/actions/workflows/main.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/zeiss/go-transform.svg)](https://pkg.go.dev/github.com/zeiss/go-transform)
[![Go Report Card](https://goreportcard.com/badge/github.com/zeiss/go-transform)](https://goreportcard.com/report/github.com/zeiss/go-transform)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Taylor Swift](https://img.shields.io/badge/secured%20by-taylor%20swift-brightgreen.svg)](https://twitter.com/SwiftOnSecurity)

Transform is a Go library that provides a simple way to transform data from one format to another.

## Installation

```bash
go get github.com/zeiss/go-transform
```

## Usage

```go
type example struct {
  Name string `tansform:"trim,lowercase"`
}

t := transform.New()
e := example{Name: "  John Doe  "}

if err := t.Transform(&e); err != nil {
  log.Fatal(err)
}

fmt.Println(e.Name) // Output: john doe
```

## Transformations

This is the list of all available transformations:

| Function | Description |
| --- | --- |
| `trim` | Removes leading and trailing whitespace. |
| `lowercase` | Converts the string to lowercase. |
| `uppercase` | Converts the string to uppercase. |
| `rtrim` | Removes trailing whitespace. |
| `ltrim` | Removes leading whitespace. |
| `uppercase` | Converts the string to uppercase. |

## License

[MIT](/LICENSE)
