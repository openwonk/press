package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
)

var (
	redis_address = "127.0.0.1:6379"
	validPath     = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
	templates     = template.Must(template.ParseFiles("views/edit.html", "views/view.html", "views/parts.html"))
	addr          = flag.Bool("addr", false, "find open address and print to final-port.txt")
)

type Page struct {
	Title string
	Body  []byte
	HTML  template.HTML
}

func (p *Page) save() error {
	conn, err := redis.Dial("tcp", redis_address)
	defer conn.Close()
	if err != nil {
		fmt.Println(err)
	}

	conn.Send("MULTI")
	conn.Send("SET", p.Title, string(p.Body))
	conn.Do("EXEC")

	reply, err := conn.Do("GET", p.Title)
	if err != nil {
		fmt.Println(err)
	}

	values, _ := redis.String(reply, nil)
	fmt.Println("BODY: ", values)

	filename := "data/" + p.Title + ".html"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("invalid page title")
	}
	return m[2], nil // The title is the second subexpression.
}

func loadPage(title string) (*Page, error) {
	conn, err := redis.Dial("tcp", redis_address)
	defer conn.Close()
	if err != nil {
		fmt.Println(err)
	}

	reply, err := conn.Do("GET", title)
	if err != nil {
		fmt.Println(err)
	}

	body, err := redis.String(reply, nil)
	if err != nil {
		return nil, err
	}
	fmt.Println("BODY: ", body)

	return &Page{Title: title, HTML: template.HTML(body)}, nil
	// return &Page{Title: title, Body: body, HTML: template.HTML(body)}, nil

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
	body := r.FormValue("body") // fmt.Println(r.PostForm)
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
		m := validPath.FindStringSubmatch(r.URL.Path) // fmt.Println(m)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func AdminTerminal(port string) {
	fmt.Println("Listening @ " + port)
	exit := false
	for !exit {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("$ ")
		scanner.Scan()

		input := scanner.Text()

		if input == "quit" || input == "exit" {
			exit = true
			break
		} else if input == "reload" {
			templates = template.Must(template.ParseFiles("views/edit.html", "views/view.html", "views/parts.html"))
			input = "reloading templates...\n"
		}

		fmt.Println(input)
	}
}

func ExampleRedis() {
	conn, err := redis.Dial("tcp", redis_address)
	if err != nil {
		fmt.Println(err)
	}

	conn.Send("MULTI")
	conn.Send("SET", "hello", "world!")
	conn.Send("SET", "ilove", "ruby")
	conn.Do("EXEC")

	reply, err := conn.Do("GET", "hello")
	if err != nil {
		fmt.Println(err)
	}

	values, _ := redis.String(reply, nil)
	fmt.Println("REPLY: ", values)
}

func main() {
	port := ":8080"
	flag.Parse()
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	// ExampleRedis(conn)

	if *addr {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile("final-port.txt", []byte(l.Addr().String()), 0644)
		if err != nil {
			log.Fatal(err)
		}
		s := &http.Server{}
		s.Serve(l)
		return
	}

	go http.ListenAndServe(port, nil)

	AdminTerminal(port)

}
