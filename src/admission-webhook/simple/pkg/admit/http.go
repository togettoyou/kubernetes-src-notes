package admit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
)

type handler func(AdmissionReview) *v1.AdmissionResponse

// AdmissionReview is used to decode both v1 and v1beta1 AdmissionReview types.
type AdmissionReview struct {
	v1.AdmissionReview
}

func Serve(w http.ResponseWriter, r *http.Request, h handler) {
	// 1. Get body
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		msg := "empty body"
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// 2. Validate content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		msg := fmt.Sprintf("contentType=%s, expect application/json", contentType)
		klog.Error(msg)
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	// 3. Parse body into a Kubernetes resource object
	reqAdmissionReview := AdmissionReview{}
	_, gvk, err := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer().Decode(body, nil, &reqAdmissionReview)
	if err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// 4. Judge which version is available, v1beta1 and v1
	var responseObj runtime.Object
	switch *gvk {
	case v1beta1.SchemeGroupVersion.WithKind("AdmissionReview"), v1.SchemeGroupVersion.WithKind("AdmissionReview"):
		respAdmissionReview := &v1.AdmissionReview{}
		respAdmissionReview.SetGroupVersionKind(*gvk)
		respAdmissionReview.Response = h(reqAdmissionReview)
		respAdmissionReview.Response.UID = reqAdmissionReview.Request.UID
		responseObj = respAdmissionReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// 5. Writes responseObj to w
	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}
