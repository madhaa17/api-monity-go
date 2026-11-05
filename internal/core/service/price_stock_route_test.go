package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"monity/internal/config"
)

const searchBody = `{"quotes":[{"symbol":"BUMI.JK","shortname":"Bumi Resources","exchDisp":"Jakarta","quoteType":"EQUITY"}]}`

func TestSearchYahooSendsRelayTokenAndBrowserUA(t *testing.T) {
	var gotToken, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken, gotUA = r.Header.Get("X-Relay-Token"), r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(searchBody))
	}))
	defer srv.Close()

	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL, StockToken: "secret"}, nil)
	res, err := svc.SearchSymbols(context.Background(), "STOCK", "bumi")
	if err != nil || len(res) != 1 || res[0].Symbol != "BUMI.JK" {
		t.Fatalf("SearchSymbols = %v, %v", res, err)
	}
	if gotToken != "secret" {
		t.Errorf("X-Relay-Token = %q, want secret", gotToken)
	}
	if gotUA != yahooUserAgent {
		t.Errorf("User-Agent = %q, want browser UA", gotUA)
	}
}

func TestStockRequestsGoThroughStockProxy(t *testing.T) {
	// For a plain-http target, an HTTP proxy receives the request with the absolute target URL.
	var proxied atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Host == "yahoo.test" {
			proxied.Add(1)
		}
		_, _ = w.Write([]byte(searchBody))
	}))
	defer proxy.Close()

	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: "http://yahoo.test", StockProxy: proxy.URL}, nil)
	if _, err := svc.SearchSymbols(context.Background(), "STOCK", "bumi"); err != nil {
		t.Fatalf("SearchSymbols: %v", err)
	}
	if proxied.Load() != 1 {
		t.Errorf("proxy saw %d requests for yahoo.test, want 1", proxied.Load())
	}
}
