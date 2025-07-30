package main

import (
	"log"
	"net/http"
	"simple/pkg/extender"
)

func main() {
	server := extender.NewServer(extender.NewSimpleHandler())

	http.HandleFunc("/filter", server.Filter())
	http.HandleFunc("/prioritize", server.Prioritize())
	http.HandleFunc("/preempt", server.ProcessPreemption())
	http.HandleFunc("/bind", server.Bind())

	log.Println("Scheduler extender listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
