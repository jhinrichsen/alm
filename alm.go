package alm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	// SignInURL according to ALM REST documentation
	SignInURL = "/api/authentication/sign-in"
	// SignOutURL according to ALM REST documentation
	SignOutURL = "/api/authentication/sign-out"
)

// Instance represents one remote service instance.
type Instance struct {
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

// Defect represents a subset of an ALM defect
type Defect struct {
	ID      int    `json:"id"`
	Subject int    `json:"subject"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}

// DefectsResponse represents a REST api call request.
type DefectsResponse struct {
	Defects []Defect `json:"results"`
}

// DomainsResponse contains a list of all available domains in ALM.
type DomainsResponse struct {
	Domains []Domain `json:"results"`
}

// Domain represents ALM domain model
type Domain struct {
	Name string `json:"name"`
}

// Release represents ALM domain model
type Release struct {
	ID      int    `json:"id"`
	Subject int    `json:"subject"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}

// URL returns a relative, constant REST API URL into a specific instance one
func (a *Instance) URL(uri string) string {
	return fmt.Sprintf("%s://%s:%d%s%s", a.Protocol, a.Server, a.Port, a.Context, uri)
}

// SignIn authenticates against ALM.
// https://admhelp.microfocus.com/alm/en/12.55/api_refs/REST/Content/REST_API/sign_in.htm
func (a *Instance) SignIn() error {
	log.Printf("signing in %s\n", a.Username)
	req, err := http.NewRequest("GET", a.URL(SignInURL), nil)
	if err != nil {
		return err
	}
	log.Printf("GET %s\n", req.URL)
	req.SetBasicAuth(a.Username, a.Password)
	res, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusOK {
		log.Println(res.Status)
	} else {
		return fmt.Errorf("error signing in: %s", res.Status)
	}

	/*
		u, err := url.Parse(a.Url(""))
		if err != nil {
			return err
		}
			for _, cookie := range a.Client.Jar.Cookies(u) {
				log.Printf("\t%+v\n", cookie)
			}
	*/
	return nil
}

// SignOut logs off from ALM
func (a *Instance) SignOut() error {
	log.Printf("signing out %s\n", a.Username)
	u := a.URL(SignOutURL)
	log.Printf("GET %s\n", u)
	res, err := a.Client.Get(u)
	if err != nil {
		return err
	}
	log.Println(res.Status)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("expected http status code 200 but got %d\n",
			res.StatusCode)

	}
	return nil
}

// GetDefect issues a remote REST call for given defect
func (a *Instance) GetDefect(defect int) (*Defect, error) {
	u := a.URL(DefectsURI(a.Domain, a.Project, defect))
	log.Printf("GET %s\n", u)
	res, err := a.Client.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("warning: http status code %d\n", res.StatusCode)
	}
	return ParseDefect(buf)
}

// ParseDefect tries to convert bytes in JSON format into a Defect.
func ParseDefect(buf []byte) (*Defect, error) {
	var d Defect
	if err := json.Unmarshal(buf, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// PutDefect issues a REST call to store a Defect.
func (a *Instance) PutDefect(d Defect) (*Defect, error) {
	u := a.URL(DefectsURI(a.Domain, a.Project, d.ID))
	enc, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", u, bytes.NewBuffer(enc))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	log.Printf("GET %s\n", req.URL)
	res, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("warning: http status code %d\n", res.StatusCode)
	}
	log.Println(res.Status)
	defer res.Body.Close()
	dec, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return ParseDefect(dec)
}

// DefectsURI returns the relative, constant REST API path for given defect.
func DefectsURI(domain, project string, defect int) string {
	return fmt.Sprintf("/api/domains/%s/projects/%s/defects/%d", domain, project, defect)
}

// GetRelease issues a REST call to retrieve a Release.
func (a *Instance) GetRelease(ID string) (*Release, error) {
	u := a.URL(ReleasesURI(a.Domain, a.Project, ID))
	log.Printf("GET %s\n", u)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Printf("release: %+v\n", res)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("warning: http status code %d\n", res.StatusCode)
	}
	log.Printf("body: %+v\n", string(buf))
	return ParseRelease(buf)
}

// ReleasesURI returns relative, constant REST API for given parameter set.
func ReleasesURI(domain, project string, release string) string {
	// return fmt.Sprintf("/api/domains/%s/projects/%s/releases/%s",
	// domain, project, release)

	return fmt.Sprintf("/api/domains/%s/projects/%s/releases",
		// return fmt.Sprintf("/api/domains/%s/projects/%s/release-folders",
		domain, project)
}

// ParseRelease tries to convert bytes in JSON format into a Release.
func ParseRelease(buf []byte) (*Release, error) {
	var r Release
	if err := json.Unmarshal(buf, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Domains issues a REST call to retrieve a list of all available ALM domains.
func (a *Instance) Domains() ([]Domain, error) {
	u := a.URL(domainsURI())
	log.Printf("GET %s\n", u)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Printf("domains: %+v\n", res)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("warning: http status code %d\n", res.StatusCode)
	}
	log.Printf("body: %+v\n", string(buf))
	return parseDomains(buf)
}

func domainsURI() string {
	// domains and projects: "/api/domains?include-projects-info=y"
	return "/api/domains"
}

func parseDomains(buf []byte) ([]Domain, error) {
	var dr DomainsResponse
	if err := json.Unmarshal(buf, &dr); err != nil {
		return nil, err
	}
	return dr.Domains, nil
}

// Defects issues a REST call to retrieve a list of defects for given parameter
// set.
func (a *Instance) Defects(domain, project string) ([]Defect, error) {
	u := a.URL(defectsURI(domain, project))
	log.Printf("GET %s\n", u)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Printf("domains: %+v\n", res)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("warning: http status code %d\n", res.StatusCode)
	}
	log.Printf("body: %+v\n", string(buf))
	return parseDefects(buf)
}

func defectsURI(domain, project string) string {
	// api/domains/{domain}/projects/{project}/defects
	return fmt.Sprintf("/api/domains/%s/projects/%s/defects",
		domain, project)
}

func parseDefects(buf []byte) ([]Defect, error) {
	var dr DefectsResponse
	if err := json.Unmarshal(buf, &dr); err != nil {
		return nil, err
	}
	return dr.Defects, nil
}

// UpdateDefects issues a REST call to update status of defects.
func (a *Instance) UpdateDefects(defects []string) error {
	for _, s := range defects {
		fmt.Printf("parsing defect %q\n", s)
		id, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		d, err := a.GetDefect(id)
		if err != nil {
			return err
		}
		log.Printf("existing defect: %+v\n", d)
		// Optionally filter on existing status
		if a.FromStatus != "" && !strings.HasPrefix(d.Status, a.FromStatus) {
			log.Printf("want status to start with %q but got %q"+
				", skipping %d\n",
				a.FromStatus, d.Status, id)
			continue
		}
		log.Printf("updating %d to %q\n", id, a.IntoStatus)
		d2 := Defect{
			ID:     id,
			Status: a.IntoStatus,
		}
		d3, err := a.PutDefect(d2)
		if err != nil {
			return err
		}
		log.Printf("updated defect to %+v\n", d3)
	}
	return nil
}

// NewReleases logs existing releases to stdout.
func (a *Instance) NewReleases(releaseIDs []string) error {
	for _, releaseID := range releaseIDs {
		release, err := a.GetRelease(releaseID)
		if err != nil {
			return err
		}
		log.Printf("release %v: %v\n", releaseID, release)
	}
	r, err := a.GetRelease("")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("release %s: %+v\n", "<>", r)
	return nil
}

// Client returns a configured http client usable for ALM access.
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
