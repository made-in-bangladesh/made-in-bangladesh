package made_in_bangladesh

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/octokit/go-octokit/octokit"
	"github.com/russross/blackfriday"
)

// Following test file acknowledged thankfully `avelino/awesome-go` for the tests

var query = startQuery()

func TestDuplicate(t *testing.T) {
	links := make(map[string]bool, 0)

	query.Find("body li > a:first-child").Each(func(_ int, s *goquery.Selection) {
		t.Run(s.Text(), func(t *testing.T) {
			href, ok := s.Attr("href")
			if !ok {
				t.Error("expected to have href")
			}

			if links[href] {
				t.Fatalf("duplicated link '%s'", href)
			}

			links[href] = true
		})
	})

	sections := make(map[string]struct{}, 0)
	query.Find("body > ul > li").Each(func(_ int, s *goquery.Selection) {
		section := strings.Fields(strings.TrimSpace(s.Text()))
		if len(section) == 0 {
			t.Fatal("no section header found")
		}

		if _, found := sections[section[0]]; found {
			t.Fatalf("duplicated section '%s'", section[0])
		}

		sections[section[0]] = struct{}{}
	})
}

func TestSorted(t *testing.T) {
	var sections []string
	query.Find("body > ul > li").Each(func(_ int, s *goquery.Selection) {
		section := strings.Fields(strings.TrimSpace(s.Text()))
		if len(section) == 0 {
			t.Fatal("no section header found")
		}
		sections = append(sections, section[0])
	})
	checkSorted(t, sections)

	query.Find("body > ul").Each(func(_ int, s *goquery.Selection) {
		testList(t, s)
	})
}

// Test if an entry has description, it must be separated from link with ` - `
func TestSeparator(t *testing.T) {
	var matched, containsLink, noDescription bool
	input, err := ioutil.ReadFile("./README.md")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(input), "\n")
	for _, line := range lines {
		line = strings.Trim(line, " ")
		containsLink = reContainsLink.MatchString(line)
		if containsLink {
			noDescription = reOnlyLink.MatchString(line)
			if noDescription {
				continue
			}

			matched = reLinkWithDescription.MatchString(line)
			if !matched {
				t.Errorf("expected entry to be in form of `* [link] - description`, got '%s'", line)
			}
		}
	}
}

const (
	requiredStarCount = 10
)

func TestStarCount(t *testing.T) {
	cl := octokit.NewClient(nil)

	var matched, containsLink, noDescription bool
	input, err := ioutil.ReadFile("./README.md")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(input), "\n")
	for _, line := range lines {
		line = strings.Trim(line, " ")
		containsLink = reContainsLink.MatchString(line)
		if containsLink {
			noDescription = reOnlyLink.MatchString(line)
			if noDescription {
				continue
			}

			matched = reLinkWithDescription.MatchString(line)
			if matched {
				link := strings.TrimSpace(line[strings.Index(line, "(")+1 : strings.Index(line, ")")])
				if strings.HasPrefix(link, "https://github.com") {
					// Only support github for now.
					parts := strings.Split(link[19:], "/")
					repo, res := cl.Repositories().One(&octokit.RepositoryURL, octokit.M{
						"owner": parts[0],
						"repo":  parts[1],
					})
					if res.Err != nil {
						panic(res.Err)
					}

					if repo.StargazersCount < requiredStarCount {
						t.Fatal("repository didn't meet expected star count")
					}
				}
			}
		}
	}
}

func testList(t *testing.T, list *goquery.Selection) {
	list.Find("ul").Each(func(_ int, items *goquery.Selection) {
		testList(t, items)
		items.RemoveFiltered("ul")
	})

	category := list.Prev().Text()

	t.Run(category, func(t *testing.T) {
		checkAlphabeticOrder(t, list)
	})
}

func checkAlphabeticOrder(t *testing.T, s *goquery.Selection) {
	items := s.Find("li > a:first-child").Map(func(_ int, li *goquery.Selection) string {
		return strings.ToLower(li.Text())
	})
	checkSorted(t, items)
}

func checkSorted(t *testing.T, items []string) {
	sorted := make([]string, len(items))
	copy(sorted, items)
	sort.Strings(sorted)

	for k, item := range items {
		if item != sorted[k] {
			t.Errorf("expected '%s' but actual is '%s'", sorted[k], item)
		}
	}
	if t.Failed() {
		t.Logf("expected order is:\n%s", strings.Join(sorted, "\n"))
	}
}

var (
	reContainsLink        = regexp.MustCompile(`\* \[.*\]\(.*\)`)
	reOnlyLink            = regexp.MustCompile(`\* \[.*\]\(.*\)$`)
	reLinkWithDescription = regexp.MustCompile(`\* \[.*\]\(.*\) - \S`)
)

func readme() []byte {
	input, err := ioutil.ReadFile("./README.md")
	if err != nil {
		panic(err)
	}

	html := append([]byte("<body>"), blackfriday.MarkdownCommon(input)...)
	html = append(html, []byte("</body>")...)
	return html
}

func startQuery() *goquery.Document {
	buf := bytes.NewBuffer(readme())
	query, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		panic(err)
	}

	return query
}
