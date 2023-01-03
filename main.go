package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

var tmpl *template.Template
var tmpl_end *template.Template

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

var questions []question
var qIndex int = 0

// FuncMap is the way to inject external data to Templates
var getCurrentQuestionIndex = func() int { return qIndex }
var getNumberOfQuestion = func() int { return len(questions) }
var funcs = template.FuncMap{"getCurrentQuestionIndex": getCurrentQuestionIndex,
	"getNumberOfQuestion": getNumberOfQuestion}

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
func getQuizzMock(category string) ([]question, error) {
	s, err := os.ReadFile("./resources/quizz.yml")
	if err != nil {
		return nil, err
	}
	// fmt.Printf("Input: %s\n", s)
	var body interface{}
	err = yaml.Unmarshal(s, &body)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	// Unmarshall to struct question
	q := []question{}

	err = json.Unmarshal(b, &q)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func quizz(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	response := r.Form.Get("answer")

	if response != "" && qIndex < len(questions) {
		fmt.Printf("[%2d/%2d] Question [%d], Got answer: %s\n", qIndex, len(questions), questions[qIndex].Id, response)
	}

	log.Println("Calling quizz()")

	if qIndex < len(questions) {
		tmpl.Execute(w, questions[qIndex])
	} else {
		tmpl_end.Execute(w, "")
		// os.Exit(0)
	}

	// next question
	qIndex++

}

func main() {

	fmt.Printf("Go go go ...\n")
	var err error
	questions, err = getQuizzMock("kubernetes")
	if err != nil {
		fmt.Printf("Error getting QuizzMock data: %s\n", err.Error())
	}

	// fmt.Printf("Got: %v", q)

	mux := http.NewServeMux()
	// tmpl = template.Must(template.ParseFiles("templates/index.html"))
	tmpl, _ = template.New("goquizz.html").Funcs(funcs).ParseFiles("templates/goquizz.html")
	tmpl_end, _ = template.New("end_goquizz.html").Funcs(funcs).ParseFiles("templates/end_goquizz.html")

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)
	// mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/quizz", quizz)

	log.Fatal(http.ListenAndServe(":9091", mux))

}
