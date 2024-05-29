package cmd

import (
	"flag"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tns/client"
	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/server"
)

var (
	// Define App host Url
	AppUrl  = os.Getenv("APP_URL")
	app_str string
	c       *client.Client
	logger  log.Logger
	quit    chan struct{}
	wg      sync.WaitGroup
)

func init() {
	serverConfig := server.Config{
		MetricsNamespace: "demo",
	}
	serverConfig.RegisterFlags(flag.CommandLine)
	flag.Parse()
	serverConfig.LogLevel.Set("debug")

	logger = level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout)), serverConfig.LogLevel.Gokit)

	app, err := url.Parse(AppUrl)
	if err != nil {
		level.Error(logger).Log("msg", "<push_request init> error initializing tracing", "err", err)
		return
	}
	app_str = app.String()
	c = client.New(logger)
	quit = make(chan struct{})
}

func push_request(text string) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		otrc_span_ch1 := opentracing.StartSpan("push_request_start_timer_span", opentracing.ChildOf(otrc_span.Context()))
		defer otrc_span_ch1.Finish()
		otrc_span_ch1.SetOperationName("push request Timer span")
		otrc_ctx_ch1 := opentracing.ContextWithSpan(otrc_ctx, otrc_span_ch1)
		timer := time.NewTimer(time.Duration(rand.Intn(2e3)) * time.Millisecond)
		for {
			select {
			case <-quit:
				return
			case <-timer.C:
				req, err := http.NewRequest("GET", app_str, nil)
				if err != nil {
					level.Error(logger).Log("msg", "<push_request timer> error building request", "err", err)
					return
				}
				req = req.WithContext(otrc_ctx_ch1)
				resp, err := c.Do(req)
				if err != nil {
					level.Error(logger).Log("msg", "<push_request timer> error doing request", "err", err)
					return
				}
				resp.Body.Close()
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		otrc_span_ch2 := opentracing.StartSpan("push_request_start_ticker_span", opentracing.ChildOf(otrc_span.Context()))
		defer otrc_span_ch2.Finish()
		otrc_span_ch2.SetOperationName("push request Ticker span")
		otrc_ctx_ch2 := opentracing.ContextWithSpan(otrc_ctx, otrc_span_ch2)
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				form := url.Values{}
				form.Add("text", text)
				req, err := http.NewRequest("POST", app_str+"/post", strings.NewReader(form.Encode()))
				req = req.WithContext(otrc_ctx_ch2)
				if err != nil {
					level.Error(logger).Log("msg", "<push_request ticker> error building request", "err", err)
					return
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				resp, err := c.Do(req)
				if err != nil {
					level.Error(logger).Log("msg", "<push_request ticker> error doing request", "err", err)
					return
				}
				resp.Body.Close()
				return
			}
		}
	}()

	wg.Wait()
	return
}
