package main

import (
	"encoding/json"
	"fmt"
	"gokubquizz/helper"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

// What we keep after a Quizz execution
type Response struct {
	user       string
	quizz      string
	env        map[string]string
	start_date int64
	end_date   int64
	duration   uint64
	result     map[string]string
}

var resultsQuizz = make(map[string]string)

var respMessage Response

var questions []question
var qIndex int = 0
var topic string
var externalUri string

// FuncMap is the way to inject external data to Templates
var getCurrentQuestionIndex = func() int { return qIndex + 1 }
var getNumberOfQuestion = func() int { return len(questions) }
var getTopic = func() string { return cases.Title(language.English, cases.Compact).String(topic) }

var funcs = template.FuncMap{"getCurrentQuestionIndex": getCurrentQuestionIndex,
	"getNumberOfQuestion": getNumberOfQuestion,
	"getTopic":            getTopic}

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

	body = helper.Convert(body)
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

// Writing or sending the Quizz response
func manageResults(r *Response) {
	fmt.Printf("%v", *r)
}

// End quizz linked to the end_goquizz.html template
func endquizz(w http.ResponseWriter, r *http.Request) {
	// here we send the response to the server
	fmt.Println("Terminating the Quizz ...")
	respMessage.end_date = time.Now().UnixMilli()
	respMessage.duration = uint64(respMessage.end_date - respMessage.start_date)
	respMessage.result = resultsQuizz

	// fmt.Printf("---> %v", respMessage)
	// Writing back to a file the result
	manageResults(&respMessage)

	os.Exit(0)
}

func quizz(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		r.ParseForm()
		response := r.Form.Get("answer")

		if qIndex < len(questions) {
			if response != "" {
				fmt.Printf("[%2d/%2d] Question [%d], Got answer: %s\n", qIndex, len(questions), questions[qIndex].Id, response)
				// Storing result
				resultsQuizz[strconv.Itoa(questions[qIndex].Id)] = response
			} else {
				fmt.Printf("Warning ! No answer for question %d\n", questions[qIndex].Id)
			}
		}
	}

	log.Printf("Calling quizz() - [index=%d]", qIndex)

	if qIndex < len(questions) {
		tmpl.Execute(w, questions[qIndex])
	} else {
		tmpl_end.Execute(w, "")
	}

	// next question
	qIndex++

}

// Read the external yaml config and prepare the Response message
func readConfig() {
	s, err := os.ReadFile("./config.yml")
	if err != nil {
		log.Fatalf("Uname to read the ./config.yml file, reason: %v", err.Error())
	}
	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(s, &data)
	if err != nil {
		log.Fatalf("Uname to unmarshall the ./config.yml Yaml file, reason: %v", err.Error())
	}
	// Quizz category in lowercase
	topic = strings.ToLower(fmt.Sprintf("%s", data["category"]))
	user := fmt.Sprintf("%s", data["user"])

	// External URI is to get the Quizz data & store the Quizz results
	externalUri = fmt.Sprintf("%s", data["uri"])
	log.Printf("External server URI: %s", externalUri)

	respMessage.quizz = topic
	respMessage.user = user

	// os.environment in a filtered map instead of list of string
	respMessage.env = helper.FilteredEnvValues([]string{"LS_COLORS", "PS1"})

	respMessage.start_date = time.Now().UnixMilli()
}

func main() {

	fmt.Printf("Go go go ...\n")

	readConfig()
	fmt.Printf("---> %v", respMessage)

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
	mux.HandleFunc("/endquizz", endquizz)

	log.Fatal(http.ListenAndServe(":9091", mux))

}
