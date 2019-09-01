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

// A Generator is an interface to dynamically navigate across a set of urls
type Generator struct {
    RootUrls []string
    MaxDepth int
    Timeout time.Duration
    httpRegex *regexp.Regexp
}

func validateGeneratorUrls(urls []string) error {
    for _, u := range urls {
        _, err := url.Parse(u)
        if err != nil {
            return err
        }
    }
    return nil
}

func NewGenerator(urls []string, maxDepth int, timeout time.Duration) (g *Generator, err error) {
    err = validateGeneratorUrls(urls)
    if err != nil {
        return nil, err
    }
    if maxDepth < 1 {
        return nil, errors.New("maxDepth in NewGenerator must be at least 1")
    }
    if timeout <= (0 * time.Second) {
        return nil, errors.New("timeout in NewGenerator must be a positive duration")
    }
    g = &Generator{
        RootUrls:  urls,
        MaxDepth:  maxDepth,
        Timeout:   timeout,
        // default. Can be changed with setter
        httpRegex: regexp.MustCompile(`http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*(),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`),
    }
    return g, nil
}

func (g *Generator) SetCustomURLRegex(r *regexp.Regexp) {
    g.httpRegex = r
}

func (g *Generator) Start() error {
    urlsRound := make([][]string, g.MaxDepth)
    expired := make(chan bool, 1)
    go func() {
        time.Sleep(g.Timeout)
        expired <- true
    }()
    rand.Seed(time.Now().UTC().UnixNano())
    jar, _ := cookiejar.New(nil)
    for _, initUrl := range g.RootUrls {
        urlsRound[0] = append(urlsRound[0], initUrl)
    }
    client := &http.Client{
        Jar: jar,
    }
    for i := 0; i < len(urlsRound); i++ {
        fmt.Printf("Round %d\n", i)
        for j := 0; j < len(urlsRound[i]); j++ {
            delay := time.After(time.Duration(rand.Intn(10)) * time.Second)
            select {
            case <- expired:
                return nil
            case <- delay:
                resp, err := client.Get(urlsRound[i][j])
                if err != nil {
                    continue
                }
                b := resp.Body
                urls := getUrls(b, g.httpRegex)
                if len(urls) == 0 {
                    continue
                }
                branchCount := rand.Intn(g.MaxDepth - i)
                fmt.Printf("branch count: %d\n", branchCount)
                for k := 0; k < branchCount; k++ {
                    nextUrlIndex := rand.Intn(len(urls))
                    fmt.Printf("Next URL index: %d\n", nextUrlIndex)
                    urlsRound[i+1] = append(urlsRound[i+1], urls[nextUrlIndex])
                }
            }
        }
    }
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