package alm

import (
	"io/ioutil"
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestDelivery(t *testing.T) {
	filename := "testdata/delivery.yml"
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	want := Delivery{
		Tmt{
			Domain:  "TMT_DOMAIN",
			Project: "TMT_PROJECT",
			Defects: []string{
				"4711",
				"4712",
				"4713",
			},
		},
	}
	var got Delivery
	if err := yaml.Unmarshal(buf, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %+v but got %+v", want, got)
	}
}

func TestMissingTmt(t *testing.T) {
	want := ErrMissingTmt
	got := (Delivery{}).validate()
	if want != got {
		t.Fatalf("want %+v but got %+v", want, got)
	}
}

func TestMissingDomain(t *testing.T) {
	want := ErrMissingDomain
	got := (Delivery{Tmt{Project: "project1"}}).validate()
	if want != got {
		t.Fatalf("want %+v but got %+v", want, got)
	}
}

func TestNoDefects(t *testing.T) {
	want := error(nil)
	got := (Delivery{Tmt{Domain: "domain1", Project: "project1"}}).validate()
	if want != got {
		t.Fatalf("want %+v but got %+v", want, got)
	}

}
