package apis

import (
	"encoding/json"

	apidiscoveryv2beta1 "k8s.io/api/apidiscovery/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	__APIGroup []byte
	_APIGroup  = &metav1.APIGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIGroup",
			APIVersion: "v1",
		},
		Name: "simple.aa.io",
		Versions: []metav1.GroupVersionForDiscovery{
			{
				GroupVersion: "simple.aa.io/v1beta1",
				Version:      "v1beta1",
			},
		},
	}

	__APIGroupList []byte
	_APIGroupList  = &metav1.APIGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIGroupList",
			APIVersion: "v1",
		},
		Groups: []metav1.APIGroup{
			*_APIGroup,
		},
	}

	__APIResourceList []byte
	_APIResourceList  = &metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: "simple.aa.io/v1beta1",
		APIResources: []metav1.APIResource{
			{
				Name:         "hellos",
				SingularName: "hello",
				Namespaced:   true,
				Kind:         "Hello",
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"update",
					"patch",
				},
				ShortNames: []string{"hi"},
				Categories: []string{"all"},
			},
		},
	}

	__APIGroupDiscoveryList []byte
	_APIGroupDiscoveryList  = &apidiscoveryv2beta1.APIGroupDiscoveryList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIGroupDiscoveryList",
			APIVersion: "apidiscovery.k8s.io/v2beta1",
		},
		Items: []apidiscoveryv2beta1.APIGroupDiscovery{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "simple.aa.io",
				},
				Versions: []apidiscoveryv2beta1.APIVersionDiscovery{
					{
						Version: "v1beta1",
						Resources: []apidiscoveryv2beta1.APIResourceDiscovery{
							{
								Resource: "hellos",
								ResponseKind: &metav1.GroupVersionKind{
									Group:   "simple.aa.io",
									Version: "v1beta1",
									Kind:    "Hello",
								},
								Scope:            apidiscoveryv2beta1.ScopeNamespace,
								SingularResource: "hello",
								Verbs: []string{
									"create",
									"delete",
									"get",
									"list",
									"update",
									"patch",
								},
								ShortNames: []string{"hi"},
								Categories: []string{"all"},
							},
						},
					},
				},
			},
		},
	}
)

func APIGroup() []byte {
	if __APIGroup == nil {
		__APIGroup = marshal(_APIGroup)
	}
	return __APIGroup
}

func APIGroupList() []byte {
	if __APIGroupList == nil {
		__APIGroupList = marshal(_APIGroupList)
	}
	return __APIGroupList
}

func APIResourceList() []byte {
	if __APIResourceList == nil {
		__APIResourceList = marshal(_APIResourceList)
	}
	return __APIResourceList
}

func APIGroupDiscoveryList() []byte {
	if __APIGroupDiscoveryList == nil {
		__APIGroupDiscoveryList = marshal(_APIGroupDiscoveryList)
	}
	return __APIGroupDiscoveryList
}

func marshal(obj interface{}) []byte {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return nil
	}
	return bytes
}
