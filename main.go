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
	User       string            `json:"user"`
	Quizz      string            `json:"quizz"`
	Env        map[string]string `json:"env"`
	Start_date int64             `json:"start_date"`
	End_date   int64             `json:"end_date"`
	Duration   uint64            `json:"duration"`
	Result     map[string]string `json:"result"`
}

var tmpl *template.Template
var tmpl_end *template.Template

var resultsQuizz = make(map[string]string)
var respMessage Response

var questions []question
var qIndex int = 0
var topic string
var externalUri string
var redirectUrl string
var user string

// to simutate the Rest call
// We use the served resources locally
// parameter "category" it a filter for the Quizz category
func getQuizzMock(category string) ([]question, error) {
	s, err := os.ReadFile(fmt.Sprintf("./resources/quizz_%s.yml", category))
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

// Writing locally (test) or sending the Quizz response to the server
func manageResults(r *Response) {
	// fmt.Printf("%v", *r)

	b, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("result.json", b, 0644)
	if err != nil {
		log.Fatal(err)
	}

}

// End quizz linked to the end_goquizz.html template
func endquizz(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		log.Printf("[GET] [index=%d] /endquizz Redirect to /quizz", qIndex)
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
	case "POST":
		// here we send the response to the server
		fmt.Println("Terminating the Quizz ...")
		respMessage.End_date = time.Now().UnixMilli()
		respMessage.Duration = uint64(respMessage.End_date - respMessage.Start_date)
		respMessage.Result = resultsQuizz

		// Writing back to a file the result
		manageResults(&respMessage)

		os.Exit(0)
	default:
		log.Fatalf("Not supported method=%s", r.Method)
	}
}

// quizz execute the Whole quizz
func quizz(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "POST":
		log.Printf("[POST] [index=%d]", qIndex)
		r.ParseForm()
		response := r.Form.Get("answer")

		if qIndex < len(questions) {
			if response != "" {
				log.Printf("[%2d/%2d] Question [%d], Got answer: %s\n", qIndex, len(questions), questions[qIndex].Id, response)
				// Storing result
				resultsQuizz[strconv.Itoa(questions[qIndex].Id)] = response
			} else {
				fmt.Printf("Warning ! No answer for question %d\n", questions[qIndex].Id)
			}
			// Next
			qIndex++

			if qIndex == len(questions) {
				log.Printf("Ending quizz() - [index=%d]", qIndex)
				tmpl_end.Execute(w, "")

			} else {
				log.Printf("[POST] calling template")
				tmpl.Execute(w, questions[qIndex])
			}
		}
	case "GET":
		if qIndex < len(questions) {
			log.Printf("[GET] [index=%d]", qIndex)
			// Here we don't use a redirect as the behaviour is weird
			tmpl.Execute(w, questions[qIndex])
		}
	default:
		log.Fatalf("Not supported method=%s", r.Method)
	}

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
	user = fmt.Sprintf("%s", data["user"])

	// External URI is to get the Quizz data & store the Quizz results
	externalUri = fmt.Sprintf("%s", data["uri"])
	log.Printf("External server URI: %s", externalUri)

	respMessage.Quizz = topic
	log.Printf("Quizz topic is: %s", topic)
	respMessage.User = user

	// os.environment in a filtered map instead of list of string
	respMessage.Env = helper.FilteredEnvValues([]string{"LS_COLORS", "PS1"})

	respMessage.Start_date = time.Now().UnixMilli()
}

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "9091"
	}
	redirectUrl = fmt.Sprintf("http://localhost:%s/quizz", port)

	// We must have a config.yml file
	readConfig()

	// Load the Quizz data
	var err error
	questions, err = getQuizzMock(topic)
	if err != nil {
		fmt.Printf("Error getting QuizzMock data: %s\n", err.Error())
	}

	getRandomResponse := func() int { return rand.Intn(len(questions[qIndex].Options)) }
	// This is the only way i found to show global variable in HTML templates
	// See: https://pkg.go.dev/html/template#Template.Funcs
	// If the template use a fonction not referenced, there is a panic
	var funcs = template.FuncMap{"getCurrentQuestionIndex": func() int { return qIndex + 1 },
		"getNumberOfQuestion": func() int { return len(questions) },
		"getTopic":            func() string { return cases.Title(language.English, cases.Compact).String(topic) },
		"getUser":             func() string { return strings.Split(user, "@")[0] },
		"getRandomResponse":   getRandomResponse}

	mux := http.NewServeMux()
	tmpl, _ = template.New("goquizz.html").Funcs(funcs).ParseFiles("templates/goquizz.html")
	tmpl_end, _ = template.New("end_goquizz.html").Funcs(funcs).ParseFiles("templates/end_goquizz.html")

	// Static resources management
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handlers
	mux.Handle("/", http.RedirectHandler(redirectUrl, http.StatusSeeOther)) // redirect to /quizz
	mux.HandleFunc("/quizz", quizz)
	mux.HandleFunc("/endquizz", endquizz) // is called when quizz is finished

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), mux))

}
