package webitel

import (
	"reflect"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestTrailer(t *testing.T) {
	trailer := Trailer(metadata.New(map[string]string{"a": "b"}))
	if !reflect.DeepEqual("b", trailer.Get("a")) {
		t.Errorf("expect %v, got %v", "b", trailer.Get("a"))
	}
	if !reflect.DeepEqual("", trailer.Get("notfound")) {
		t.Errorf("expect %v, got %v", "", trailer.Get("notfound"))
	}
}
