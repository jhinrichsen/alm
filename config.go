package main

import (
	"io/ioutil"

	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// merge multiple partly filled structs into one, precedence from the left to the right
func merge(as ...AlmInstance) (*AlmInstance, error) {
	var merged AlmInstance
	for _, a := range as {
		if err := mergo.Merge(&merged, a); err != nil {
			return nil, err
		}
	}
	return &merged, nil
}

// DefaultConfig returns the fully qualified filename of the config file, i.e. `${HOME}/.alm.yaml`
func DefaultConfig() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".alm.yaml"), nil
}

// ReadCfg reads the config file
func ReadCfg(filename string) (AlmInstance, error) {
	var a AlmInstance
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return a, err
	}
	if err := yaml.Unmarshal(buf, &a); err != nil {
		log.Fatalf("error parsing %s: %v\n", filename, err)
	}
	return a, nil
}

func ReadEnv(prefix string, a *AlmInstance) {
	strct := reflect.ValueOf(a).Elem()
	for i := 0; i < strct.NumField(); i++ {
		name := strct.Type().Field(i).Name
		key := prefix + strings.ToUpper(name)
		value := os.Getenv(key)
		log.Printf("%s=%s\n", key, value)

		if value == "" {
			continue
		}
		// write back value to struct
		log.Printf("picking field %s\n", name)
		f := strct.FieldByName(name)
		log.Printf("picked %+v\n", f)
		if !f.IsValid() {
			log.Fatalf("internal error: not a valid field: %+v\n", f)
		}
		if !f.CanSet() {
			log.Fatalf("internal error: cannot set %+v\n", f)
		}
		switch f.Kind() {
		case reflect.String:
			f.SetString(value)
		case reflect.Int:
			x, err := strconv.Atoi(value)
			if err != nil {
				log.Fatalf("cannot convert %q to int: %v", value, err)
			}
			f.SetInt(int64(x))
		default:
			panic("unsupported type")
		}
	}
}
