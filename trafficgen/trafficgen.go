package trafficgen

import (
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "math/rand"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "regexp"
    "time"
)

// A HTTPGenerator is an interface to dynamically navigate across a set of urls
type HTTPGenerator struct {
    RootUrls []string
    // Max number of request rounds
    MaxDepth int
    // Max number of requests per root branch per round
    MaxWidth int
    Timeout time.Duration
    // can be overwritten with SetCustomURLRegex
    httpRegex *regexp.Regexp
}

func validateHTTPGeneratorUrls(urls []string) error {
    for _, u := range urls {
        _, err := url.Parse(u)
        if err != nil {
            return err
        }
    }
    return nil
}

// NewHTTPGenerator creates a new HTTPGenerator. The generator starts by
// requesting the URLs passed in to the generator. It then scrapes the website
// for more URLs. How many requests occur depends on the maxDepth and madWidth
// arguments.
//
// maxDepth specifies the maximum number of "rounds" of requests will occur.
// maxWidth specifies the maximum number of requests per round per rootUrl
// branch.
func NewHTTPGenerator(rootUrls []string, maxDepth int, maxWidth int, timeout time.Duration) (g *HTTPGenerator, err error) {
    err = validateHTTPGeneratorUrls(rootUrls)
    if err != nil {
        return nil, err
    }
    if maxDepth < 1 {
        return nil, errors.New("maxDepth in NewGenerator must be at least 1")
    }
    if timeout <= (0 * time.Second) {
        return nil, errors.New(
            "timeout in NewGenerator must be a positive duration")
    }
    g = &HTTPGenerator{
        RootUrls: rootUrls,
        MaxDepth: maxDepth,
        MaxWidth: maxWidth,
        Timeout:  timeout,
        // default. Can be changed with setter
        httpRegex: regexp.MustCompile(`http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!'*(),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`),
    }
    return g, nil
}

func (g *HTTPGenerator) SetCustomURLRegex(r *regexp.Regexp) {
    g.httpRegex = r
}

func (g *HTTPGenerator) Start() error {
    urlsRound := make([][]string, g.MaxDepth)
    urlsRound[0] = g.RootUrls
    expired := make(chan bool, 1)
    go func() {
        time.Sleep(g.Timeout)
        expired <- true
    }()
    rand.Seed(time.Now().UTC().UnixNano())
    jar, _ := cookiejar.New(nil)
    client := &http.Client{
        Jar: jar,
    }
    for i := 0; i < g.MaxDepth; i++ {
        fmt.Printf("Round %d\n", i)
        for j := 0; j < len(urlsRound[i]); j++ {
            delay := time.After(time.Duration(rand.Intn(10)) * time.Second)
            select {
            case <- expired:
                fmt.Println("Timeout reached")
                return nil
            case <- delay:
                fmt.Printf( "Requesting %s\n", urlsRound[i][j])
                resp, err := client.Get(urlsRound[i][j])
                if err != nil {
                    fmt.Println(err)
                    continue
                }
                b := resp.Body
                urls := getUrls(b, g.httpRegex)
                if len(urls) == 0 {
                    continue
                }
                branchCount := rand.Intn(g.MaxWidth - i)
                fmt.Printf("Branch count: %d\n", branchCount)
                for k := 0; k < branchCount; k++ {
                    nextUrlIndex := rand.Intn(len(urls))
                    urlsRound[i+1] = append(urlsRound[i+1], urls[nextUrlIndex])
                }
            }
        }
    }
    return nil
}

func getUrls(body io.Reader, httpRegex *regexp.Regexp) (urls []string) {
    contents, err := ioutil.ReadAll(body)
    if err != nil {
        log.Println(err)
        return urls
    }
    urls = httpRegex.FindAllString(string(contents), -1)
    return urls
}