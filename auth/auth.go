package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/gorilla/sessions"
)

func GetClient(ctx context.Context, session *sessions.Session) *http.Client {
	token := new(oauth2.Token)
	token.AccessToken = session.Values["AccessToken"].(string)
	token.RefreshToken = session.Values["RefreshToken"].(string)
	t := session.Values["Expiry"].(string)
	token.Expiry, _ = time.Parse(time.RFC3339, t)
	token.TokenType = session.Values["TokenType"].(string)

	return conf.Client(ctx, token)
}

type AuthRoute struct {
}

func (h AuthRoute) ServeHTTP(res http.ResponseWriter, req *http.Request, session *sessions.Session) {
	// var head string
	// head, req.URL.Path = ShiftPath(req.URL.Path)

	if req.URL.Path == "/login" {
		serveLoginHTTP(res, req, session)
	} else if req.URL.Path == "/oauthurl" {
		serveAuthHTTP(res, req, session)
	}
	// else if head == "videos" {
	// 	h.VideoListRoute.ServeHTTP(res, req)
	// }
}

// Credentials which stores google ids.
type Credentials struct {
	Cid     string `json:"client_id"`
	Csecret string `json:"client_secret"`
}

// User is a retrieved and authentiacted user.
type User struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Gender        string `json:"gender"`
}

var cred Credentials
var conf *oauth2.Config
var state string

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func init() {
	file, err := ioutil.ReadFile("./creds.json")
	if err != nil {
		log.Printf("File error: %v\n", err)
		os.Exit(1)
	}
	json.Unmarshal(file, &cred)

	conf = &oauth2.Config{
		ClientID:     cred.Cid,
		ClientSecret: cred.Csecret,
		RedirectURL:  "http://asadpatelytpager.mooo.com/auth/oauthurl",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email", // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
			"https://www.googleapis.com/auth/youtube",        // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
		},
		Endpoint: google.Endpoint,
	}
}

// func indexHandler(c *gin.Context) {
// 	c.HTML(http.StatusOK, "index.tmpl", gin.H{})
// }

func getLoginURL(state string) string {
	return conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func serveLoginHTTP(res http.ResponseWriter, req *http.Request, session *sessions.Session) {
	state = randToken()
	session.Values["state"] = state
	session.Save(req, res)
	res.Write([]byte("<html><title>Golang Google</title> <body> <a href='" + getLoginURL(state) + "'><button>Login with Google!</button> </a> </body></html>"))
}

func serveAuthHTTP(res http.ResponseWriter, req *http.Request, session *sessions.Session) {
	// Handle the exchange code to initiate a transport.
	retrievedState := session.Values["state"]
	if retrievedState != req.URL.Query().Get("state") {
		http.Error(res, fmt.Sprintf("Invalid session state: %s", retrievedState), http.StatusUnauthorized)
		return
	}

	tok, err := conf.Exchange(oauth2.NoContext, req.URL.Query().Get("code"))
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	client := conf.Client(oauth2.NoContext, tok)
	email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	defer email.Body.Close()
	data, _ := ioutil.ReadAll(email.Body)
	log.Println("Email body: ", string(data))

	session.Values["AccessToken"] = tok.AccessToken
	session.Values["RefreshToken"] = tok.RefreshToken
	session.Values["TokenType"] = tok.TokenType
	session.Values["Expiry"] = tok.Expiry.Format(time.RFC3339)
	session.Save(req, res)

	http.Redirect(res, req, "/home", http.StatusPermanentRedirect)
}
