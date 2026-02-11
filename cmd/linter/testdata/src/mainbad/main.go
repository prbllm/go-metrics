package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("main without forbidden calls")
}

func helper() {
	panic("panic outside main") // want "paniccheck: panic"

	log.Fatal("fatal outside main")     // want "paniccheck: log.Fatal"
	log.Fatalf("fatalf outside main")   // want "paniccheck: log.Fatalf"
	log.Fatalln("fatalln outside main") // want "paniccheck: log.Fatalln"

	log.Print("this is allowed")

	os.Exit(1) // want "paniccheck: os.Exit"
}

type S struct{}

func (S) method() {
	panic("method panic outside main") // want "paniccheck: panic"
}
