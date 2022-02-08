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
	gh "github.com/google/go-github/v42/github"
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

func pattern() (string, error) {
	pattern := input("pattern")
	useGlob := core.GetBoolInput("use-glob")
	if useGlob {
		// Ensure the pattern is valid and eventually raise an error if not
		_, err := path.Match(pattern, "")
		if err != nil {
			core.Errorf("invalid pattern %s: %v", pattern, err)
			return "", err
		}
		return globToRegexp(pattern), nil
	}
	return pattern, nil
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
	sha := github.Context.Payload.GetBefore()
	if github.Context.Payload.PullRequest != nil {
		sha = github.Context.Payload.PullRequest.GetBase().GetSHA()
	}
	return orError(`"base" input is required when triggering on events different from pushes`, first(input("base"), sha))
}

func head() string {
	sha := github.Context.Payload.GetAfter()
	if github.Context.Payload.PullRequest != nil {
		sha = github.Context.Payload.PullRequest.GetHead().GetSHA()
	}
	return orError(`"head" input is required when triggering on events different from pushes`, first(input("head"), sha))
}

func owner() string {
	return first(input("owner"), github.Context.Repo.Owner)
}

func repo() string {
	return first(input("repo"), github.Context.Repo.Repo)
}

func modifiedFiles() []string {
	r := []string{}
	comparison, _, err := github.GitHub.Repositories.CompareCommits(context.Background(), owner(), repo(), base(), head(), &gh.ListOptions{})
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
	exp, err := pattern()
	if err != nil {
		core.Errorf("failed to compile pattern: %v", err)
		return []string{}
	}
	pattern, err := regexp.Compile(exp)
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
