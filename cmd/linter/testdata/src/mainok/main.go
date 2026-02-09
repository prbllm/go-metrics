package main

import (
	"log"
	"os"
)

func main() {
	panic("panic inside main is allowed")

	log.Fatal("fatal inside main is allowed")
	log.Fatalf("fatalf inside main is allowed")
	log.Fatalln("fatalln inside main is allowed")

	os.Exit(1)
}
