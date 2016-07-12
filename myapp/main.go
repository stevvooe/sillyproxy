package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

var tmplString = `
<h1>{{.Hostname}} ({{.Color}})</h1>
`

func main() {
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	tmpl := template.Must(template.New("index").Parse(tmplString))

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		color := r.Header.Get("Color")
		if color == "" {
			color = "unknown"
		}

		log.Println(r.Method, r.RequestURI, r.RemoteAddr, color)

		if os.Getenv("V2") == "" {
			fmt.Fprintln(w, color, hostname)
		} else {
			if err := tmpl.Execute(w, struct {
				Hostname string
				Color    string
			}{
				Hostname: hostname,
				Color:    color,
			}); err != nil {
				log.Println(err)
			}
		}

	})

	fmt.Println("listen", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalln(err)
	}
}
