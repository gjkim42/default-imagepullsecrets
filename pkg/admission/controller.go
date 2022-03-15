package admission

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

type Controller struct {
	secrets []string
	patch   []byte
}

func NewController(secrets []string) *Controller {
	return &Controller{
		secrets: secrets,
		patch:   []byte(convertToPatch(secrets)),
	}
}

func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.ErrorS(nil, "Got wrong contentType", "got", contentType, "expect", "application/json")
		return
	}

	klog.V(2).InfoS("Handling", "request", string(body))

	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to deserialize request object: %v", err)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case v1beta1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1beta1.AdmissionReview)
		if !ok {
			klog.ErrorS(nil, "Wrong AdmissionReview type", "expect", "v1beta1.AdmissionReview", "got", fmt.Sprintf("%T", obj))
			return
		}
		responseAdmissionReview := &v1beta1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = c.admitV1beta1(*requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	case v1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1.AdmissionReview)
		if !ok {
			klog.ErrorS(nil, "Wrong AdmissionReview type", "expect", "v1.AdmissionReview", "got", fmt.Sprintf("%T", obj))
			return
		}
		responseAdmissionReview := &v1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = c.admit(*requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	klog.V(2).Info("Sending", "response", responseObj)
	respBytes, err := json.Marshal(responseObj)
	if err != nil {

		msg := fmt.Sprintf("Failed to serialize response object: %v", err)
		klog.Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		klog.ErrorS(err, "Failed to write response")
	}
}

func (c *Controller) admit(review v1.AdmissionReview) *v1.AdmissionResponse {
	klog.InfoS("Admitting pods")
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if review.Request.Resource != podResource {
		err := fmt.Errorf("Expected resource to be %s", podResource)
		klog.Error(err)
		return toV1AdmissionResponse(err)
	}

	pt := v1.PatchTypeJSONPatch
	return &v1.AdmissionResponse{
		Allowed:   true,
		Patch:     c.patch,
		PatchType: &pt,
	}
}

func (c *Controller) admitV1beta1(review v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	in := v1.AdmissionReview{Request: convertAdmissionRequestToV1(review.Request)}
	out := c.admit(in)
	return convertAdmissionResponseToV1beta1(out)
}

func convertToPatch(secrets []string) string {
	imagePullSecrets := make([]string, 0, len(secrets))
	for _, s := range secrets {
		imagePullSecrets = append(imagePullSecrets, fmt.Sprintf(`{ "name": "%s" }`, s))
	}
	return fmt.Sprintf(`[
					 {
						 "op": "add",
						 "path": "/spec/imagePullSecrets",
						 "value": [%s]
					 }
				 ]`, strings.Join(imagePullSecrets, ","))
}
