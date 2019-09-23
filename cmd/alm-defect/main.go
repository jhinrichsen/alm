package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gitlab.com/jhinrichsen/alm"
)

// Version is populated from git during link step
var Version = "undefined"

// Commit is populated from git during link step
var Commit = "undefined"

// Return codes
const (
	NoError         = iota // 0 indicates no error
	ErrorUnspecific        // panic e.a.
	ErrorUsage
	ErrorParsing
)

// Error codes:
// 1: general error
// 2: bad commandline/ usage
func main() {
	var validActions = []string{
		"defects",
		"delivery",
		"domains",
		"release",
		"update",
	}

	isValidAction := func(action string) bool {
		for _, s := range validActions {
			if action == s {
				return true
			}
		}
		return false
	}

	// commandline parameter
	cp := alm.Instance{}
	flag.StringVar(&cp.Server, "server", "",
		"IP address of ALM server instance")
	flag.StringVar(&cp.Protocol, "protocol", "https", "ALM server protocol")
	flag.IntVar(&cp.Port, "port", 0, "ALM server protocol")
	flag.StringVar(&cp.Context, "context", "/qcbin", "ALM server webroot")

	flag.StringVar(&cp.Username, "username", "", "ALM user name")
	flag.StringVar(&cp.Password, "password", "", "ALM user name")

	flag.StringVar(&cp.FromStatus, "fromstatus", "",
		"only tickets in this status will be changed")
	flag.StringVar(&cp.IntoStatus, "intostatus", "",
		"tickets will be changed to this status")

	dc, err := alm.DefaultConfig()
	if err != nil {
		log.Fatal(err)
	}
	config := flag.String("config", dc, "configuration file")
	insecure := flag.Bool("insecure", false,
		"disable TLS certificates (not suggested)")
	prefix := flag.String("prefix", "ALM_",
		"prefix for environment variables")

	flag.Parse()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s defect | delivery | domains | release ID*\n",
			os.Args[0])
		flag.PrintDefaults()
	}

	// Check if command supplied
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(ErrorUsage)
	}
	action := flag.Args()[0]
	if !isValidAction(action) {
		fmt.Fprintf(os.Stderr, "Not a valid action: %q\n", action)
		flag.Usage()
		os.Exit(ErrorUsage)
	}

	// configuration file
	cf, err := alm.ReadCfg(*config)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("configuration file: %+v\n", cf)

	// environment variables
	var ev alm.Instance
	alm.ReadEnv(*prefix, &ev)
	log.Printf("environment variables: %+v\n", ev)

	a, err := alm.Merge(cp, ev, cf)
	if err != nil {
		log.Fatal(err)
	}
	a.Client = *alm.Client(*insecure)
	log.Printf("using ALM instance %+v\n", a)

	if err := a.SignIn(); err != nil {
		log.Fatal(err)
	}
	defer func() {
		a.SignOut()
	}()

	log.Printf("running action %q\n", action)
	switch action {

	case "defects":
		ds, err := a.Defects(a.Domain, a.Project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error retrieving defects: %s\n",
				err)
		}
		for _, d := range ds {
			fmt.Printf("%+v\n", d)
		}

	case "delivery":
		t, err := alm.Parse(os.Stdin, *a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing stdin: %s\n", err)
			os.Exit(ErrorParsing)
		}
		a.Domain = t.Domain
		a.Project = t.Project
		if err := a.UpdateDefects(t.Defects); err != nil {
			fmt.Fprintf(os.Stderr, "error updating defects: %s\n",
				err)
		}

	case "domains":
		a.Domains()

	case "update":
		a.UpdateDefects(flag.Args()[1:])

	default: // releases
		a.NewReleases(flag.Args()[1:])
	}
}
