package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/cache"

	"github.com/shopspring/decimal"
)

func TestYahooSymbolCandidates(t *testing.T) {
	tests := []struct {
		symbol string
		want   []string
	}{
		{"BBCA", []string{"BBCA.JK"}},         // known IDX stock
		{"BUMI", []string{"BUMI.JK", "BUMI"}}, // unknown: IDX first, then bare
		{"AAPL", []string{"AAPL.JK", "AAPL"}},
		{"BUMI.JK", []string{"BUMI.JK"}}, // explicit suffix is used as is
	}
	for _, tt := range tests {
		if got := yahooSymbolCandidates(tt.symbol); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("yahooSymbolCandidates(%q) = %v, want %v", tt.symbol, got, tt.want)
		}
	}
}

// fakeYahoo serves the chart endpoint for the given Yahoo symbols and 404 for anything else,
// recording the symbols that were requested.
func fakeYahoo(t *testing.T, known ...string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := strings.TrimPrefix(r.URL.Path, "/v8/finance/chart/")
		mu.Lock()
		requested = append(requested, symbol)
		mu.Unlock()
		for _, k := range known {
			if k == symbol {
				fmt.Fprint(w, `{"chart":{"result":[{"meta":{"currency":"IDR","regularMarketPrice":140},`+
					`"timestamp":[1,2],"indicators":{"quote":[{"close":[135,140]}]}}],"error":null}}`)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found"}}}`)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), requested...)
	}
}

func TestStockPriceAndChart_UnlistedIDXTickerUsesJK(t *testing.T) {
	srv, requested := fakeYahoo(t, "BUMI.JK")
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	price, err := svc.GetStockPriceWithCurrency(context.Background(), "bumi", "IDR")
	if err != nil {
		t.Fatalf("GetStockPriceWithCurrency: %v", err)
	}
	if price.Price != 140 || price.Symbol != "BUMI" {
		t.Errorf("price = %+v, want BUMI at 140", price)
	}

	chart, err := svc.GetStockChart(context.Background(), "BUMI", "1mo", "1d")
	if err != nil {
		t.Fatalf("GetStockChart: %v", err)
	}
	if len(chart.Data) != 2 || chart.Data[1].P != 140 {
		t.Errorf("chart data = %+v, want two points ending at 140", chart.Data)
	}
	if got := requested(); !reflect.DeepEqual(got, []string{"BUMI.JK", "BUMI.JK"}) {
		t.Errorf("requested %v, want only BUMI.JK", got)
	}
}

func TestStockPrice_FallsBackToBareSymbol(t *testing.T) {
	srv, requested := fakeYahoo(t, "AAPL")
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	if _, err := svc.GetStockPriceWithCurrency(context.Background(), "AAPL", "IDR"); err != nil {
		t.Fatalf("GetStockPriceWithCurrency: %v", err)
	}
	if got := requested(); !reflect.DeepEqual(got, []string{"AAPL.JK", "AAPL"}) {
		t.Errorf("requested %v, want AAPL.JK then AAPL", got)
	}

	// The resolved symbol is remembered: the chart goes straight to AAPL.
	if _, err := svc.GetStockChart(context.Background(), "AAPL", "1mo", "1d"); err != nil {
		t.Fatalf("GetStockChart: %v", err)
	}
	if got := requested(); !reflect.DeepEqual(got, []string{"AAPL.JK", "AAPL", "AAPL"}) {
		t.Errorf("requested %v, want no second AAPL.JK probe", got)
	}
}

func TestStockChart_UnknownSymbolReturnsError(t *testing.T) {
	srv, _ := fakeYahoo(t)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	_, err := svc.GetStockChart(context.Background(), "ZZZZ", "1mo", "1d")
	if err == nil || !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "ZZZZ") {
		t.Errorf("err = %v, want a 404 error naming ZZZZ", err)
	}
}

func TestStockPrice_LotSizeFromResolvedSymbol(t *testing.T) {
	srv, _ := fakeYahoo(t, "BUMI.JK", "AAPL")
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	for symbol, want := range map[string]int64{"BUMI": IDXLotSize, "AAPL": 0} {
		price, err := svc.GetStockPriceWithCurrency(context.Background(), symbol, "IDR")
		if err != nil {
			t.Fatalf("%s: %v", symbol, err)
		}
		if price.LotSize != want {
			t.Errorf("%s LotSize = %d, want %d", symbol, price.LotSize, want)
		}
	}
}

func TestEffectiveQuantity(t *testing.T) {
	sym := func(s string) *string { return &s }
	qty := decimal.NewFromInt(347)
	tests := []struct {
		name  string
		asset models.Asset
		price *port.PriceData
		want  int64
	}{
		{"unlisted IDX stock priced via .JK counts lots", models.Asset{Type: models.AssetTypeStock, Symbol: sym("BUMI"), Quantity: qty}, &port.PriceData{LotSize: IDXLotSize}, 34700},
		{"US stock counts shares", models.Asset{Type: models.AssetTypeStock, Symbol: sym("AAPL"), Quantity: qty}, &port.PriceData{}, 347},
		{"known IDX stock without a price counts lots", models.Asset{Type: models.AssetTypeStock, Symbol: sym("BBCA"), Quantity: qty}, nil, 34700},
		{".JK symbol without a price counts lots", models.Asset{Type: models.AssetTypeStock, Symbol: sym("bumi.jk"), Quantity: qty}, nil, 34700},
		{"crypto is never in lots", models.Asset{Type: models.AssetTypeCrypto, Symbol: sym("BTC"), Quantity: qty}, &port.PriceData{LotSize: IDXLotSize}, 347},
	}
	for _, tt := range tests {
		if got := effectiveQuantity(&tt.asset, tt.price); !got.Equal(decimal.NewFromInt(tt.want)) {
			t.Errorf("%s: got %s, want %d", tt.name, got, tt.want)
		}
	}
}

// flakyYahoo fails the first `failures` requests with status, then serves BUMI.JK at 140.
func flakyYahoo(t *testing.T, failures int, status int) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n <= failures {
			w.WriteHeader(status)
			return
		}
		fmt.Fprint(w, `{"chart":{"result":[{"meta":{"currency":"IDR","regularMarketPrice":140},`+
			`"timestamp":[1,2],"indicators":{"quote":[{"close":[135,140]}]}}],"error":null}}`)
	}))
	t.Cleanup(srv.Close)
	return srv, func() int { mu.Lock(); defer mu.Unlock(); return calls }
}

func noRetryDelay(t *testing.T) {
	t.Helper()
	saved := retryDelays
	retryDelays = []time.Duration{0, 0}
	t.Cleanup(func() { retryDelays = saved })
}

func TestStockPrice_RetriesTransientFailures(t *testing.T) {
	noRetryDelay(t)
	srv, calls := flakyYahoo(t, 2, http.StatusServiceUnavailable)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	price, err := svc.GetStockPriceWithCurrency(context.Background(), "BUMI.JK", "IDR")
	if err != nil {
		t.Fatalf("GetStockPriceWithCurrency: %v", err)
	}
	if price.Price != 140 || price.Stale {
		t.Errorf("price = %+v, want a fresh 140", price)
	}
	if calls() != 3 {
		t.Errorf("calls = %d, want 3 (two 503s, then success)", calls())
	}
}

func TestStockPrice_DoesNotRetryClientErrors(t *testing.T) {
	noRetryDelay(t)
	srv, calls := flakyYahoo(t, 1, http.StatusBadRequest)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	if _, err := svc.GetStockPriceWithCurrency(context.Background(), "BUMI.JK", "IDR"); err == nil {
		t.Fatal("want an error for a 400")
	}
	if calls() != 1 {
		t.Errorf("calls = %d, want 1", calls())
	}
}

func TestStockPriceAndChart_LastKnownWhenYahooDown(t *testing.T) {
	noRetryDelay(t)
	c := cache.NewMemoryCache()
	asOf := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	lastPrice, _ := json.Marshal(port.PriceData{Symbol: "BUMI", Price: 150, Currency: "IDR", Source: "yahoo", FetchedAt: asOf, LotSize: IDXLotSize})
	lastChart, _ := json.Marshal(port.ChartResponse{Symbol: "BUMI", Currency: "IDR", Data: []port.ChartDataPoint{{T: 1, P: 150}}})
	_ = c.Set(context.Background(), "last:stock:v2:BUMI:IDR", lastPrice, time.Hour)
	_ = c.Set(context.Background(), "last:chart:stock:BUMI:1y:1d", lastChart, time.Hour)

	srv, _ := flakyYahoo(t, 1000, http.StatusServiceUnavailable)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, c)

	price, err := svc.GetStockPriceWithCurrency(context.Background(), "BUMI", "IDR")
	if err != nil {
		t.Fatalf("GetStockPriceWithCurrency: %v", err)
	}
	if price.Price != 150 || !price.Stale || !price.FetchedAt.Equal(asOf) || price.LotSize != IDXLotSize {
		t.Errorf("price = %+v, want the stale last-known 150 from %s", price, asOf)
	}

	chart, err := svc.GetStockChart(context.Background(), "BUMI", "1mo", "1d")
	if err != nil {
		t.Fatalf("GetStockChart: %v", err)
	}
	if !chart.Stale || len(chart.Data) != 1 {
		t.Errorf("chart = %+v, want the stale last-known chart", chart)
	}
}

func TestStockPrice_SuccessIsKeptAsLastKnown(t *testing.T) {
	srv, _ := fakeYahoo(t, "BUMI.JK")
	c := cache.NewMemoryCache()
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, c)

	if _, err := svc.GetStockPriceWithCurrency(context.Background(), "BUMI", "IDR"); err != nil {
		t.Fatalf("GetStockPriceWithCurrency: %v", err)
	}
	raw, err := c.Get(context.Background(), "last:stock:v2:BUMI:IDR")
	if err != nil || !strings.Contains(string(raw), `"price":140`) {
		t.Errorf("last-known entry = %s (err %v), want the fetched price", raw, err)
	}
}

func TestSearchYahoo_IDXFirstAndOnlyStocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/finance/search" || r.URL.Query().Get("q") != "bbca" {
			t.Errorf("unexpected request %s", r.URL)
		}
		fmt.Fprint(w, `{"quotes":[`+
			`{"symbol":"BBCA","shortname":"JPM Canada ETF","exchDisp":"BATS","quoteType":"ETF"},`+
			`{"symbol":"BBCA=F","shortname":"Futures","exchDisp":"CME","quoteType":"FUTURE"},`+
			`{"symbol":"BBCA.JK","shortname":"Bank Central Asia","longname":"PT Bank Central Asia Tbk","exchDisp":"Jakarta","quoteType":"EQUITY"}]}`)
	}))
	t.Cleanup(srv.Close)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	got, err := svc.SearchSymbols(context.Background(), "STOCK", "bbca")
	if err != nil {
		t.Fatal(err)
	}
	want := []port.SymbolSearchResult{
		{Symbol: "BBCA.JK", Name: "PT Bank Central Asia Tbk", Exchange: "Jakarta", Kind: "EQUITY", LotSize: IDXLotSize},
		{Symbol: "BBCA", Name: "JPM Canada ETF", Exchange: "BATS", Kind: "ETF"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SearchSymbols = %+v, want %+v", got, want)
	}
}

// fakeCoinGecko serves /search with the given coins JSON and /simple/price for any id at 100.
func fakeCoinGecko(t *testing.T, coins string, gotKey *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotKey != nil {
			*gotKey = r.Header.Get("x-cg-demo-api-key")
		}
		switch r.URL.Path {
		case "/search":
			fmt.Fprintf(w, `{"coins":%s}`, coins)
		case "/simple/price":
			id := r.URL.Query().Get("ids")
			fmt.Fprintf(w, `{%q:{"idr":100}}`, id)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCryptoPrice_ResolvesUnmappedTickerToTopRankedCoin(t *testing.T) {
	var key string
	srv := fakeCoinGecko(t, `[`+
		`{"id":"pepe-fake","name":"Fake Pepe","symbol":"pepe","market_cap_rank":0},`+
		`{"id":"pepe","name":"Pepe","symbol":"pepe","market_cap_rank":30},`+
		`{"id":"pepecoin","name":"PepeCoin","symbol":"pepecoin","market_cap_rank":900}]`, &key)
	svc := NewPriceService(&config.PriceAPIConfig{CryptoAPI: srv.URL, CryptoAPIKey: "demo-key"}, nil)

	price, err := svc.GetCryptoPriceWithCurrency(context.Background(), "pepe", "IDR")
	if err != nil {
		t.Fatal(err)
	}
	if price.Symbol != "PEPE" || price.Price != 100 {
		t.Errorf("price = %+v, want PEPE at 100", price)
	}
	if key != "demo-key" {
		t.Errorf("demo API key header = %q, want demo-key", key)
	}

	results, err := svc.SearchSymbols(context.Background(), "CRYPTO", "pepe")
	if err != nil {
		t.Fatal(err)
	}
	// Ranked coins first, and only the coin the PEPE ticker resolves to.
	if len(results) != 2 || results[0].CoinID != "pepe" || results[1].CoinID != "pepecoin" {
		t.Errorf("search = %+v, want [pepe pepecoin]", results)
	}
}

func TestCryptoPrice_UnknownTickerIsUnsupported(t *testing.T) {
	srv := fakeCoinGecko(t, `[{"id":"other","name":"Other","symbol":"oth","market_cap_rank":5}]`, nil)
	svc := NewPriceService(&config.PriceAPIConfig{CryptoAPI: srv.URL}, nil)

	_, err := svc.GetCryptoPriceWithCurrency(context.Background(), "NOPE", "IDR")
	if err == nil || !strings.Contains(err.Error(), "unsupported crypto symbol") {
		t.Errorf("err = %v, want unsupported crypto symbol", err)
	}
}

func TestExchangeRate_FallsBackWhenYahooFails(t *testing.T) {
	yahoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(yahoo.Close)
	fx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest/USD" {
			t.Errorf("unexpected FX request %s", r.URL)
		}
		fmt.Fprint(w, `{"result":"success","rates":{"IDR":16000}}`)
	}))
	t.Cleanup(fx.Close)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: yahoo.URL, FXAPI: fx.URL}, nil).(*PriceService)

	rate, err := svc.getExchangeRate(context.Background(), "USD", "IDR")
	if err != nil {
		t.Fatal(err)
	}
	if rate != 16000 {
		t.Errorf("rate = %v, want 16000", rate)
	}
}

func TestStockChart_SkipsNullCloses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"chart":{"result":[{"meta":{"currency":"IDR"},`+
			`"timestamp":[1,2,3],"indicators":{"quote":[{"close":[135,null,140]}]}}],"error":null}}`)
	}))
	t.Cleanup(srv.Close)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	chart, err := svc.GetStockChart(context.Background(), "BBCA", "1mo", "1d")
	if err != nil {
		t.Fatalf("GetStockChart: %v", err)
	}
	want := []port.ChartDataPoint{{T: 1, P: 135}, {T: 3, P: 140}}
	if !reflect.DeepEqual(chart.Data, want) {
		t.Errorf("chart data = %+v, want %+v (null close dropped, not 0)", chart.Data, want)
	}
}

func TestMarketAwareTTL(t *testing.T) {
	// A session from 09:00 to 16:00 on day 0 (Unix seconds from the epoch).
	var period yahooTradingPeriod
	period.Regular.Start = 9 * 3600
	period.Regular.End = 16 * 3600
	at := func(h, m int) time.Time { return time.Unix(int64(h*3600+m*60), 0) }
	open := time.Minute

	tests := []struct {
		name   string
		now    time.Time
		period yahooTradingPeriod
		want   time.Duration
	}{
		{"unknown session", at(12, 0), yahooTradingPeriod{}, open},
		{"open", at(12, 0), period, open},
		{"before open", at(7, 0), period, 2 * time.Hour},
		{"just before open", at(8, 59).Add(30 * time.Second), period, open},
		{"after close: next day's open, capped", at(17, 0), period, closedMaxTTL},
		{"late after close", at(30, 0), period, 3 * time.Hour},
		{"next open passed (weekend)", at(40, 0), period, closedMinTTL},
	}
	for _, tt := range tests {
		if got := marketAwareTTL(tt.now, tt.period, open); got != tt.want {
			t.Errorf("%s: marketAwareTTL = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSliceChart(t *testing.T) {
	day := int64(24 * 3600)
	last := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC).Unix()
	var data []port.ChartDataPoint
	for d := int64(200); d >= 0; d-- {
		data = append(data, port.ChartDataPoint{T: last - d*day, P: float64(d)})
	}
	chart := &port.ChartResponse{Symbol: "BBCA", Currency: "IDR", Data: data}

	if got := sliceChart(chart, 0).Data; len(got) != 5 || got[4].T != last {
		t.Errorf("5d: got %d points, want the last 5", len(got))
	}
	monthAgo := time.Unix(last, 0).AddDate(0, -1, 0).Unix()
	if got := sliceChart(chart, 1).Data; got[0].T != monthAgo || got[len(got)-1].T != last {
		t.Errorf("1mo: got %d..%d, want %d..%d", got[0].T, got[len(got)-1].T, monthAgo, last)
	}
	if got := sliceChart(chart, 12); got != chart {
		t.Errorf("1y: want the chart unchanged")
	}
	if len(chart.Data) != 201 {
		t.Errorf("slicing changed the source chart")
	}
}

func TestStockChart_RangesShareOneFetch(t *testing.T) {
	srv, requested := fakeYahoo(t, "BBCA.JK")
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	for _, rng := range []string{"5d", "1mo", "3mo", "6mo", "1y"} {
		if _, err := svc.GetStockChart(context.Background(), "BBCA", rng, "1d"); err != nil {
			t.Fatalf("GetStockChart(%s): %v", rng, err)
		}
	}
	if got := requested(); len(got) != 1 {
		t.Errorf("Yahoo requested %d times, want 1 (one 1y chart sliced per range)", len(got))
	}
}

func TestStockChart_ConcurrentMissesShareOneFetch(t *testing.T) {
	var calls int
	var mu sync.Mutex
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		<-release
		fmt.Fprint(w, `{"chart":{"result":[{"meta":{"currency":"IDR"},`+
			`"timestamp":[1,2],"indicators":{"quote":[{"close":[135,140]}]}}],"error":null}}`)
	}))
	t.Cleanup(srv.Close)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.GetStockChart(context.Background(), "BBCA", "1mo", "1d"); err != nil {
				t.Errorf("GetStockChart: %v", err)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond) // let all five reach the in-flight request
	close(release)
	wg.Wait()
	if calls != 1 {
		t.Errorf("Yahoo requested %d times, want 1", calls)
	}
}

// usdYahooWithoutFX serves AAPL in USD and fails every exchange rate request, as does the FX API.
func usdYahooWithoutFX(t *testing.T) *PriceService {
	t.Helper()
	noRetryDelay(t)
	yahoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/AAPL") {
			fmt.Fprint(w, `{"chart":{"result":[{"meta":{"currency":"USD","regularMarketPrice":180}}],"error":null}}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(yahoo.Close)
	fx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(fx.Close)
	return NewPriceService(&config.PriceAPIConfig{StockAPI: yahoo.URL, FXAPI: fx.URL}, cache.NewMemoryCache()).(*PriceService)
}

func TestStockPrice_NoExchangeRateKeepsSourceCurrency(t *testing.T) {
	svc := usdYahooWithoutFX(t)

	for i := 0; i < 2; i++ { // the second call is served from the cache
		price, err := svc.GetStockPriceWithCurrency(context.Background(), "AAPL", "IDR")
		if err != nil {
			t.Fatalf("GetStockPriceWithCurrency: %v", err)
		}
		if price.Price != 180 || price.Currency != "USD" {
			t.Errorf("call %d: price = %v %s, want 180 USD (never labeled IDR)", i, price.Price, price.Currency)
		}
	}
}

func TestStockPrice_NoExchangeRateUsesLastConvertedPrice(t *testing.T) {
	svc := usdYahooWithoutFX(t)
	asOf := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	last, _ := json.Marshal(port.PriceData{Symbol: "AAPL", Price: 2_900_000, Currency: "IDR", Source: "yahoo", FetchedAt: asOf})
	_ = svc.cache.Set(context.Background(), "last:stock:v2:AAPL:IDR", last, time.Hour)

	for i := 0; i < 2; i++ {
		price, err := svc.GetStockPriceWithCurrency(context.Background(), "AAPL", "IDR")
		if err != nil {
			t.Fatalf("GetStockPriceWithCurrency: %v", err)
		}
		if price.Price != 2_900_000 || price.Currency != "IDR" || !price.Stale {
			t.Errorf("call %d: price = %+v, want the stale last-known IDR price", i, price)
		}
	}
}

// flakyCoinGecko fails the first failures requests with status (sending retryAfter as Retry-After
// when set) and then serves BTC at 100 and a two-point chart, counting the calls.
func flakyCoinGecko(t *testing.T, failures, status int, retryAfter string) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n <= failures {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			w.WriteHeader(status)
			return
		}
		switch r.URL.Path {
		case "/simple/price":
			fmt.Fprint(w, `{"bitcoin":{"idr":100}}`)
		case "/coins/bitcoin/market_chart":
			fmt.Fprint(w, `{"prices":[[1000,90],[2000,100]]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}
}

func TestCryptoPriceAndChart_RetryTransientFailures(t *testing.T) {
	noRetryDelay(t)
	srv, calls := flakyCoinGecko(t, 2, http.StatusTooManyRequests, "")
	svc := NewPriceService(&config.PriceAPIConfig{CryptoAPI: srv.URL}, nil)

	price, err := svc.GetCryptoPriceWithCurrency(context.Background(), "BTC", "IDR")
	if err != nil {
		t.Fatalf("GetCryptoPriceWithCurrency: %v", err)
	}
	if price.Price != 100 || price.Stale || calls() != 3 {
		t.Errorf("price = %+v after %d calls, want a fresh 100 after 3", price, calls())
	}

	chart, err := svc.GetCryptoChart(context.Background(), "BTC", "IDR", 7)
	if err != nil {
		t.Fatalf("GetCryptoChart: %v", err)
	}
	if len(chart.Data) != 2 || chart.Stale {
		t.Errorf("chart = %+v, want the fresh two-point chart", chart)
	}
}

func TestCryptoPriceAndChart_LastKnownWhenCoinGeckoDown(t *testing.T) {
	noRetryDelay(t)
	c := cache.NewMemoryCache()
	asOf := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	lastPrice, _ := json.Marshal(port.PriceData{Symbol: "BTC", Price: 95, Currency: "IDR", Source: "coingecko", FetchedAt: asOf})
	lastChart, _ := json.Marshal(port.ChartResponse{Symbol: "BTC", Currency: "IDR", Data: []port.ChartDataPoint{{T: 1, P: 95}}})
	_ = c.Set(context.Background(), "last:crypto:BTC:IDR", lastPrice, time.Hour)
	_ = c.Set(context.Background(), "last:chart:crypto:BTC:IDR:7", lastChart, time.Hour)

	srv, _ := flakyCoinGecko(t, 1000, http.StatusServiceUnavailable, "")
	svc := NewPriceService(&config.PriceAPIConfig{CryptoAPI: srv.URL}, c)

	price, err := svc.GetCryptoPriceWithCurrency(context.Background(), "BTC", "IDR")
	if err != nil {
		t.Fatalf("GetCryptoPriceWithCurrency: %v", err)
	}
	if price.Price != 95 || !price.Stale || !price.FetchedAt.Equal(asOf) {
		t.Errorf("price = %+v, want the stale last-known 95 from %s", price, asOf)
	}

	chart, err := svc.GetCryptoChart(context.Background(), "BTC", "IDR", 7)
	if err != nil {
		t.Fatalf("GetCryptoChart: %v", err)
	}
	if !chart.Stale || len(chart.Data) != 1 {
		t.Errorf("chart = %+v, want the stale last-known chart", chart)
	}
}

func TestCryptoPrice_SuccessIsKeptAsLastKnown(t *testing.T) {
	srv, _ := flakyCoinGecko(t, 0, 0, "")
	c := cache.NewMemoryCache()
	svc := NewPriceService(&config.PriceAPIConfig{CryptoAPI: srv.URL}, c)

	if _, err := svc.GetCryptoPriceWithCurrency(context.Background(), "BTC", "IDR"); err != nil {
		t.Fatalf("GetCryptoPriceWithCurrency: %v", err)
	}
	raw, err := c.Get(context.Background(), "last:crypto:BTC:IDR")
	if err != nil || !strings.Contains(string(raw), `"price":100`) {
		t.Errorf("last-known entry = %s (err %v), want the fetched price", raw, err)
	}
}

func TestDoWithRetry_LongRetryAfterStopsRetrying(t *testing.T) {
	noRetryDelay(t)
	srv, calls := flakyCoinGecko(t, 1000, http.StatusTooManyRequests, "60")
	svc := NewPriceService(&config.PriceAPIConfig{CryptoAPI: srv.URL}, nil)

	if _, err := svc.GetCryptoPriceWithCurrency(context.Background(), "BTC", "IDR"); err == nil {
		t.Fatal("want an error for a 429")
	}
	if calls() != 1 {
		t.Errorf("calls = %d, want 1 (a 60s Retry-After is not waited out)", calls())
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		v    string
		want time.Duration
	}{
		{"", 0},
		{"2", 2 * time.Second},
		{"-5", 0},
		{"soon", 0},
		{now.Add(90 * time.Second).Format(http.TimeFormat), 90 * time.Second},
		{now.Add(-time.Minute).Format(http.TimeFormat), 0},
	}
	for _, tt := range tests {
		if got := parseRetryAfter(tt.v, now); got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestIsIntradayInterval(t *testing.T) {
	for interval, want := range map[string]bool{
		"5m": true, "15m": true, "30m": true, "1h": true, "1d": false, "1wk": false, "1mo": false, "3mo": false,
	} {
		if got := isIntradayInterval(interval); got != want {
			t.Errorf("isIntradayInterval(%q) = %v, want %v", interval, got, want)
		}
	}
}

func TestStockChart_IntradayHasPreviousClose(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, `{"chart":{"result":[{"meta":{"currency":"IDR","chartPreviousClose":4940},`+
			`"timestamp":[1,2],"indicators":{"quote":[{"close":[4950,4980]}]}}],"error":null}}`)
	}))
	t.Cleanup(srv.Close)
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, nil)

	chart, err := svc.GetStockChart(context.Background(), "BBCA.JK", "1d", "5m")
	if err != nil {
		t.Fatalf("GetStockChart: %v", err)
	}
	if gotQuery != "range=1d&interval=5m" {
		t.Errorf("query = %q, want range=1d&interval=5m (not cut from the daily chart)", gotQuery)
	}
	if chart.PreviousClose != 4940 || len(chart.Data) != 2 {
		t.Errorf("chart = %+v, want two points with previous close 4940", chart)
	}

	// Other ranges carry no previous close: Yahoo's is the close before the range, not the day.
	chart, err = svc.GetStockChart(context.Background(), "BBCA.JK", "5d", "15m")
	if err != nil {
		t.Fatalf("GetStockChart: %v", err)
	}
	if chart.PreviousClose != 0 {
		t.Errorf("5d previous close = %v, want 0", chart.PreviousClose)
	}
}

// waitForCache polls c until key holds a value containing want, or fails after a second.
func waitForCache(t *testing.T, c cache.Cache, key, want string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		raw, err := c.Get(context.Background(), key)
		if err == nil && strings.Contains(string(raw), want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("cache %s = %s (err %v), want it to contain %s", key, raw, err, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestStockPriceAndChart_ServeRecentLastKnownWhileRevalidating(t *testing.T) {
	c := cache.NewMemoryCache()
	fetched := time.Now().Add(-2 * time.Minute)
	lastPrice, _ := json.Marshal(port.PriceData{Symbol: "BUMI", Price: 150, Currency: "IDR", Source: "yahoo", FetchedAt: fetched})
	lastChart, _ := json.Marshal(port.ChartResponse{Symbol: "BUMI", Currency: "IDR", Data: []port.ChartDataPoint{{T: 1, P: 150}}, FetchedAt: fetched})
	_ = c.Set(context.Background(), "last:stock:v2:BUMI:IDR", lastPrice, time.Hour)
	_ = c.Set(context.Background(), "last:chart:stock:BUMI:1y:1d", lastChart, time.Hour)
	srv, _ := fakeYahoo(t, "BUMI.JK")
	svc := NewPriceService(&config.PriceAPIConfig{StockAPI: srv.URL}, c)

	price, err := svc.GetStockPriceWithCurrency(context.Background(), "BUMI", "IDR")
	if err != nil {
		t.Fatalf("GetStockPriceWithCurrency: %v", err)
	}
	if price.Price != 150 || price.Stale {
		t.Errorf("price = %+v, want the recent last-known 150, not marked stale", price)
	}
	waitForCache(t, c, "stock:v2:BUMI:IDR", `"price":140`)

	chart, err := svc.GetStockChart(context.Background(), "BUMI", "1mo", "1d")
	if err != nil {
		t.Fatalf("GetStockChart: %v", err)
	}
	if len(chart.Data) != 1 || chart.Stale {
		t.Errorf("chart = %+v, want the recent last-known chart", chart)
	}
	waitForCache(t, c, "chart:stock:BUMI:1y:1d", `"p":140`)

	// Once refreshed, the fresh data is served.
	price, _ = svc.GetStockPriceWithCurrency(context.Background(), "BUMI", "IDR")
	if price.Price != 140 {
		t.Errorf("price after refresh = %v, want 140", price.Price)
	}
}

func TestCryptoPrice_OldLastKnownIsNotServedBeforeFetching(t *testing.T) {
	c := cache.NewMemoryCache()
	last, _ := json.Marshal(port.PriceData{Symbol: "BTC", Price: 95, Currency: "IDR", Source: "coingecko", FetchedAt: time.Now().Add(-time.Hour)})
	_ = c.Set(context.Background(), "last:crypto:BTC:IDR", last, time.Hour)
	srv, calls := flakyCoinGecko(t, 0, 0, "")
	svc := NewPriceService(&config.PriceAPIConfig{CryptoAPI: srv.URL}, c)

	price, err := svc.GetCryptoPriceWithCurrency(context.Background(), "BTC", "IDR")
	if err != nil {
		t.Fatalf("GetCryptoPriceWithCurrency: %v", err)
	}
	if price.Price != 100 || calls() != 1 {
		t.Errorf("price = %v after %d calls, want the fetched 100 (an hour-old price is not served)", price.Price, calls())
	}
}
