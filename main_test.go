package main

import (
	"bytes"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/actions-go/toolkit/core"
	"github.com/actions-go/toolkit/github"
	gh "github.com/google/go-github/v42/github"
	"github.com/stretchr/testify/assert"
)

const (
	testBase  = "f69adcd6ffe81704a4b2b7fbd50dfedb3469ee74"
	testHead  = "dd77f9146eff4b0f053a3ff81e0198c0c85873d8"
	testOwner = "actions-go"
	testRepo  = "toolkit"
)

var (
	origEnv = getenv()
)

func getenv() map[string]string {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		kvs := strings.SplitN(kv, "=", 2)
		if len(kvs) > 1 {
			env[kvs[0]] = kvs[1]
		} else {
			env[kvs[0]] = ""
		}
	}
	return env
}

func reset() {
	clean()
	github.Context = github.ParseActionEnv()
	for k := range getenv() {
		if _, ok := origEnv[k]; !ok {
			os.Unsetenv(k)
		}
	}
	for k, v := range origEnv {
		os.Setenv(k, v)
	}
}

func clean() {
	github.Context.Payload.PushEvent = nil
	github.Context.Payload.PullRequest = nil
	github.Context.Repo.Owner = ""
	github.Context.Repo.Repo = ""
	os.Unsetenv("INPUT_HEAD")
	os.Unsetenv("INPUT_BASE")
	os.Unsetenv("INPUT_PATTERN")
	os.Unsetenv("INPUT_USE-GLOB")
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("INPUT_GITHUB_TOKEN")
	os.Unsetenv("INPUT_TOKEN")
}

func setup() {
	github.Context.Repo.Owner = testOwner
	github.Context.Repo.Repo = testRepo
	github.Context.Payload.PushEvent = &gh.PushEvent{
		Before: gh.String(testBase),
		After:  gh.String(testHead),
	}
}

func setupPR() {
	github.Context.Payload.PullRequest = &gh.PullRequest{
		Head: &gh.PullRequestBranch{
			SHA: gh.String(testHead),
		},
		Base: &gh.PullRequestBranch{
			SHA: gh.String(testBase),
		},
	}
}

func TestIntegrated(t *testing.T) {
	defer reset()
	clean()
	setup()
	os.Setenv("INPUT_PATTERN", "**/*_test.go")
	os.Setenv("INPUT_USE-GLOB", "true")
	b := bytes.NewBuffer(nil)
	core.SetStdout(b)
	main()
	assert.Contains(t, b.String(), "::set-output name=modified::true")
	assert.Contains(t, b.String(), `::set-output name=modified-files::["github/github_test.go"]`)
}

func TestBase(t *testing.T) {
	defer reset()
	clean()
	assert.Equal(t, "", base())
	setup()
	assert.Equal(t, testBase, base())
	clean()
	setupPR()
	assert.Equal(t, testBase, base())
	os.Setenv("INPUT_BASE", "1234abcd")
	assert.Equal(t, "1234abcd", base())
}

func TestHead(t *testing.T) {
	defer reset()
	clean()
	assert.Equal(t, "", head())
	setup()
	assert.Equal(t, testHead, head())
	clean()
	setupPR()
	assert.Equal(t, testHead, head())
	os.Setenv("INPUT_HEAD", "1234abcdef")
	assert.Equal(t, "1234abcdef", head())
}

func TestOwner(t *testing.T) {
	defer reset()
	clean()
	assert.Equal(t, "", owner())
	setup()
	assert.Equal(t, testOwner, owner())
}

func TestRepo(t *testing.T) {
	defer reset()
	clean()
	assert.Equal(t, "", repo())
	setup()
	assert.Equal(t, testRepo, repo())
}

func TestPattern(t *testing.T) {
	defer reset()
	clean()
	setup()
	pat, err := pattern()
	assert.NoError(t, err)
	assert.Equal(t, "", pat)
	os.Setenv("INPUT_PATTERN", ".*")
	pat, err = pattern()
	assert.NoError(t, err)
	assert.Equal(t, ".*", pat)
	os.Setenv("INPUT_USE-GLOB", "true")
	pat, err = pattern()
	assert.NoError(t, err)
	assert.Equal(t, `^\.[^/]*$`, pat)
}

func TestModified(t *testing.T) {
	defer reset()
	clean()
	assert.Equal(t, []string{}, modifiedFiles())
	setup()
	assert.Len(t, modifiedFiles(), 2)
	assert.Contains(t, modifiedFiles(), "github/github.go")
	assert.Contains(t, modifiedFiles(), "github/github_test.go")
}

func TestFilterMatching(t *testing.T) {
	defer reset()
	clean()
	os.Setenv("INPUT_PATTERN", "**/*_test.go")
	// When the pattern is invalid
	assert.Equal(t, []string{}, filterMatching([]string{"some/path/pkg_test.go"}))
	os.Setenv("INPUT_USE-GLOB", "true")
	assert.Equal(t, []string{"some/path/pkg_test.go"}, filterMatching([]string{"some/path/pkg_test.go"}))
	assert.Equal(t, []string{}, filterMatching([]string{"some/path/pkg.go"}))
	os.Setenv("INPUT_PATTERN", "[")
	assert.Equal(t, []string{}, filterMatching([]string{"some/path/pkg_test.go"}))
}

func testSamePatternBehaviour(t *testing.T, pattern, s string) {
	r := regexp.MustCompile(globToRegexp(pattern))
	matched, err := path.Match(pattern, s)
	assert.NoError(t, err)
	assert.Equal(t, matched, r.MatchString(s))
}

func TestGlobToRegex(t *testing.T) {
	assert.Equal(t, `^[^/]*/\*hello[^/]world[a-z]\.go$`, globToRegexp(`*/\*\hello?world[a-z].go`))
	assert.Equal(t, `^.*/\*hello[^/]world[a-z]\.go$`, globToRegexp(`**/\*\hello?world[a-z].go`))
	testSamePatternBehaviour(t, "*/pkg_test.go", "pkg/pkg_test.go")
	testSamePatternBehaviour(t, "*/???_tes[a-z].go", "pkg_test.go")
	testSamePatternBehaviour(t, `*/???_tes[a-z]\.go`, "pkg_test.go")
	testSamePatternBehaviour(t, `*/???_\*.go`, "pkg_*.go")
	testSamePatternBehaviour(t, "*/pkg_test.go", "some/pkg/pkg_test.go")
	assert.True(t, regexp.MustCompile(globToRegexp("**/pkg_test.go")).MatchString("some/pkg/pkg_test.go"))
	assert.True(t, regexp.MustCompile(globToRegexp("some/**")).MatchString("some/pkg/pkg_test.go"))
	assert.True(t, regexp.MustCompile(globToRegexp("some/**/*_test.go")).MatchString("some/pkg/path/pkg_test.go"))
	assert.False(t, regexp.MustCompile(globToRegexp("some/**/*_test.go")).MatchString("some/pkg/path/pkg.go"))
}

func TestAssertGlob(t *testing.T) {
	match, err := path.Match(`\*`, "*")
	assert.NoError(t, err)
	assert.True(t, match)
}
