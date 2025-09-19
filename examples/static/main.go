package main

import (
	"fmt"

	"github.com/TheVovchenskiy/confgo"
)

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func main() {
	cm, err := confgo.NewConfigManagerFor[Config](confgo.WithJSONFile("examples/static/config.json"))
	if err != nil {
		panic(err)
	}
	cm.MustStart()
	defer cm.MustStop()

	fmt.Printf("%#v\n", cm.Config())
}
