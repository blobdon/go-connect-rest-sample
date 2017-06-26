package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

var (
	client       *http.Client
	clientID     string
	clientSecret string
	guid         string
	user         User
)

// data for template
type D map[string]interface{}

// User represents a Microsoft Graph user
type User struct {
	Username string `json:"displayName"`
	Email    string `json:"mail"`
}

type Body struct {
	ContentType string
	Content     string
}
type EmailAddress struct {
	Address string
}
type Recipient struct {
	EmailAddress EmailAddress
}
type Message struct {
	Subject      string
	Body         Body
	ToRecipients []Recipient
}

// # read private credentials from text file
func getCreds(filepath string) (string, string, error) {
	var err error
	var id, secret string
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		id = scanner.Text()
	}
	if scanner.Scan() {
		secret = scanner.Text()
	}
	if id[0] == '*' || secret[0] == '*' {
		err := errors.New("Missing Configuration: _PRIVATE.txt needs to be edited to add client ID and secret")
		return "", "", err
	}
	return id, secret, err
}

// Handler for home page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("/Users/blobdon/code/go/src/github.com/blobdon/go-connect-rest-sample/tpl/connect.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	t.Execute(w, struct{}{})
}

// Handler for login route
func loginHandler(w http.ResponseWriter, r *http.Request) {
	// guid should be long random string, find golang uuid lib?
	// guid used to only accept initiated logins, compared after response later
	if guid == "" {
		guid = "wbvcewiqf923h8vh139fh3491v"
	}
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"User.Read", "Mail.Send"},
		RedirectURL:  "http://localhost:8080/login",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
	}
	var code string
	code = r.URL.Query().Get("code")
	if len(code) == 0 {
		// Redirect user to consent page to ask for permission
		// for the scopes specified above.
		authurl := conf.AuthCodeURL(guid, oauth2.AccessTypeOffline)
		http.Redirect(w, r, authurl, http.StatusSeeOther)
		return
	}
	// Before calling Exchange, be sure to validate FormValue("state").
	if r.FormValue("state") != guid {
		log.Fatal("State has been messed with, end authentication")
		// reset state to prevent re-use
		guid = ""
	}
	ctx := context.Background()
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	client = conf.Client(ctx, tok)
	http.Redirect(w, r, "/main", http.StatusSeeOther)
	return
}

// Handler for logout route
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// reset client to forget token
	client = http.DefaultClient
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}

// Handler for main route
func mainHandler(w http.ResponseWriter, r *http.Request) {
	res, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		log.Println("Failed to get user/me:", err)
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&user)
	if err != nil {
		log.Println("Failed to parse user data:", err)
	}

	t, err := template.ParseFiles("/Users/blobdon/code/go/src/github.com/blobdon/go-connect-rest-sample/tpl/main.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusSeeOther)
	}
	t.Execute(w, D{
		"me":          user,
		"showSuccess": false,
		"showError":   false,
	})
}

// Handler for sendmail route
func sendMailHandler(w http.ResponseWriter, r *http.Request) {
	// Create the email to be sent via the Graph API
	var emailBody bytes.Buffer
	t, err := template.ParseFiles("/Users/blobdon/code/go/src/github.com/blobdon/go-connect-rest-sample/tpl/email.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	t.Execute(&emailBody, user.Username)
	// Gather and encode payload data for posting email message to graph
	recip := Recipient{}
	recip.EmailAddress.Address = r.URL.Query().Get("emailAddress")
	recips := []Recipient{recip}
	msg := D{
		"Message": Message{
			Subject: "Welcome to the Microsoft Graph Connect sample for Python",
			Body: Body{
				ContentType: "HTML",
				Content:     emailBody.String(),
			},
			ToRecipients: recips,
		},
	}
	postJSON := new(bytes.Buffer)
	err = json.NewEncoder(postJSON).Encode(msg)
	if err != nil {
		fmt.Println("error encoding msg to json:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Post the message to the graph API endpoint for sending email
	endpointURL := "https://graph.microsoft.com/v1.0/me/sendMail"
	res, err := client.Post(endpointURL, "application/json", postJSON)
	if err != nil {
		fmt.Println("error posting msg:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Parse template for response to app client
	t2, err := template.ParseFiles("/Users/blobdon/code/go/src/github.com/blobdon/go-connect-rest-sample/tpl/main.html")
	if err != nil {
		fmt.Println("Error parsing template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Graph API will respond with 202 if sendmail request was successful
	if res.StatusCode == 202 {
		t2.Execute(w, D{
			"me":          user,
			"showSuccess": true,
			"showError":   false,
			"recipient":   recip.EmailAddress.Address,
		})
	} else {
		t2.Execute(w, D{
			"me":          user,
			"showSuccess": false,
			"showError":   true,
		})
	}
}

func main() {
	var err error
	clientID, clientSecret, err = getCreds("/Users/blobdon/code/go/src/github.com/blobdon/go-connect-rest-sample/private.txt")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/main", mainHandler)
	http.HandleFunc("/sendmail", sendMailHandler)
	http.ListenAndServe(":8080", nil)
	fmt.Println("Success", client.Head)
}
