// Copyright 2022 Larry Rau. All rights reserved.
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// code borrowed from:https://go.dev/doc/articles/wiki/final.go

package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/larryr/lsrv/lsrv"
)

type Page struct {
	Title string
	Body  []byte
}

var validPath *regexp.Regexp
var templates *template.Template

func init() {
	setup()

	templates = template.Must(template.ParseFiles("edit.html", "view.html"))
	validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

}

const (
	addr    = "0.0.0.0"
	secCert = "cert.pem"
	secKey  = "key.pem"
)

var (
	notls   *bool   = flag.Bool("notls", false, "use unsecure http")
	gencert *bool   = flag.Bool("gencert", false, "generate cert and key")
	port    *int    = flag.Int("port", 8080, "port for server to listen on")
	host    *string = flag.String("host", "localhost", "host name for certificate")
)

func main() {
	flag.Parse()

	var srvAddr = fmt.Sprintf("%s:%d", addr, *port)
	fmt.Printf("lsrv: listening: %s\n", srvAddr)
	log.Printf("args: notls=%v gencert=%v port=%v host=%v", *notls, *gencert, *port, *host)

	if *gencert {
		// generate certificate/key and exit.
		err := lsrv.GenerateCert(*host, "")
		if err != nil {
			log.Fatalf("error generating cert:%v", err)
		}
		log.Printf("certificate/key generated!\n")
		return
	}

	// setup to run server
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "content/"+r.URL.Path[1:])
	})

	var err error
	if *notls {
		err = http.ListenAndServe(srvAddr, nil)
	} else {
		err = http.ListenAndServeTLS(srvAddr, secCert, secKey, nil)
	}
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Printf("exiting!\n")
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

var (
	edit_html string = `
<h1>Editing {{.Title}}</h1>

<form action="/save/{{.Title}}" method="POST">
<div><textarea name="body" rows="20" cols="80">{{printf "%s" .Body}}</textarea></div>
<div><input type="submit" value="Save"></div>
</form>`

	view_html string = `
	<h1>{{.Title}}</h1>
<p>[<a href="/edit/{{.Title}}">edit</a>]</p>
<div>{{printf "%s" .Body}}</div>
`
)

func setup() {

	makeFile("edit.html", edit_html)

	makeFile("view.html", view_html)

}

func makeFile(name, content string) {
	out, err := os.Create(name)
	if err == nil {
		out.WriteString(content)
	}
}
