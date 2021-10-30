package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type filterFlags []string

func (f *filterFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (f *filterFlags) String() string {
	str := "(" + strings.Join(*f, ") OR (") + ")"
	str = strings.ReplaceAll(str, ",", " AND ")
	return str
}

func (f *filterFlags) toFilters() [][]string {
	filters := [][]string{}
	for _, filter := range *f {
		filters = append(filters, strings.Split(filter, ","))
	}
	return filters
}

// TODO: Show help
func doHelp() {
	log.Fatal("Unimplemented")
}

func checkAllConfig(config *ghpConfig, client *ghpClient) {
	valid, err := client.validToken()
	if err != nil {
		fmt.Printf("There was a problem with the current stored token: %v", err)
	}
	if !valid {
		fmt.Println("ghp is not configured yet, run 'ghp auth' and 'ghp config'")
		os.Exit(1)
	}
	if config.DefaultProjectID == 0 {
		fmt.Println("You don't have configured ghp yet, run 'ghp config'")
		os.Exit(1)
	}
}

func doList(state ghpConfig, cache *appCache, f filterFlags) {
	fmt.Printf("Requesting full project %v, this can take some time\n", state.DefaultProject)
	p := new(ProjectProxy)
	err := p.init(state, cache, state.DefaultProjectID)
	if err != nil {
		fmt.Printf("Error creating client %v", err)
	}
	err = p.pullColums(state.DefaultProjectID)
	if err != nil {
		fmt.Printf("Error reading project %v", err)
	}
	// log.Printf("prj %+v", p)
	if len(f) != 0 {
		fmt.Printf("Appliying filters: %v\n", f.String())
	}
	//p.listProject(f.toFilters())
	fancyList(p, f.toFilters())
	fmt.Printf("\ncache performance:\nHits: %v\nMiss:%v\n", cache.cacheHits, cache.cacheMiss)
}

func main() {
	state, err := stateLoad()
	if err != nil {
		fmt.Printf("Empty state: %v\n", err)
	}
	cache := initCache()
	client := createClient(state.AccessToken)

	// parse flags
	var filters filterFlags
	flag.Var(&filters, "filter", "Issue filtering, use a comma separated for AND filter and several -filter paramenters for OR filter")
	flag.Parse()

	if len(flag.Args()) < 2 {
		checkAllConfig(state, client)
		doList(*state, cache, filters)
		os.Exit(0)
	}

	command := flag.Arg(1)

	switch command {
	case "auth":
		valid, _ := client.validToken()
		if valid {
			if !askForConfirmation("There's already valid token are you sure") {
				os.Exit(0)
			}
		}
		err := client.getOauthToken()
		if err != nil {
			fmt.Printf("Error Performing oauth: %v", err)
			os.Exit(1)
		}
		state.AccessToken = client.getToken()
		valid, err = client.validToken()
		if err != nil {
			fmt.Printf("Error whith token: %v", err)
			os.Exit(1)
		}
		if valid {
			err := state.save()
			if err != nil {
				fmt.Printf("Error saving config: %v", err)
				os.Exit(1)
			}
			fmt.Printf("Auth changes will clear options, please run 'ghp config'\n")
		}
	case "config":
		valid, _ := client.validToken()
		if !valid {
			fmt.Printf("There's no valid oauth token, please run 'ghp auth'")
		}
		err = renewConfig(state)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(0)
		}
		err := state.save()
		if err != nil {
			fmt.Printf("Error saving state: %v", err)
		}
	case "help":
		doHelp()
	case "list":
		checkAllConfig(state, client)
		doList(*state, cache, filters)
	default:
		fmt.Printf("Unsupported command %v\n\n", command)
		doHelp()
	}
}
