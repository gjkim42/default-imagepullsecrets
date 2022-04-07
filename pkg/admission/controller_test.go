package admission

import (
	"strings"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/klogr"
)

func TestAdmit(t *testing.T) {
	testCases := []struct {
		desc    string
		review  v1.AdmissionReview
		allowed bool
	}{
		{
			desc: "should accept a valid pod",
			review: v1.AdmissionReview{
				Request: &v1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "",
						Version:  "v1",
						Resource: "pods",
					},
				},
			},
			allowed: true,
		},
		{
			desc: "should not accept a resource other than pods",
			review: v1.AdmissionReview{
				Request: &v1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "",
						Version:  "v1",
						Resource: "configmaps",
					},
				},
			},
			allowed: false,
		},
	}

	for _, tc := range testCases {
		t.Run("v1 "+tc.desc, func(t *testing.T) {
			c := NewController(klogr.New(), nil)
			res := c.admit(tc.review)
			if res.Allowed != tc.allowed {
				t.Errorf("expected %v, got %v", tc.allowed, res.Allowed)
			}
		})
		t.Run("v1beta1 "+tc.desc, func(t *testing.T) {
			c := NewController(klogr.New(), nil)
			res := c.admitV1beta1(v1beta1.AdmissionReview{
				Request: convertAdmissionRequestToV1beta1(tc.review.Request),
			})
			if res.Allowed != tc.allowed {
				t.Errorf("expected %v, got %v", tc.allowed, res.Allowed)
			}
		})
	}
}

func TestConvertToPatch(t *testing.T) {
	testCases := []struct {
		desc    string
		secrets []string
		patch   string
	}{
		{
			desc: "should add a new secret",
			secrets: []string{
				"foo",
			},
			patch: `[
				{
					"op":"add",
					"path":"/spec/imagePullSecrets",
					"value":[{"name":"foo"}]
				}
			]`,
		},
		{
			desc: "should add a multiple new secrets",
			secrets: []string{
				"foo",
				"bar",
			},
			patch: `[
				{
					"op":"add",
					"path":"/spec/imagePullSecrets",
					"value":[
						{"name":"foo"},
						{"name":"bar"}
					]
				}
			]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			expected, got := removeSpaces(tc.patch), removeSpaces(convertToPatch(tc.secrets))
			if diff := cmp.Diff(expected, got); len(diff) != 0 {
				t.Errorf("unexpected patch (-want +got):\n%s", diff)
			}
		})
	}
}

func removeSpaces(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if !unicode.IsSpace(ch) {
			b.WriteRune(ch)
		}
	}
	return b.String()
}
