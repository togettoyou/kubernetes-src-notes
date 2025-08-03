package apis

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Hello struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec HelloSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

func (h *Hello) DeepCopyObject() runtime.Object {
	nh := *h
	return &nh
}

var _ runtime.Object = &Hello{}

type HelloSpec struct {
	Msg string `json:"msg,omitempty" protobuf:"bytes,10,opt,name=msg"`
}

var (
	__TODOHello []byte
	_TODOHello  = &Hello{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Hello",
			APIVersion: "simple.aa.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-hello",
			Namespace: "default",
		},
		Spec: HelloSpec{
			Msg: "hello AA",
		},
	}

	__TODOHelloTable []byte
	_TODOHelloTable  = &metav1.Table{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Table",
			APIVersion: "meta.k8s.io/v1",
		},
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Msg", Type: "string", Format: "msg"},
		},
		Rows: []metav1.TableRow{
			{
				Cells:  []interface{}{_TODOHello.Name, _TODOHello.Spec.Msg},
				Object: runtime.RawExtension{Object: _TODOHello},
			},
		},
	}
)

func TODOHello() []byte {
	if __TODOHello == nil {
		__TODOHello = marshal(_TODOHello)
	}
	return __TODOHello
}

func TODOHelloTable() []byte {
	if __TODOHelloTable == nil {
		__TODOHelloTable = marshal(_TODOHelloTable)
	}
	return __TODOHelloTable
}
