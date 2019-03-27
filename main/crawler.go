package main

import (
	"fmt"
	"runtime"
	"sync"
)

//
// Several solutions to the crawler exercise from the Go tutorial
// https://tour.golang.org/concurrency/10
//

//
// Serial crawler
//

func Serial(url string, fetcher Fetcher, fetched map[string]bool) {
	if fetched[url] {
		return
	}
	fetched[url] = true
	_, urls, err := fetcher.Fetch(url)
	if err != nil {
		return
	}
	for _, u := range urls {
		Serial(u, fetcher, fetched)
	}
	return
}

//
// Concurrent crawler with shared state and Mutex
//

type fetchState struct {
	mu      sync.Mutex
	/*A locked Mutex is not associated with a particular goroutine.
	It is allowed for one goroutine to lock a Mutex and then arrange for another goroutine to unlock it.
	 mutex和goroutine无关,可以一个goroutine加锁,另一个解锁,所以一般每个goroutine都是先加锁再解锁*/
	fetched map[string]bool//各个goroutine共享的变量
}

func makeState() *fetchState {//初始化
	f := &fetchState{}
	f.fetched = make(map[string]bool)
	return f
}

func ConcurrentMutex(url string, fetcher Fetcher, f *fetchState) {
	f.mu.Lock()
	if f.fetched[url] {
		f.mu.Unlock()//不同分支的Unlock
		return
	}
	f.fetched[url] = true
	f.mu.Unlock()//不同分支的Unlock

	_, urls, err := fetcher.Fetch(url)
	if err != nil {
		return
	}
	var done sync.WaitGroup
	for _, u := range urls {
		done.Add(1)
		go func(u string) {
			defer done.Done()
			ConcurrentMutex(u, fetcher, f)
		}(u)
	}
	done.Wait()
	return
}

//
// Concurrent crawler with channels
//

func worker(url string, ch chan []string, fetcher Fetcher) {
	_, urls, err := fetcher.Fetch(url)
	if err != nil {
		ch <- []string{}//必不可少,否则n值就不对了
	} else {
		ch <- urls
	}
}

func master(ch chan []string, fetcher Fetcher) {
	n := 1//n记录的是有多少worker goroutine（master本身是一个worker）在向ch中写入[]string,当n==0时应该停止读取ch,否则会出现deadlock
	//简单地说,n记录的是有多少worker在工作，当n==0时,工作已经完成
	fetched := make(map[string]bool)//由master管理全局信息,非共享变量
	for urls := range ch {//死循环不断的读取ch
		//fmt.Println("range channel urls = ", urls)
		for _, u := range urls {
			if fetched[u] == false {
				fetched[u] = true
				n += 1//增加一个worker,意味着会有一个worker向ch中写入[]string
				go worker(u, ch, fetcher)
			}
		}
		n -= 1//一个worker goroutine完成
		if n == 0 {
			break
		}
	}
}

func ConcurrentChannel(url string, fetcher Fetcher) {
	ch := make(chan []string)
	go func() {
		ch <- []string{url}//ch中记录着需要被处理的url
	}()
	master(ch, fetcher)
}

//
// main
//

func main() {
	runtime.GOMAXPROCS(8)
	fmt.Printf("=== Serial===\n")
	Serial("http://golang.org/", fetcher, make(map[string]bool))

	fmt.Printf("=== ConcurrentMutex ===\n")
	ConcurrentMutex("http://golang.org/", fetcher, makeState())

	fmt.Printf("=== ConcurrentChannel ===\n")
	ConcurrentChannel("http://golang.org/", fetcher)
}

//
// Fetcher
//

type Fetcher interface {
	// Fetch returns a slice of URLs found on the page.
	Fetch(url string) (body string, urls []string, err error)
}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		fmt.Printf("found:   %s\t%q\n", url, res.body)
		return res.body, res.urls, nil
	}
	fmt.Printf("missing: %s\n", url)
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"http://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"http://golang.org/pkg/",
			"http://golang.org/cmd/",
		},
	},
	"http://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"http://golang.org/",
			"http://golang.org/cmd/",
			"http://golang.org/pkg/fmt/",
			"http://golang.org/pkg/os/",
		},
	},
	"http://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"http://golang.org/",
			"http://golang.org/pkg/",
		},
	},
	"http://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"http://golang.org/",
			"http://golang.org/pkg/",
		},
	},
}
