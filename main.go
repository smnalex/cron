package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/smnalex/cron/parser"
)

func main() {
	cron := flag.String("e", "", "'<minutes> <hours> <days of month> <month> <days of week> <command>'")
	flag.Parse()

	if *cron == "" {
		fmt.Println("Specify a valid expression, e.g '0 1,2 1-2/3 */3 1,3-4/2 echo `hello`'")
		return
	}

	if p, err := parser.Parse(*cron); err != nil {
		log.Printf("unable to parse input: %s", err.Error())
	} else {
		p.PrintTable(os.Stdout)
	}
}
