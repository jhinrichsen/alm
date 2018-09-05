// Error codes:
// 1: general error
// 2: bad commandline/ usage

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	yaml "gopkg.in/yaml.v2"
)

const (
	SignInUrl  = "/api/authentication/sign-in"
	SignOutUrl = "/api/authentication/sign-out"
)

type AlmInstance struct {
	Protocol string
	Server   string
	Port     int
	Context  string

	Username string
	Password string

	Domain  string
	Project string

	FromStatus, IntoStatus string

	Client http.Client
}

type Defect struct {
	Id      int    `json:"id"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}

func main() {
	// commandline parameter
	cp := AlmInstance{}
	flag.StringVar(&cp.Server, "server", "", "IP address of ALM server instance")
	flag.StringVar(&cp.Protocol, "protocol", "https", "ALM server protocol")
	flag.IntVar(&cp.Port, "port", 0, "ALM server protocol")
	flag.StringVar(&cp.Context, "context", "/qcbin", "ALM server webroot")

	flag.StringVar(&cp.Username, "username", "", "ALM user name")
	flag.StringVar(&cp.Password, "password", "", "ALM user name")

	flag.StringVar(&cp.FromStatus, "fromstatus", "", "only tickets in this status will be changed")
	flag.StringVar(&cp.IntoStatus, "intostatus", "", "tickets will be changed to this status")

	config := flag.String("config", DefaultConfig(), "configuration file")
	insecure := flag.Bool("insecure", false, "disable TLS certificates (not suggested)")
	prefix := flag.String("prefix", "ALM_", "prefix for environment variables")
	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s DEFECTID*\n", os.Args[0])
		flag.PrintDefaults()
	}

	// Check if defect IDs supplied
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	// configuration file
	cf, err := ReadCfg(*config)
	log.Printf("configuration file: %+v\n", cf)
	die(err)

	// environment variables
	var ev AlmInstance
	ReadEnv(*prefix, &ev)
	log.Printf("environment variables: %+v\n", ev)

	a, err := merge(cp, ev, cf)
	die(err)
	a.Client = *Client(*insecure)
	log.Printf("using ALM instance %+v\n", a)

	if err := a.SignIn(); err != nil {
		die(err)
	}
	defer func() {
		a.SignOut()
	}()
	for _, s := range flag.Args() {
		id, err := strconv.Atoi(s)
		die(err)
		d, err := a.GetDefect(id)
		log.Printf("existing defect: %+v\n", d)
		// Optionally filter on existing status
		if a.FromStatus != "" && d.Status != a.FromStatus {
			log.Printf("want status %q but got %q, skipping %d\n", a.FromStatus, d.Status, id)
		}
		log.Printf("updating %d to %q\n", id, a.IntoStatus)
		d2 := Defect{
			Id:     id,
			Status: a.IntoStatus,
		}
		d3, err := a.PutDefect(d2)
		log.Printf("updated defect to %+v\n", d3)
	}
}

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

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
func DefaultConfig() string {
	u, err := user.Current()
	die(err)
	return filepath.Join(u.HomeDir, ".alm.yaml")
}

func Client(insecure bool) *http.Client {
	var client http.Client
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		log.Fatal(err)
	}
	client.Jar = jar
	return &client
}

// ReadCfg reads the config file
func ReadCfg(filename string) (AlmInstance, error) {
	buf, err := ioutil.ReadFile(filename)
	die(err)
	var a AlmInstance
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

func (a *AlmInstance) Url(uri string) string {
	return fmt.Sprintf("%s://%s:%d%s%s", a.Protocol, a.Server, a.Port, a.Context, uri)
}

// Login authenticates against ALM
// https://admhelp.microfocus.com/alm/en/12.55/api_refs/REST/Content/REST_API/sign_in.htm
func (a *AlmInstance) SignIn() error {
	log.Printf("signing in %s\n", a.Username)
	req, err := http.NewRequest("GET", a.Url(SignInUrl), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.Username, a.Password)
	res, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	log.Printf("response: %+v\n", res)
	log.Printf("cookies:\n")
	u, err := url.Parse(a.Url(""))
	if err != nil {
		return err
	}
	for _, cookie := range a.Client.Jar.Cookies(u) {
		log.Printf("\t%+v\n", cookie)
	}
	return nil
}

func (a *AlmInstance) SignOut() error {
	log.Printf("signing out %s\n", a.Username)
	res, err := a.Client.Get(a.Url(SignOutUrl))
	if err != nil {
		return err
	}
	log.Printf("response: %+v\n", res)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("expected http status code 200 but got %d\n", res.StatusCode)

	}
	return nil
}

func (a *AlmInstance) GetDefect(defect int) (*Defect, error) {
	u := a.Url(DefectsUri(a.Domain, a.Project, defect))
	log.Printf("GET %s\n", u)
	res, err := a.Client.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Printf("defect: %+v\n", res)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("warning: http status code %d\n", res.StatusCode)
	}
	return ParseDefect(buf)
}

func ParseDefect(buf []byte) (*Defect, error) {
	var d Defect
	if err := json.Unmarshal(buf, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (a *AlmInstance) PutDefect(d Defect) (*Defect, error) {
	u := a.Url(DefectsUri(a.Domain, a.Project, d.Id))
	enc, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", u, bytes.NewBuffer(enc))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	log.Printf("request: %+v\n", req)
	res, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("warning: http status code %d\n", res.StatusCode)
	}
	log.Printf("response: %+v\n", res)
	defer res.Body.Close()
	dec, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return ParseDefect(dec)
}

// DefectsUri returns the uri for given defect
func DefectsUri(domain, project string, defect int) string {
	return fmt.Sprintf("/api/domains/%s/projects/%s/defects/%d", domain, project, defect)
}
