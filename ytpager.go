package main

import (
	"asdatapel/ytpager/auth"
	"asdatapel/ytpager/model"
	"asdatapel/ytpager/service"
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/gorilla/sessions"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func ShiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

type RootRoute struct {
	AuthRoute      *auth.AuthRoute
	VideoListRoute *VideoListRoute
}

func (h RootRoute) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	fmt.Println(req.URL.Path)

	session, err := store.Get(req, "cookie-name")
	if err != nil {
		http.Error(res, fmt.Errorf("failed to get user session: %v", err).Error(), http.StatusInternalServerError)
	}

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	if head == "" || head == "home" {
		body, _ := ioutil.ReadFile("html/home.html")
		tmpl, _ := template.New("name").Parse(string(body))
		tmpl.Execute(res, 0)
	} else if head == "static" {
		http.FileServer(http.Dir("static")).ServeHTTP(res, req)
	} else if head == "videos" {
		h.VideoListRoute.ServeHTTP(res, req, session)
	} else if head == "auth" {
		h.AuthRoute.ServeHTTP(res, req, session)
	}
}

type VideoListRoute struct {
	ListTemplate *template.Template
}

func (h VideoListRoute) ServeHTTP(res http.ResponseWriter, req *http.Request, session *sessions.Session) {
	channelName, rest := ShiftPath(req.URL.Path)

	if channelName == "" {
		http.Error(res, "Need a channel", http.StatusNotFound)
		return
	}

	requestedPage, _ := ShiftPath(rest)
	if requestedPage == "" {
		http.Redirect(res, req, "/videos/"+channelName+"/1", 301)
		return
	}

	ctx := context.Background()

	client := auth.GetClient(ctx, session)
	videoService := service.NewYoutube(ctx, client)

	pageNum, _ := strconv.ParseInt(requestedPage, 0, 0)

	channel := videoService.GetChannel(channelName)
	videos, _ := videoService.ListVideos(&channel, pageNum)

	videoPage := model.VideoPage{
		Index:   pageNum,
		Videos:  videos,
		Channel: &channel,
	}

	h.ListTemplate.Execute(res, videoPage)
}

func main() {
	body, _ := ioutil.ReadFile("html/page.html")
	tmpl, _ := template.New("name").Funcs(sprig.FuncMap()).Parse(string(body))

	rootRoute := &RootRoute{
		VideoListRoute: &VideoListRoute{
			ListTemplate: tmpl,
		},
		AuthRoute: &auth.AuthRoute{},
	}

	log.Fatal(http.ListenAndServe(":80", rootRoute))
}
