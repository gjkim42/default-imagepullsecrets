package admission

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Controller struct {
	log logr.Logger

	secrets []string
	patch   []byte
}

func NewController(log logr.Logger, secrets []string) *Controller {
	return &Controller{
		log:     log,
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
		c.log.Error(nil, "Got wrong contentType", "got", contentType, "expect", "application/json")
		return
	}

	c.log.V(2).Info("Handling", "request", string(body))

	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to deserialize request object: %v", err)
		c.log.Error(err, msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case v1beta1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1beta1.AdmissionReview)
		if !ok {
			c.log.Error(nil, "Wrong AdmissionReview type", "expect", "v1beta1.AdmissionReview", "got", fmt.Sprintf("%T", obj))
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
			c.log.Error(nil, "Wrong AdmissionReview type", "expect", "v1.AdmissionReview", "got", fmt.Sprintf("%T", obj))
			return
		}
		responseAdmissionReview := &v1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = c.admit(*requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		c.log.Error(nil, msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	c.log.V(2).Info("Sending", "response", responseObj)
	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		msg := fmt.Sprintf("Failed to serialize response object: %v", err)
		c.log.Error(err, msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		c.log.Error(err, "Failed to write response")
	}
}

func (c *Controller) admit(review v1.AdmissionReview) *v1.AdmissionResponse {
	c.log.Info("Admitting pods")
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if review.Request.Resource != podResource {
		err := fmt.Errorf("expected resource to be %s", podResource)
		c.log.Error(err, "Failed to admit")
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
