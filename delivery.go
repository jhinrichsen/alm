package main

import (
	"errors"
	"io"
	"io/ioutil"
	"reflect"

	yaml "gopkg.in/yaml.v2"
)

// YAML structure:
// tmt:
//     domain: {{ your TMT domain }}
//     project: {{ your TMT project }}
//     defects:
// 	            - {{ TMT defect ID #1 }}
// 	            - {{ TMT defect ID #2 }}

type Delivery struct {
	Tmt Tmt
}

type Tmt struct {
	Domain  string
	Project string
	Defects []string
}

// A couple of parsing errors
var (
	MissingTmt     = errors.New("Missing required element tmt:")
	EmptyTmt       = errors.New("Empty tmt:, missing required elements")
	MissingDomain  = errors.New("Missing required element tmt: domain:")
	MissingProject = errors.New("Missing required element tmt: project:")
)

func (a Delivery) validate() error {
	if reflect.DeepEqual(a, Delivery{}) {
		return MissingTmt
	}
	if reflect.DeepEqual(a.Tmt, Tmt{}) {
		return EmptyTmt
	}
	if a.Tmt.Domain == "" {
		return MissingDomain
	}
	if a.Tmt.Project == "" {
		return MissingProject
	}
	// an empty list of defects is fine (just a feature releaes w/o
	// bugfixes)
	return nil
}

// parse expects a standard YAML structure that contains fixed issues.
func parse(in io.Reader, a AlmInstance) (Tmt, error) {
	var d Delivery
	buf, err := ioutil.ReadAll(in)
	if err != nil {
		return d.Tmt, err
	}
	if err := yaml.Unmarshal(buf, &d); err != nil {
		return d.Tmt, err
	}
	return d.Tmt, nil
}
