package main

import (
	"fmt"
	"time"

	"github.com/TheVovchenskiy/confgo"
)

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func main() {
	cm, err := confgo.NewConfigManagerFor[Config](confgo.WithDynamicJSONFile("examples/dynamic/config.json",
		func() {
			fmt.Printf("Config has been updated!\n")
		},
		func(err error) {
			fmt.Printf("Error while updating config: %v\n", err)
		}))
	if err != nil {
		panic(err)
	}
	cm.MustStart()
	defer cm.MustStop()

	for {
		fmt.Printf("%#v\n", cm.Config())
		time.Sleep(3 * time.Second)
	}
}
