package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const MAGIC = "href=\""
const QUOTE = '"'
const HTTP = "HTTP"
const file string = "./websites.db"

type Mapper struct {
	mu   sync.Mutex
	dict map[string]bool
	db   *sql.DB
}

func (m *Mapper) Visited(url string) bool {
	return m.dict[url]
}

func (m *Mapper) Add(url string) {
	m.mu.Lock()
	m.dict[url] = true
	m.mu.Unlock()
}

func (m *Mapper) Insert(url string) {
	_, err := m.db.Exec("INSERT INTO websites VALUES(?, 0);", url)
	if err != nil {
		log.Println("[!] Error on inserting: ", err)
		return
	}
}

func worker(url string, c chan []string) {
	resp, err := http.Get(url)

	if err != nil {
		log.Println("[!] Error on GET: ", err)
		return
	}

	blob, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[!] Error on GET: ", err)
		return
	}
	result := bytes.Index(blob, []byte(MAGIC))

	for result != 1 {
		if result+len(MAGIC) > len(blob) {
			return
		}
		href := blob[result+len(MAGIC):]
		space := bytes.Index(href, []byte{QUOTE})
		if space > 0 {
			link := href[:space]
			if bytes.HasPrefix(link, []byte("http://")) || bytes.HasPrefix(link, []byte("https://")) {
				c <- []string{string(link)}
			}
			blob = href[space+1:]
		}
		result = bytes.Index(blob, []byte(MAGIC))
	}

}

func master(m *Mapper, c chan []string) {

	for urls := range c {
		for _, url := range urls {
			if !m.Visited(url) {
				m.Add(url)
				m.Insert(url)
				fmt.Println(url)
				go worker(url, c)
			}
		}
	}

}

func main() {
	c := make(chan []string)
	starting := "https://en.wikipedia.org/wiki/Bacon"

	sql3, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Println("[!] Error on opening db: ", err)
		return
	}
	m := Mapper{mu: sync.Mutex{}, dict: make(map[string]bool), db: sql3}

	go func(url string) {
		c <- []string{url}
	}(starting)

	master(&m, c)

}
