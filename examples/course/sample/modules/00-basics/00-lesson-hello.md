---
title: Hello, Go
---

Every Go program starts in a function called `main`:

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, Go!")
}
```

`fmt.Println` prints a line. Save this as `hello.go` and run `go run hello.go`.
