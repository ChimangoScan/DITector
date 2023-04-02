package db

import (
	"fmt"
	"testing"
)

func TestEscapeString(t *testing.T) {
	fmt.Println(EscapeString(`\0`))
	fmt.Println(EscapeString("\"hello' \\\nworld'\""))
}
