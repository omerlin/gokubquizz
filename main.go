package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Transform a Yaml to a generic struct using generics
func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}

// to simutate the Rest call
// We use the served resources locally
func getQuizzMock() []byte {
	s, err := ioutil.ReadFile("./resources/quizz.yml")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Input: %s\n", s)
	var body interface{}
	if err := yaml.Unmarshal([]byte(s), &body); err != nil {
		panic(err)
	}

	body = convert(body)
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return b
}

func main() {
	fmt.Printf("Output: %s\n", getQuizzMock())
}
