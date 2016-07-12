package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var (
	addr   string
	logger = log.New(os.Stderr, "sillyproxy: ", log.Flags())
)

func init() {
	flag.StringVar(&addr, "addr", ":8080", "listen address")
}

func main() {
	var (
		colors   = []string{"blue", "green"}
		backends []backend
	)

	flag.Parse()
	if flag.NFlag() > 0 {
		colors = append(colors, flag.Args()...)
	}

	log.Println("colors", colors)
	for _, color := range colors {
		backend, err := getColorServiceFromEnv(color)
		if err != nil {
			log.Println(err)
			continue
		}
		backends = append(backends, backend)
	}

	rp := httputil.ReverseProxy{
		Director: weightedDirector(backends...),
		ErrorLog: logger,
	}

	http.HandleFunc("/down", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintln(w, "no backends available")
	})

	http.Handle("/", &rp)

	log.Println("listen", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalln(err)
	}
}

func getColorServiceFromEnv(color string) (backend, error) {
	color = strings.ToUpper(color)
	serviceEnv := fmt.Sprintf("%s", color)
	service := os.Getenv(serviceEnv)
	serviceWeight := os.Getenv(serviceEnv + "_WEIGHT")

	if service == "" {
		return backend{}, errors.Errorf("service %v missing in env, set %v", color, serviceEnv)
	}

	if serviceWeight == "" {
		serviceWeight = "1" // default
	}

	return parseBackend(strings.ToLower(color), service, serviceWeight)
}

type backend struct {
	color  string
	url    *url.URL
	weight int
}

func parseBackend(color, location, weight string) (backend, error) {
	u, err := url.Parse(location)
	if err != nil {
		return backend{}, err
	}

	if u.Scheme == "" {
		return backend{}, errors.New("scheme required")
	}

	if u.Host == "" {
		return backend{}, errors.New("host required")
	}

	w, err := strconv.Atoi(weight)
	if err != nil {
		log.Println("invalid weight", weight)
		if weight != "" {
			return backend{}, err
		}

		w = 1
	}

	return backend{
		color:  color,
		url:    u,
		weight: w,
	}, nil
}

func (b backend) String() string {
	return fmt.Sprintf("backend{color: %v, url: %v, weight: %v}", b.color, b.url, b.weight)
}

func weightedDirector(backends ...backend) func(*http.Request) {
	var (
		cdf []float64
		cum float64
	)

	// build out a cdf for random, weighted selection
	for _, backend := range backends {
		log.Println(backend)
		cum += float64(backend.weight)
		cdf = append(cdf, cum)
	}

	return func(req *http.Request) {
		http.DefaultTransport.(*http.Transport).CloseIdleConnections()
		req.Close = true // reconnet each time to backend
		log.Println(req.Method, req.RequestURI, req.RemoteAddr)
		if len(backends) == 0 {
			req.URL.Path = "/down"
			req.URL.Scheme = "http"
			req.URL.Host = req.Host
			log.Println("unroutable request")
			return
		}

		var backend backend
		if len(backends) == 1 {
			backend = backends[0]
		} else {
			r := cdf[len(cdf)-1] * rand.Float64()
			i := sort.SearchFloat64s(cdf, r)
			backend = backends[i]
		}

		logger.Println("selected", backend)

		req.URL.Scheme = backend.url.Scheme
		req.URL.Host = backend.url.Host
		req.URL.Path = backend.url.Path

		req.Header.Set("Color", backend.color)
	}
}
