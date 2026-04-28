package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elazarl/goproxy"
)

// --- Pure functions ---

func TestFormat(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0.0B"},
		{1, "1.0B"},
		{512, "512.0B"},
		{1023, "1023.0B"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1048576, "1.0M"},
		{1073741824, "1.0G"},
		{-512, "-512.0B"},
		{-1024, "-1.0K"},
	}
	for _, tt := range tests {
		got := format(tt.input)
		if got != tt.expected {
			t.Errorf("format(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMinutesAgo(t *testing.T) {
	now := time.Now().Unix()
	got := minutesAgo(now - 120) // 2 minutes ago
	if got < 2 || got > 3 {
		t.Errorf("minutesAgo(now-120) = %d, want ~2", got)
	}
	got = minutesAgo(now - 3600) // 1 hour ago
	if got < 59 || got > 61 {
		t.Errorf("minutesAgo(now-3600) = %d, want ~60", got)
	}
}

func TestRandRange(t *testing.T) {
	r := [2]float64{0.1, 0.6}
	for i := 0; i < 1000; i++ {
		v := randRange(r)
		if v < r[0] || v > r[1] {
			t.Errorf("randRange(%v) = %f, out of range", r, v)
		}
	}
}

// --- Config loading ---

func TestLoadConfigDefault(t *testing.T) {
	config := loadConfig("/nonexistent/path/config.yaml")
	if _, ok := config["default"]; !ok {
		t.Fatal("expected 'default' key in config")
	}
	d := config["default"]
	if d.Uploaded != [2]float64{0.1, 0.6} {
		t.Errorf("default Uploaded = %v, want [0.1 0.6]", d.Uploaded)
	}
	if d.Downloaded != [2]float64{0, 0.07} {
		t.Errorf("default Downloaded = %v, want [0 0.07]", d.Downloaded)
	}
	if d.Speed != 51200 {
		t.Errorf("default Speed = %d, want 51200", d.Speed)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.yaml")
	yaml := `
"example.com":
  uploaded: [0.5, 1.0]
  downloaded: [0.1, 0.2]
  speed: 102400
  port: 6881
  peerid: "ABCDEF"
  useragent: "TestAgent/1.0"
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	config := loadConfig(path)
	if _, ok := config["default"]; !ok {
		t.Fatal("expected 'default' key in config")
	}
	host, ok := config["example.com"]
	if !ok {
		t.Fatal("expected 'example.com' key in config")
	}
	if host.Uploaded != [2]float64{0.5, 1.0} {
		t.Errorf("Uploaded = %v", host.Uploaded)
	}
	if host.Speed != 102400 {
		t.Errorf("Speed = %d", host.Speed)
	}
	if host.Port != 6881 {
		t.Errorf("Port = %d", host.Port)
	}
	if host.PeerId != "ABCDEF" {
		t.Errorf("PeerId = %q", host.PeerId)
	}
	if host.UserAgent != "TestAgent/1.0" {
		t.Errorf("UserAgent = %q", host.UserAgent)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(": invalid yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	// Should not panic; should still return default config
	config := loadConfig(path)
	if _, ok := config["default"]; !ok {
		t.Fatal("expected 'default' key even with invalid config")
	}
}

// --- Database operations ---

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInitDB(t *testing.T) {
	db := openTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
	// Should be idempotent
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAndLoadReqInfo(t *testing.T) {
	db := openTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
	ri := ReqInfo{
		InfoHash:       "abc123def456",
		Host:           "tracker.example.com",
		ReportUploaded: 5000,
		Uploaded:       3000,
		Downloaded:     2000,
		Epoch:          1700000000,
		Incomplete:     5,
	}
	if err := saveReqInfo(db, ri); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadReqInfo(db, ri.InfoHash)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.InfoHash != ri.InfoHash {
		t.Errorf("InfoHash = %q, want %q", loaded.InfoHash, ri.InfoHash)
	}
	if loaded.Host != ri.Host {
		t.Errorf("Host = %q, want %q", loaded.Host, ri.Host)
	}
	if loaded.ReportUploaded != ri.ReportUploaded {
		t.Errorf("ReportUploaded = %d, want %d", loaded.ReportUploaded, ri.ReportUploaded)
	}
	if loaded.Uploaded != ri.Uploaded {
		t.Errorf("Uploaded = %d, want %d", loaded.Uploaded, ri.Uploaded)
	}
	if loaded.Downloaded != ri.Downloaded {
		t.Errorf("Downloaded = %d, want %d", loaded.Downloaded, ri.Downloaded)
	}
	if loaded.Epoch != ri.Epoch {
		t.Errorf("Epoch = %d, want %d", loaded.Epoch, ri.Epoch)
	}
	if loaded.Incomplete != ri.Incomplete {
		t.Errorf("Incomplete = %d, want %d", loaded.Incomplete, ri.Incomplete)
	}
}

func TestLoadReqInfoNotFound(t *testing.T) {
	db := openTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
	_, err := loadReqInfo(db, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSaveReqInfoReplace(t *testing.T) {
	db := openTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
	ri := ReqInfo{
		InfoHash:       "hash1",
		Host:           "tracker.example.com",
		ReportUploaded: 100,
		Uploaded:       100,
		Downloaded:     0,
		Epoch:          1700000000,
		Incomplete:     3,
	}
	if err := saveReqInfo(db, ri); err != nil {
		t.Fatal(err)
	}
	// Replace with new values
	ri.ReportUploaded = 200
	ri.Uploaded = 150
	if err := saveReqInfo(db, ri); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadReqInfo(db, "hash1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ReportUploaded != 200 {
		t.Errorf("ReportUploaded = %d, want 200", loaded.ReportUploaded)
	}
}

func TestLoadAllReqInfo(t *testing.T) {
	db := openTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		ri := ReqInfo{
			InfoHash:       fmt.Sprintf("hash%d", i),
			Host:           "tracker.example.com",
			ReportUploaded: int64(i * 100),
			Uploaded:       int64(i * 50),
			Downloaded:     int64(i * 10),
			Epoch:          1700000000 + int64(i),
			Incomplete:     int64(i),
		}
		if err := saveReqInfo(db, ri); err != nil {
			t.Fatal(err)
		}
	}
	all, err := loadAllReqInfo(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("len(all) = %d, want 3", len(all))
	}
}

func TestLoadAllReqInfoEmpty(t *testing.T) {
	db := openTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
	all, err := loadAllReqInfo(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Errorf("len(all) = %d, want 0", len(all))
	}
}

func TestSaveIncomplete(t *testing.T) {
	db := openTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}
	ri := ReqInfo{
		InfoHash:       "hash_incomplete",
		Host:           "tracker.example.com",
		ReportUploaded: 0,
		Uploaded:       0,
		Downloaded:     0,
		Epoch:          1700000000,
		Incomplete:     10,
	}
	if err := saveReqInfo(db, ri); err != nil {
		t.Fatal(err)
	}
	if err := saveIncomplete(db, ri.InfoHash, 7); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadReqInfo(db, ri.InfoHash)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Incomplete != 7 {
		t.Errorf("Incomplete = %d, want 7", loaded.Incomplete)
	}
}

// --- Certificate generation ---

func TestLoadCA(t *testing.T) {
	loadCA()
	if goproxy.GoproxyCa.Leaf == nil {
		t.Fatal("CA Leaf certificate not parsed")
	}
	if goproxy.GoproxyCa.PrivateKey == nil {
		t.Fatal("CA PrivateKey is nil")
	}
}

func TestSignCert(t *testing.T) {
	loadCA()
	cert, err := signCert(&goproxy.GoproxyCa, "tracker.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.Certificate) != 2 {
		t.Fatalf("expected 2 certificates in chain, got %d", len(cert.Certificate))
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if leaf.Subject.CommonName != "tracker.example.com" {
		t.Errorf("CommonName = %q, want %q", leaf.Subject.CommonName, "tracker.example.com")
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != "tracker.example.com" {
		t.Errorf("DNSNames = %v, want [tracker.example.com]", leaf.DNSNames)
	}
	if leaf.PublicKeyAlgorithm != x509.ECDSA {
		t.Errorf("PublicKeyAlgorithm = %v, want ECDSA", leaf.PublicKeyAlgorithm)
	}
	// Verify the cert is signed by the CA
	if err := leaf.CheckSignatureFrom(goproxy.GoproxyCa.Leaf); err != nil {
		t.Errorf("cert not signed by CA: %v", err)
	}
	// Verify key type
	if _, ok := cert.PrivateKey.(*ecdsa.PrivateKey); !ok {
		t.Error("PrivateKey is not *ecdsa.PrivateKey")
	}
	if cert.PrivateKey.(*ecdsa.PrivateKey).Curve != elliptic.P256() {
		t.Error("PrivateKey curve is not P-256")
	}
}

// --- Query parsing ---

// newClientRequest creates an *http.Request suitable for use with http.Client.
// Unlike httptest.NewRequest, it does not set RequestURI which the client transport rejects.
func newClientRequest(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	return req
}

func TestQueryInt64(t *testing.T) {
	// Test valid query parameter parsing via an actual proxy OnRequest handler.
	proxy := goproxy.NewProxyHttpServer()
	var parsedUploaded, parsedDownloaded int64
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		parsedUploaded = queryInt64(req, ctx, "uploaded")
		parsedDownloaded = queryInt64(req, ctx, "downloaded")
		return req, nil
	})
	// Mock upstream so no real network access needed.
	mockUp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockUp.Close()
	proxy.Tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, mockUp.Listener.Addr().String())
	}

	ts := httptest.NewServer(proxy)
	defer ts.Close()
	proxyURL, _ := url.Parse(ts.URL)

	req, _ := http.NewRequest("GET", "http://tracker.example.com/announce?uploaded=12345&downloaded=678&missing=abc", nil)
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if parsedUploaded != 12345 {
		t.Errorf("uploaded = %d, want 12345", parsedUploaded)
	}
	if parsedDownloaded != 678 {
		t.Errorf("downloaded = %d, want 678", parsedDownloaded)
	}
}

// --- Proxy integration tests ---

// proxyTestEnv holds the test proxy and mock upstream servers.
type proxyTestEnv struct {
	proxy   *httptest.Server
	db      *sql.DB
	// upstream is a mock upstream HTTP server. Outbound connections from the
	// proxy are redirected here, so no real network access is needed.
	upstream    *httptest.Server
	// lastReq is the last request received by the mock upstream.
	lastReq     *http.Request
}

// setupProxy creates a goproxy instance with a mock upstream. The mock upstream
// responds with a minimal bencoded tracker response (including incomplete=42)
// so the response handler can extract it. The proxy's Tr.DialContext is
// overridden to redirect all outbound TCP connections to the mock upstream.
func setupProxy(t *testing.T) *proxyTestEnv {
	t.Helper()
	loadCA()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}

	env := &proxyTestEnv{db: db}

	// Mock upstream: records the last request and returns a tracker response.
	env.upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env.lastReq = r
		// Return a minimal bencoded tracker response with incomplete field.
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("d8:completei0e10:incompletei42ee"))
	}))
	t.Cleanup(func() { env.upstream.Close() })

	config := map[string]Setting{
		"default": {
			Uploaded:    [2]float64{0.1, 0.6},
			Downloaded:  [2]float64{0, 0.07},
			PercentMin:  0.2,
			PercentMax:  0.5,
			PercentStep: 0.02,
			Speed:       51200,
			Port:        0,
			PeerId:      "",
			UserAgent:   "",
		},
		"tracker.override.com": {
			Uploaded:    [2]float64{0.5, 1.0},
			Downloaded:  [2]float64{0.1, 0.2},
			PercentMin:  0.0,
			PercentMax:  0.0,
			PercentStep: 0.02,
			Speed:       0,
			Port:        12345,
			PeerId:      "ABCDEF",
			UserAgent:   "OverrideAgent/1.0",
		},
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	// Redirect all outbound connections to the mock upstream.
	proxy.Tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, env.upstream.Listener.Addr().String())
	}

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		var reqInfo ReqInfo
		reqInfo.Host = req.URL.Hostname()
		if sni, ok := ctx.UserData.(string); ok && net.ParseIP(reqInfo.Host) != nil {
			if port := req.URL.Port(); port != "" {
				req.URL.Host = net.JoinHostPort(sni, port)
			} else {
				req.URL.Host = sni
			}
			reqInfo.Host = sni
		}
		if reqInfo.Host == "127.0.0.1" || !strings.Contains(strings.Trim(reqInfo.Host, "."), ".") {
			return nil, &http.Response{
				StatusCode: http.StatusBadGateway,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       io.NopCloser(strings.NewReader("Rejected by proxy")),
				Request:    req,
				Header:     make(http.Header),
			}
		}
		query := req.URL.Query()
		reqInfo.InfoHash = query.Get("info_hash")
		reqInfo.Uploaded = queryInt64(req, ctx, "uploaded")
		reqInfo.Downloaded = queryInt64(req, ctx, "downloaded")
		if reqInfo.InfoHash == "" || reqInfo.Uploaded < 0 || reqInfo.Downloaded < 0 {
			return req, nil
		}
		setting := config["default"]
		if hostSetting, ok := config[reqInfo.Host]; ok {
			setting = hostSetting
		}
		if setting.PeerId != "" {
			req.URL.RawQuery = peerIdMatcher.ReplaceAllString(req.URL.RawQuery,
				fmt.Sprintf("${1}peer_id=-%s-", setting.PeerId))
		}
		if setting.Port > 0 && setting.Port < 65536 {
			req.URL.RawQuery = portMatcher.ReplaceAllString(req.URL.RawQuery,
				fmt.Sprintf("${1}port=%d${2}", setting.Port))
		}
		if setting.UserAgent != "" {
			req.Header.Set("User-Agent", setting.UserAgent)
		}
		reqInfo.Epoch = time.Now().Unix()
		reqInfo.ReportUploaded = reqInfo.Uploaded
		reqInfo.Incomplete = incompleteUnknown
		if prevReqInfo, err := loadReqInfo(db, reqInfo.InfoHash); err != nil {
			// not in DB
		} else {
			if query.Get("event") != "started" {
				deltaUploaded := reqInfo.Uploaded - prevReqInfo.Uploaded
				deltaDownloaded := reqInfo.Downloaded - prevReqInfo.Downloaded
				deltaEpoch := reqInfo.Epoch - prevReqInfo.Epoch
				if deltaUploaded >= 0 && deltaDownloaded >= 0 && deltaEpoch <= maxDeltaSeconds {
					reqInfo.ReportUploaded = prevReqInfo.ReportUploaded
					reqInfo.ReportUploaded += deltaUploaded
					if prevReqInfo.Incomplete >= 1 {
						reqInfo.ReportUploaded += int64(float64(deltaUploaded) * randRange(setting.Uploaded))
						reqInfo.ReportUploaded += int64(float64(deltaDownloaded) * randRange(setting.Downloaded))
						percent := math.Min(setting.PercentMin+float64(prevReqInfo.Incomplete-1)*setting.PercentStep, setting.PercentMax)
						if rand.Float64() < percent {
							reqInfo.ReportUploaded += int64(float64(deltaEpoch*setting.Speed) * rand.Float64())
						}
					}
					req.URL.RawQuery = uploadedMatcher.ReplaceAllString(req.URL.RawQuery,
						fmt.Sprintf("${1}uploaded=%d${2}", reqInfo.ReportUploaded))
				}
			}
		}
		if err := saveReqInfo(db, reqInfo); err != nil {
			ctx.Warnf("%s", err)
		}
		return req, nil
	})

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp != nil && resp.StatusCode == http.StatusOK {
			if bodyBytes, err := io.ReadAll(resp.Body); err != nil {
				ctx.Warnf("%s", err)
			} else {
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				if match := incompleteMatcher.FindSubmatch(bodyBytes); match != nil {
					query := ctx.Req.URL.Query()
					infoHash := query.Get("info_hash")
					incomplete, _ := strconv.ParseInt(string(match[1]), 10, 64)
					if queryInt64(ctx.Req, ctx, "left") > 0 || query.Get("event") == "completed" {
						incomplete--
					}
					if err := saveIncomplete(db, infoHash, incomplete); err != nil {
						ctx.Warnf("%s", err)
					}
				}
			}
		}
		return resp
	})

	env.proxy = httptest.NewServer(proxy)
	t.Cleanup(func() { env.proxy.Close() })
	return env
}

func TestProxyRejectLocalhost(t *testing.T) {
	env := setupProxy(t)
	proxyURL, _ := url.Parse(env.proxy.URL)

	req := newClientRequest("GET", "http://127.0.0.1/announce?info_hash=abc&uploaded=100&downloaded=50", nil)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
}

func TestProxyRejectNonFQDN(t *testing.T) {
	env := setupProxy(t)
	proxyURL, _ := url.Parse(env.proxy.URL)

	req := newClientRequest("GET", "http://localhost/announce?info_hash=abc&uploaded=100&downloaded=50", nil)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
}

func TestProxyPassThroughMissingFields(t *testing.T) {
	env := setupProxy(t)
	proxyURL, _ := url.Parse(env.proxy.URL)

	req := newClientRequest("GET", "http://tracker.example.com/announce?uploaded=100&downloaded=50", nil)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// Request without info_hash passes through to mock upstream.
	// The mock upstream always returns 200, so we should get a 200 back.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestProxyPeerIdOverride(t *testing.T) {
	env := setupProxy(t)
	proxyURL, _ := url.Parse(env.proxy.URL)

	req := newClientRequest("GET", "http://tracker.override.com/announce?info_hash=abc123&uploaded=100&downloaded=50&peer_id=-ZX1234-&port=9999", nil)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Verify the upstream received the modified request.
	if env.lastReq == nil {
		t.Fatal("mock upstream did not receive a request")
	}
	upQuery := env.lastReq.URL.Query()
	if got := upQuery.Get("peer_id"); got != "-ABCDEF-" {
		t.Errorf("upstream peer_id = %q, want %q", got, "-ABCDEF-")
	}
	if got := upQuery.Get("port"); got != "12345" {
		t.Errorf("upstream port = %q, want %q", got, "12345")
	}
}

func TestProxyUserAgentOverride(t *testing.T) {
	env := setupProxy(t)
	proxyURL, _ := url.Parse(env.proxy.URL)

	req := newClientRequest("GET", "http://tracker.override.com/announce?info_hash=abc123&uploaded=100&downloaded=50", nil)
	req.Header.Set("User-Agent", "OriginalAgent/1.0")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Verify the upstream received the overridden User-Agent.
	if env.lastReq == nil {
		t.Fatal("mock upstream did not receive a request")
	}
	if got := env.lastReq.Header.Get("User-Agent"); got != "OverrideAgent/1.0" {
		t.Errorf("upstream User-Agent = %q, want %q", got, "OverrideAgent/1.0")
	}
}

func TestProxyRatioCalculation(t *testing.T) {
	// This test validates the core ratio manipulation logic by making sequential
	// requests through the proxy and checking that the uploaded parameter is inflated.
	env := setupProxy(t)
	proxyURL, _ := url.Parse(env.proxy.URL)

	// Pre-seed the DB with initial torrent state.
	infoHash := "abc123delta"
	seedInfo := ReqInfo{
		InfoHash:       infoHash,
		Host:           "tracker.example.com",
		ReportUploaded: 1000,
		Uploaded:       1000,
		Downloaded:     500,
		Epoch:          time.Now().Unix() - 60,
		Incomplete:     10,
	}
	if err := saveReqInfo(env.db, seedInfo); err != nil {
		t.Fatal(err)
	}

	req := newClientRequest("GET", fmt.Sprintf(
		"http://tracker.example.com/announce?info_hash=%s&uploaded=2000&downloaded=700&left=0",
		url.QueryEscape(infoHash),
	), nil)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	updated, err := loadReqInfo(env.db, infoHash)
	if err != nil {
		t.Fatal(err)
	}
	// ReportUploaded should be higher than the original 1000 due to ratio inflation.
	if updated.ReportUploaded <= 1000 {
		t.Errorf("ReportUploaded = %d, want > 1000 (ratio should inflate upload)", updated.ReportUploaded)
	}
	if updated.Uploaded != 2000 {
		t.Errorf("Uploaded = %d, want 2000", updated.Uploaded)
	}
	t.Logf("ReportUploaded inflated from 1000 to %d", updated.ReportUploaded)
}

// --- Response handler / incomplete extraction ---

func TestIncompleteMatcher(t *testing.T) {
	tests := []struct {
		body       string
		expected   string
		shouldMatch bool
	}{
		{"d8:completei10:incompletei42ee", "42", true},
		{"10:incompletei0e", "0", true},
		{"10:incompletei12345e", "12345", true},
		{"no match here", "", false},
	}
	for _, tt := range tests {
		match := incompleteMatcher.FindSubmatch([]byte(tt.body))
		if tt.shouldMatch {
			if match == nil {
				t.Errorf("expected match for %q", tt.body)
			} else if string(match[1]) != tt.expected {
				t.Errorf("incomplete = %q, want %q", string(match[1]), tt.expected)
			}
		} else {
			if match != nil {
				t.Errorf("expected no match for %q, got %q", tt.body, string(match[1]))
			}
		}
	}
}

func TestPeerIdMatcher(t *testing.T) {
	tests := []struct {
		input     string
		replaceID string
		expected  string
	}{
		{"peer_id=-ZX1234-&uploaded=100", "ABCDEF", "peer_id=-ABCDEF-&uploaded=100"},
		{"uploaded=100&peer_id=-ZX1234-", "ABCDEF", "uploaded=100&peer_id=-ABCDEF-"},
		{"&peer_id=-ZX1234-", "ABCDEF", "&peer_id=-ABCDEF-"},
	}
	for _, tt := range tests {
		replacement := fmt.Sprintf("${1}peer_id=-%s-", tt.replaceID)
		got := peerIdMatcher.ReplaceAllString(tt.input, replacement)
		if got != tt.expected {
			t.Errorf("ReplaceAllString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestPortMatcher(t *testing.T) {
	tests := []struct {
		input    string
		replace  int64
		expected string
	}{
		{"port=6881&uploaded=100", 12345, "port=12345&uploaded=100"},
		{"uploaded=100&port=6881", 12345, "uploaded=100&port=12345"},
	}
	for _, tt := range tests {
		got := portMatcher.ReplaceAllString(tt.input,
			fmt.Sprintf("${1}port=%d${2}", tt.replace))
		if got != tt.expected {
			t.Errorf("ReplaceAllString(%q, %d) = %q, want %q", tt.input, tt.replace, got, tt.expected)
		}
	}
}

func TestUploadedMatcher(t *testing.T) {
	got := uploadedMatcher.ReplaceAllString("uploaded=100&downloaded=50",
		"${1}uploaded=500${2}")
	if got != "uploaded=500&downloaded=50" {
		t.Errorf("ReplaceAllString = %q", got)
	}
}

// --- Dial guard ---

// makeDialGuard returns the same DialContext function used in main.go.
func makeDialGuard() func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		tcpaddr, err := net.ResolveTCPAddr(network, addr)
		if err != nil {
			return nil, err
		}
		if tcpaddr.IP.IsPrivate() {
			return nil, fmt.Errorf("private IP blocked: %s", tcpaddr.IP)
		}
		var d net.Dialer
		return d.DialContext(ctx, network, tcpaddr.String())
	}
}

func TestDialGuardBlocksPrivate(t *testing.T) {
	guard := makeDialGuard()
	privateIPs := []string{
		"10.0.0.1:80",
		"172.16.0.1:80",
		"192.168.1.1:80",
		"10.255.255.255:443",
		"172.31.255.255:8080",
		"192.168.255.255:12345",
	}
	for _, addr := range privateIPs {
		_, err := guard(context.Background(), "tcp", addr)
		if err == nil {
			t.Errorf("expected error for private IP %s", addr)
		} else if !strings.Contains(err.Error(), "private IP blocked") {
			t.Errorf("expected 'private IP blocked' error for %s, got: %v", addr, err)
		}
	}
}

func TestDialGuardAllowsPublic(t *testing.T) {
	guard := makeDialGuard()
	// 8.8.8.8 is a public IP — the guard should resolve it and attempt to connect
	// (connection will likely timeout/fail, but should NOT be the "private IP blocked" error)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := guard(ctx, "tcp", "8.8.8.8:80")
	if err != nil {
		if strings.Contains(err.Error(), "private IP blocked") {
			t.Errorf("public IP unexpectedly blocked: %v", err)
		}
	}
}
