package nonmain

import (
	"log"
	"os"
)

func main() {
	panic("panic in non-main package") // want "paniccheck: panic"

	log.Fatal("fatal in non-main package") // want "paniccheck: log.Fatal"

	os.Exit(1) // want "paniccheck: os.Exit"
}
