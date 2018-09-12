package main


import (
    "log"
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/gorilla/mux"
    "github.com/namsral/flag"
)


var (
    promPort string
	matches = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logol_match",
			Help: "Number of matches.",
		},
		[]string{"modvar"},
	)
)

func getMatch(w http.ResponseWriter, request *http.Request) {
    vars := mux.Vars(request)
    modvar := vars["modvar"]
    matches.With(prometheus.Labels{"modvar":modvar}).Inc()
}

func main() {
    flag.StringVar(&promPort, "listen", ":8080", "interface/port to listen on, default :8080")
    flag.Parse()
    prometheus.MustRegister(matches)
    r := mux.NewRouter()
    r.HandleFunc("/metric/{modvar}", getMatch)
    r.Handle("/metrics", promhttp.Handler())
    //http.Handle("/metrics", promhttp.Handler())
    http.Handle("/", r)

	log.Fatal(http.ListenAndServe(promPort, nil))
}
