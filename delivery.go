package alm

import (
	"errors"
	"io"
	"io/ioutil"
	"reflect"

	yaml "gopkg.in/yaml.v2"
)

// YAML structure:
// tmt:
//     domain: {{ your ALM domain }}
//     project: {{ your ALM project }}
//     defects:
// 	            - {{ ALM defect ID #1 }}
// 	            - {{ ALM defect ID #2 }}

// Delivery represents an ALM domain model
type Delivery struct {
	Tmt Tmt
}

// Tmt is a root ALM domain model.
type Tmt struct {
	Domain  string
	Project string
	Defects []string
}

// A couple of parsing errors
var (
	// ErrMissingTmt indicates a missing root entry /tmt
	ErrMissingTmt = errors.New("missing required element tmt")
	// ErrEmptyTmt indicates root entry /tmt exists, but has no embedded
	// information.
	ErrEmptyTmt = errors.New("empty tmt:, missing required elements")
	// ErrMissingDomain indicates absence of tmt/domain
	ErrMissingDomain = errors.New("missing required element /tmt/domain")
	// ErrMissingProject indicates absence of tmt/project
	ErrMissingProject = errors.New("missing required element /tmt/project")
)

func (a Delivery) validate() error {
	if reflect.DeepEqual(a, Delivery{}) {
		return ErrMissingTmt
	}
	if reflect.DeepEqual(a.Tmt, Tmt{}) {
		return ErrEmptyTmt
	}
	if a.Tmt.Domain == "" {
		return ErrMissingDomain
	}
	if a.Tmt.Project == "" {
		return ErrMissingProject
	}
	// an empty list of defects is fine (just a feature releaes w/o
	// bugfixes)
	return nil
}

// Parse expects a standard YAML structure that contains fixed issues.
func Parse(in io.Reader, a Instance) (Tmt, error) {
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
