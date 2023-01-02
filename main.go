package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	"gopkg.in/yaml.v2"
)

// Structs for JSON must be capitalized
type options struct {
	Id    int    `json:"id"`
	Label string `json:"label"`
}

type question struct {
	Id       int       `json:"id"`
	Question string    `json:"question"`
	Options  []options `json:"options"`
}

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
// parameter "category" it a filter for the Quizz category
func getQuizzMock(category string) []byte {
	s, err := ioutil.ReadFile("./resources/quizz.yml")
	if err != nil {
		panic(err)
	}
	// fmt.Printf("Input: %s\n", s)
	var body interface{}
	if err := yaml.Unmarshal([]byte(s), &body); err != nil {
		panic(err)
	}

	body = convert(body)
	// fmt.Printf("type: %T\n", body)
	// We need to declare the structure type to get only the category
	// We need also to declare that the output is of type "[]interface{}" to be able to apply a len() function
	subbody := body.(map[string]interface{})[category].([]interface{})

	// We shuffle data reinitializing the rand seed (to not have same answer)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(subbody), func(i, j int) { subbody[i], subbody[j] = subbody[j], subbody[i] })

	b, err := json.Marshal(subbody)
	if err != nil {
		panic(err)
	}
	return b
}

func main() {
	jsonBytes := getQuizzMock("kubernetes")
	// fmt.Printf("Json: %s", jsonBytes)
	q := []question{}

	err := json.Unmarshal(jsonBytes, &q)
	if err != nil {
		fmt.Println("Error : %v", err)
	}
	fmt.Printf("Got: %v", q)
}
