package press

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