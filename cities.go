package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"strings"
)

//go:embed cities.json
var citiesJSON []byte

var cities map[string][]string

func init() {
	if err := json.Unmarshal(citiesJSON, &cities); err != nil {
		log.Fatalln(err)
	}
}

func getCities(region string) []string {
	if c, ok := cities[region]; ok {
		return c
	}
	for name := range cities {
		if strings.Contains(name, region) {
			return cities[name]
		}
	}
	return nil
}
