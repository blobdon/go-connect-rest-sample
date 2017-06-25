package main

import (
	"bufio"
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
)

// data for template
type D map[string]interface{}

// User represents a Microsoft Graph user
type User struct {
	Username string `json:"displayName"`
	Email    string `json:"mail"`
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
	// if err := scanner.Err(); err != nil {
	// 	log.Fatal(err)
	// }
	return id, secret, err
}

// # since this sample runs locally without HTTPS, disable InsecureRequestWarning
// requests.packages.urllib3.disable_warnings()

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
	fmt.Println("First Code", len(code), code, code == "")
	if len(code) == 0 {
		// Redirect user to consent page to ask for permission
		// for the scopes specified above.
		authurl := conf.AuthCodeURL(guid, oauth2.AccessTypeOffline)
		fmt.Printf("Visit the URL for the auth dialog: %v", authurl)
		http.Redirect(w, r, authurl, http.StatusSeeOther)
		return
	}
	// Before calling Exchange, be sure to validate FormValue("state").
	if r.FormValue("state") != guid {
		log.Fatal("State has been messed with, end authentication")
		// reset state to prevent re-use
		guid = ""
	}
	fmt.Println("Second Code", len(code), code)
	ctx := context.Background()
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	client = conf.Client(ctx, tok)
	http.Redirect(w, r, "/main", http.StatusSeeOther)
	return
}

// @app.route('/logout')
// def logout():
// Handler for logout route
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// reset client to forget token
	client = http.DefaultClient
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}

//     session.pop('microsoft_token', None)
//     session.pop('state', None)
//     return redirect(url_for('index'))

// def authorized():
//     response = msgraphapi.authorized_response()

//     if response is None:
//         return "Access Denied: Reason={0}\nError={1}".format( \
//             request.args['error'], request.args['error_description'])

// @app.route('/main')
// Handler for main route
func mainHandler(w http.ResponseWriter, r *http.Request) {
	me := User{}
	res, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		log.Println("Failed to get user/me:", err)
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&me)
	if err != nil {
		log.Println("Failed to parse user data:", err)
	}

	t, err := template.ParseFiles("/Users/blobdon/code/go/src/github.com/blobdon/go-connect-rest-sample/tpl/main.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusSeeOther)
	}
	t.Execute(w, D{
		"me":          me,
		"showSuccess": false,
		"showError":   false,
	})
}

// def main():
//     if session['alias']:
//         username = session['alias']
//         email_address = session['userEmailAddress']
//         return render_template('main.html', name=username, emailAddress=email_address)
//     else:
//         return render_template('main.html')

// @app.route('/send_mail')
// def send_mail():
//     """Handler for send_mail route."""
//     email_address = request.args.get('emailAddress') # get email address from the form
//     response = call_sendmail_endpoint(session['access_token'], session['alias'], email_address)
//     if response == 'SUCCESS':
//         show_success = 'true'
//         show_error = 'false'
//     else:
//         print(response)
//         show_success = 'false'
//         show_error = 'true'

//     session['pageRefresh'] = 'false'
//     return render_template('main.html', name=session['alias'],
//                            emailAddress=email_address, showSuccess=show_success,
//                            showError=show_error)

// # If library is having trouble with refresh, uncomment below and implement
// # refresh handler see https://github.com/lepture/flask-oauthlib/issues/160 for
// # instructions on how to do this. Implements refresh token logic.
// # @app.route('/refresh', methods=['POST'])
// # def refresh():
// @msgraphapi.tokengetter
// def get_token():
//     """Return the Oauth token."""
//     return session.get('microsoft_token')

// def call_sendmail_endpoint(access_token, name, email_address):
//     """Call the resource URL for the sendMail action."""
//     send_mail_url = 'https://graph.microsoft.com/v1.0/me/microsoft.graph.sendMail'

// 	# set request headers
//     headers = {'User-Agent' : 'python_tutorial/1.0',
//                'Authorization' : 'Bearer {0}'.format(access_token),
//                'Accept' : 'application/json',
//                'Content-Type' : 'application/json'}

// 	# Use these headers to instrument calls. Makes it easier to correlate
//     # requests and responses in case of problems and is a recommended best
//     # practice.
//     request_id = str(uuid.uuid4())
//     instrumentation = {'client-request-id' : request_id,
//                        'return-client-request-id' : 'true'}
//     headers.update(instrumentation)

// 	# Create the email that is to be sent via the Graph API
//     email = {'Message': {'Subject': 'Welcome to the Microsoft Graph Connect sample for Python',
//                          'Body': {'ContentType': 'HTML',
//                                   'Content': render_template('email.html', name=name)},
//                          'ToRecipients': [{'EmailAddress': {'Address': email_address}}]
//                         },
//              'SaveToSentItems': 'true'}

//     response = requests.post(url=send_mail_url,
//                              headers=headers,
//                              data=json.dumps(email),
//                              verify=False,
//                              params=None)

//     if response.ok:
//         return 'SUCCESS'
//     else:
//         return '{0}: {1}'.format(response.status_code, response.text)

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
	http.ListenAndServe(":8080", nil)
	fmt.Println("Success", client.Head)
}
