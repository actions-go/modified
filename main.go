package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"regexp"
	"strings"

	"github.com/actions-go/toolkit/core"
	"github.com/actions-go/toolkit/github"
)

func globToRegexp(g string) string {
	// TODO handle windows cases from https://golang.org/src/path/filepath/match.go
	r := "^"
	idx := 0
	l := len(g)
	for ; idx < l; idx++ {
		c := g[idx]
		switch c {
		case '\\':
			if idx+1 < l {
				switch g[idx+1] {
				case '*', '?', '\\', '[':
					r += `\`
				}
				r += string(g[idx+1])
				idx++
			} else {
				r += string(c)
			}
		case '.':
			r += `\.`
		case '?':
			r += `[^/]`
		case '*':
			pattern := `[^/]*`
			if idx+1 < l && g[idx+1] == '*' {
				pattern = ".*"
				idx++
			}
			r += pattern
		default:
			r += string(c)
		}
	}
	return r + "$"
}

func pattern() string {
	pattern := input("pattern")
	useGlob := core.GetBoolInput("use-glob")
	if useGlob {
		// Ensure the pattern is valid and eventually raise an error if not
		_, err := path.Match(pattern, "")
		if err != nil {
			core.Errorf("invalid pattern %s: %v", pattern, err)
			return ""
		}
		return globToRegexp(pattern)
	}
	return pattern
}

func first(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

func input(name string) string {
	return core.GetInputOrDefault(name, "")
}

func orError(msg, s string) string {
	if s == "" {
		core.Errorf(msg)
	}
	return s
}

func base() string {
	return orError(`"base" input is required when triggering on events different from pushes`, first(input("base"), github.Context.Payload.GetBefore()))
}

func head() string {
	return orError(`"head" input is required when triggering on events different from pushes`, first(input("head"), github.Context.Payload.GetAfter()))
}

func owner() string {
	return first(input("owner"), github.Context.Repo.Owner)
}

func repo() string {
	return first(input("repo"), github.Context.Repo.Repo)
}

func modifiedFiles() []string {
	r := []string{}
	comparison, _, err := github.GitHub.Repositories.CompareCommits(context.Background(), owner(), repo(), base(), head())
	if err != nil {
		core.Errorf("failed to compare commits through the API: %v", err)
		return r
	}
	for _, f := range comparison.Files {
		r = append(r, f.GetFilename())
	}
	return r
}

func filterMatching(paths []string) []string {
	pattern, err := regexp.Compile(pattern())
	if err != nil {
		core.Errorf("failed to compile pattern: %v", err)
		return []string{}
	}
	r := []string{}
	for _, name := range paths {
		if pattern.MatchString(name) {
			r = append(r, name)
		}
	}
	return r
}

func toJSON(v interface{}) string {
	b := bytes.NewBuffer(nil)
	err := json.NewEncoder(b).Encode(v)
	if err != nil {
		core.Errorf("failed to run json encoding for %v: %v", v, err)
		return ""
	}
	return strings.Trim(b.String(), "\n")
}

func main() {
	matchingFiles := filterMatching(modifiedFiles())
	core.SetOutput("modified", toJSON(len(matchingFiles) > 0))
	core.SetOutput("modified-files", toJSON(matchingFiles))
}
