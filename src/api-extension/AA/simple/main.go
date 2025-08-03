package main

import (
	"flag"
	"fmt"
	"net/http"
	"simple/pkg/apis"
	"strings"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
)

var (
	crt  string
	key  string
	port int
)

func init() {
	flag.StringVar(&crt, "tls-crt-file", "tls.crt", "CRT 证书文件")
	flag.StringVar(&key, "tls-key-file", "tls.key", "KEY 私钥文件")
	flag.IntVar(&port, "port", 443, "Webhook Server 端口")
}

func main() {
	flag.Parse()

	r := mux.NewRouter()

	// APIService: v1beta1.simple.aa.io
	// GV: simple.aa.io/v1beta1

	// API Discovery
	r.HandleFunc("/apis", func(w http.ResponseWriter, r *http.Request) {
		klog.Info("API Discovery", "/apis")

		// 判断 header 是否有 Accept: application/json;as=APIGroupDiscoveryList;v=v2beta1;g=apidiscovery.k8s.io
		// 来区分是请求 APIGroupDiscoveryList 还是 APIGroupList 对象
		var as, v, g string
		accept := r.Header.Get("Accept")
		if accept != "" {
			for _, data := range strings.Split(accept, ";") {
				if values := strings.Split(data, "="); len(values) == 2 {
					switch values[0] {
					case "as":
						as = values[1]
					case "v":
						v = values[1]
					case "g":
						g = values[1]
					}
				}
			}
		}
		_ = v
		if as == "APIGroupDiscoveryList" && g == "apidiscovery.k8s.io" {
			w.Header().Set("Content-Type", "application/json;as=APIGroupDiscoveryList;v=v2beta1;g=apidiscovery.k8s.io")
			w.Write(apis.APIGroupDiscoveryList())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(apis.APIGroupList())
	})
	r.HandleFunc("/apis/simple.aa.io", func(w http.ResponseWriter, r *http.Request) {
		klog.Info("API Discovery", "/apis/simple.aa.io")
		w.Header().Set("Content-Type", "application/json")
		w.Write(apis.APIGroup())
	})
	r.HandleFunc("/apis/simple.aa.io/v1beta1", func(w http.ResponseWriter, r *http.Request) {
		klog.Info("API Discovery", "/apis/simple.aa.io/v1beta1")
		w.Header().Set("Content-Type", "application/json")
		w.Write(apis.APIResourceList())
	})

	// CR CRUD Handle
	// K: Hello
	hellos := r.PathPrefix("/apis/simple.aa.io/v1beta1").Subrouter()

	handle := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "application/json") && strings.Contains(accept, "as=Table") {
			w.Write(apis.TODOHelloTable())
			return
		}
		w.Write(apis.TODOHello())
	}

	// list -A
	hellos.HandleFunc("/hellos", func(w http.ResponseWriter, r *http.Request) {
		klog.Info(r.Method, " /hellos")

		handle(w, r)
	}).Methods("GET")

	// list by namespaces
	hellos.HandleFunc("/namespaces/{ns}/hellos", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		klog.Info(r.Method, fmt.Sprintf(" /namespaces/%s/hellos", vars["ns"]))

		handle(w, r)
	}).Methods("GET")

	// get by name
	hellos.HandleFunc("/namespaces/{ns}/hellos/{name}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		klog.Info(r.Method, fmt.Sprintf(" /namespaces/%s/hellos/%s", vars["ns"], vars["name"]))

		handle(w, r)
	}).Methods("GET")

	panic(http.ListenAndServeTLS(fmt.Sprintf(":%d", port), crt, key, r))
}
