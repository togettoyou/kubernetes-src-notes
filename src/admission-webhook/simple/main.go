package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"simple/pkg/admit"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	http.HandleFunc("/validating", func(w http.ResponseWriter, r *http.Request) {
		admit.Serve(w, r, func(review admit.AdmissionReview) *v1.AdmissionResponse {
			req := review.Request
			switch req.Kind.Kind {
			case "Pod":
				var pod corev1.Pod
				if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
					msg := fmt.Sprintf("could not decode pod: %v", err)
					klog.Error(msg)
					http.Error(w, msg, http.StatusInternalServerError)
					return &v1.AdmissionResponse{
						Allowed: false,
						Result: &metav1.Status{
							Message: msg,
						},
					}
				}
				if _, ok := pod.Labels["app"]; !ok {
					return &v1.AdmissionResponse{
						Allowed: false,
						Result: &metav1.Status{
							Reason: "Pod does not have an app label",
						},
					}
				}
			}
			return &v1.AdmissionResponse{
				Allowed: true,
			}
		})
	})
	http.HandleFunc("/mutating", func(w http.ResponseWriter, r *http.Request) {
		admit.Serve(w, r, func(review admit.AdmissionReview) *v1.AdmissionResponse {
			req := review.Request
			if req.Kind.Kind != "Pod" {
				return &v1.AdmissionResponse{
					Allowed: true,
				}
			}

			// 为pod添加simple-app标签
			patchBytes, patchType, err := admit.PatchTypeJSONPatch(admit.PatchOperation{
				Op:    "add",
				Path:  "/metadata/labels/simple-app",
				Value: "mutating",
			})
			if err != nil {
				msg := fmt.Sprintf("marshall jsonpatch: %v", err)
				klog.Error(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return &v1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Message: msg,
					},
				}
			}
			return &v1.AdmissionResponse{
				Allowed:   true,
				Patch:     patchBytes,
				PatchType: patchType,
			}
		})
	})

	panic(http.ListenAndServeTLS(fmt.Sprintf(":%d", port), crt, key, nil))
}
