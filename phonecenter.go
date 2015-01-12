package main

import (
	"code.google.com/p/goauth2/oauth"
	"flag"
	"fmt"
	calendar "google.golang.org/api/calendar/v3"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var config = &oauth.Config{
	ClientId:     "",
	ClientSecret: "",
	Scope:        "https://www.googleapis.com/auth/calendar",
	RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
	AuthURL:      "https://accounts.google.com/o/oauth2/auth",
	TokenURL:     "https://accounts.google.com/o/oauth2/token",
}
var addr = flag.String("addr", ":1718", "http service address")
var whisperTmpl = template.Must(template.New("qr").Parse(whisperStr))
var callTmpl = template.Must(template.New("call").Parse(callStr))
var (
	refreshToken, calendarId string
)

func main() {
	config.ClientId = os.Getenv("GOOGLE_CLIENT_ID")
	config.ClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	calendarId = os.Getenv("CALENDAR_ID")
	refreshToken = os.Getenv("REFRESH_TOKEN")

	http.Handle("/call", http.HandlerFunc(CallHandler))
	http.Handle("/whisper", http.HandlerFunc(WhisperHandler))
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func CallHandler(w http.ResponseWriter, req *http.Request) {
	var token *oauth.Token
	if refreshToken == "" {
		// Generate a URL to visit for authorization.
		authUrl := config.AuthCodeURL("state")
		log.Printf("Go to the following link in your browser: %v\n", authUrl)
		tr := &oauth.Transport{
			Config:    config,
			Transport: http.DefaultTransport,
		}

		// Read the code, and exchange it for a token.
		var code string
		fmt.Scanln(&code)
		_, err := tr.Exchange(code)
		if err != nil {
			log.Fatalf("An error occurred exchanging the code: %v\n", err)
		}

		fmt.Println(tr.Token)
		token = tr.Token
	} else {
		token = &oauth.Token{RefreshToken: refreshToken}
	}

	t := &oauth.Transport{
		Token:     token,
		Config:    config,
		Transport: http.DefaultTransport,
	}

	svc, err := calendar.New(t.Client())
	if err != nil {
		log.Fatalf("An error occurred creating Calendar client: %v\n", err)
	}

	res, err := svc.Events.List(calendarId).TimeMin("2015-01-11T0:0:00.0Z").TimeMax("2015-01-12T0:0:00.0Z").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve calendar events list: %v", err)
	}
	now := time.Now()
	var startTime time.Time
	var endTime time.Time
	phoneNumbers := []string{}
	for _, v := range res.Items {
		fmt.Printf("%#v\n", v.Start.DateTime)
		fmt.Printf("%#v\n", v.Start.Date)
		if v.Start.DateTime != "" {
			fmt.Println("OK?")
			startTime, _ = time.Parse(time.RFC3339, v.Start.DateTime)
		} else {
			startTime, _ = time.Parse(time.RFC3339, v.Start.Date+"T00:00:00-05:00")
		}
		if v.End.DateTime != "" {
			endTime, _ = time.Parse(time.RFC3339, v.End.DateTime)
		} else {
			endTime, _ = time.Parse(time.RFC3339, v.End.Date+"T00:00:00-05:00")
		}

		fmt.Println(startTime)
		fmt.Println(endTime)

		if now.After(startTime) && now.Before(endTime) {
			fmt.Println("OO")
			phoneNumbers = append(phoneNumbers, parseNumbers(v.Location)...)
		}
		log.Printf("Calendar ID %q event: %v: %q\n", calendarId, v.Updated, v.Summary, v.Location)
	}

	setHeaders(w)
	callTmpl.Execute(w, phoneNumbers)
}

func WhisperHandler(w http.ResponseWriter, req *http.Request) {
	setHeaders(w)
	whisperTmpl.Execute(w, "")
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/xml")
}

func parseNumbers(numberString string) []string {
	return strings.Split(numberString, ",")
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

const callStr = `
<Response><Dial timeout="60">{{range .}}<Number url="whisper">{{.}}</Number>{{end}}</Dial></Response>
`
const whisperStr = `
<Response><Gather numDigits="1" finishOnKey="" timeout="3"><Say>Namely Support Call. Press any key.</Say></Gather></Response>
`
