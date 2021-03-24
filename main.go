package main

import (
	"bytes"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"github.com/abourget/goproxy"
	"github.com/kr/pretty"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v2"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed static templates
var embedFS embed.FS

// Arg ...
type Arg struct {
	Addr    *string
	Conf    *string
	DbPath  *string
	Verbose *bool
	Version *bool
}

// Setting ...
type Setting struct {
	Uploaded    [2]float64
	Downloaded  [2]float64
	PercentMin  float64
	PercentMax  float64
	PercentStep float64
	Speed       int64
	Port        int64
}

// ReqInfo ...
type ReqInfo struct {
	InfoHash       string
	Host           string
	ReportUploaded int64
	Uploaded       int64
	Downloaded     int64
	Epoch          int64
	Incomplete     int64
}

var (
	incompleteMatcher = regexp.MustCompile(`10:incompletei(\d+)e`)
	portMatcher       = regexp.MustCompile(`(^|&)port=\d+(&|$)`)
	uploadedMatcher   = regexp.MustCompile(`(^|&)uploaded=\d+(&|$)`)
	// Version info
	Version = "v0.6"
)

func loadCA() {
	goproxy.CA_CERT = []byte(`-----BEGIN CERTIFICATE-----
MIIF9DCCA9ygAwIBAgIJAODqYUwoVjJkMA0GCSqGSIb3DQEBCwUAMIGOMQswCQYD
VQQGEwJJTDEPMA0GA1UECAwGQ2VudGVyMQwwCgYDVQQHDANMb2QxEDAOBgNVBAoM
B0dvUHJveHkxEDAOBgNVBAsMB0dvUHJveHkxGjAYBgNVBAMMEWdvcHJveHkuZ2l0
aHViLmlvMSAwHgYJKoZIhvcNAQkBFhFlbGF6YXJsQGdtYWlsLmNvbTAeFw0xNzA0
MDUyMDAwMTBaFw0zNzAzMzEyMDAwMTBaMIGOMQswCQYDVQQGEwJJTDEPMA0GA1UE
CAwGQ2VudGVyMQwwCgYDVQQHDANMb2QxEDAOBgNVBAoMB0dvUHJveHkxEDAOBgNV
BAsMB0dvUHJveHkxGjAYBgNVBAMMEWdvcHJveHkuZ2l0aHViLmlvMSAwHgYJKoZI
hvcNAQkBFhFlbGF6YXJsQGdtYWlsLmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBAJ4Qy+H6hhoY1s0QRcvIhxrjSHaO/RbaFj3rwqcnpOgFq07gRdI9
3c0TFKQJHpgv6feLRhEvX/YllFYu4J35lM9ZcYY4qlKFuStcX8Jm8fqpgtmAMBzP
sqtqDi8M9RQGKENzU9IFOnCV7SAeh45scMuI3wz8wrjBcH7zquHkvqUSYZz035t9
V6WTrHyTEvT4w+lFOVN2bA/6DAIxrjBiF6DhoJqnha0SZtDfv77XpwGG3EhA/qoh
hiYrDruYK7zJdESQL44LwzMPupVigqalfv+YHfQjbhT951IVurW2NJgRyBE62dLr
lHYdtT9tCTCrd+KJNMJ+jp9hAjdIu1Br/kifU4F4+4ZLMR9Ueji0GkkPKsYdyMnq
j0p0PogyvP1l4qmboPImMYtaoFuYmMYlebgC9LN10bL91K4+jLt0I1YntEzrqgJo
WsJztYDw543NzSy5W+/cq4XRYgtq1b0RWwuUiswezmMoeyHZ8BQJe2xMjAOllASD
fqa8OK3WABHJpy4zUrnUBiMuPITzD/FuDx4C5IwwlC68gHAZblNqpBZCX0nFCtKj
YOcI2So5HbQ2OC8QF+zGVuduHUSok4hSy2BBfZ1pfvziqBeetWJwFvapGB44nIHh
WKNKvqOxLNIy7e+TGRiWOomrAWM18VSR9LZbBxpJK7PLSzWqYJYTRCZHAgMBAAGj
UzBRMB0GA1UdDgQWBBR4uDD9Y6x7iUoHO+32ioOcw1ICZTAfBgNVHSMEGDAWgBR4
uDD9Y6x7iUoHO+32ioOcw1ICZTAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEB
CwUAA4ICAQAaCEupzGGqcdh+L7BzhX7zyd7yzAKUoLxFrxaZY34Xyj3lcx1XoK6F
AqsH2JM25GixgadzhNt92JP7vzoWeHZtLfstrPS638Y1zZi6toy4E49viYjFk5J0
C6ZcFC04VYWWx6z0HwJuAS08tZ37JuFXpJGfXJOjZCQyxse0Lg0tuKLMeXDCk2Y3
Ba0noeuNyHRoWXXPyiUoeApkVCU5gIsyiJSWOjhJ5hpJG06rQNfNYexgKrrraEin
o0jmEMtJMx5TtD83hSnLCnFGBBq5lkE7jgXME1KsbIE3lJZzRX1mQwUK8CJDYxye
i6M/dzSvy0SsPvz8fTAlprXRtWWtJQmxgWENp3Dv+0Pmux/l+ilk7KA4sMXGhsfr
bvTOeWl1/uoFTPYiWR/ww7QEPLq23yDFY04Q7Un0qjIk8ExvaY8lCkXMgc8i7sGY
VfvOYb0zm67EfAQl3TW8Ky5fl5CcxpVCD360Bzi6hwjYixa3qEeBggOixFQBFWft
8wrkKTHpOQXjn4sDPtet8imm9UYEtzWrFX6T9MFYkBR0/yye0FIh9+YPiTA6WB86
NCNwK5Yl6HuvF97CIH5CdgO+5C7KifUtqTOL8pQKbNwy0S3sNYvB+njGvRpR7pKV
BUnFpB/Atptqr4CUlTXrc5IPLAqAfmwk5IKcwy3EXUbruf9Dwz69YA==
-----END CERTIFICATE-----`)
	goproxy.CA_KEY = []byte(`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`)
	if err := goproxy.LoadDefaultConfig(); err != nil {
		log.Print(err)
	}
}

func parseArg() Arg {
	var arg Arg
	defaultConfPath := ""
	defaultDbPath := ":memory:"
	if usr, err := user.Current(); err != nil {
		log.Print(err)
	} else {
		defaultConfPath = filepath.Join(usr.HomeDir, ".torrent-ratio.yaml")
		defaultDbPath = filepath.Join(usr.HomeDir, ".torrent-ratio.db")
	}
	arg.Addr = flag.String("addr", "127.0.0.1:8082", "proxy listen address")
	arg.Conf = flag.String("conf", defaultConfPath, "config file")
	arg.DbPath = flag.String("db", defaultDbPath, "database path")
	arg.Verbose = flag.Bool("v", false, "enable verbose logging")
	arg.Version = flag.Bool("V", false, "print version")
	flag.Parse()
	return arg
}

func loadConfig(file string) map[string]Setting {
	config := map[string]Setting{
		"default": {
			Uploaded:    [2]float64{0.1, 0.6},
			Downloaded:  [2]float64{0, 0.07},
			PercentMin:  0.2,
			PercentMax:  0.5,
			PercentStep: 0.02,
			Speed:       51200,
			Port:        0,
		},
	}
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Print(err)
		}
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Print(err)
	}
	return config
}

func queryInt64(ctx *goproxy.ProxyCtx, key string) int64 {
	query := ctx.Req.URL.Query()
	i, err := strconv.ParseInt(query.Get(key), 10, 64)
	if err != nil {
		ctx.Warnf("%s", err)
		i = -1
	}
	return i
}

func randRange(r [2]float64) float64 {
	return r[0] + rand.Float64()*(r[1]-r[0])
}

func initDB(db *sql.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS torrent (
		InfoHash TEXT PRIMARY KEY,
		Host TEXT,
		ReportUploaded INTEGER,
		Uploaded INTEGER,
		Downloaded INTEGER,
		Epoch INTEGER,
		Incomplete INTEGER
	)`
	_, err := db.Exec(sql)
	return err
}

func saveReqInfo(db *sql.DB, reqInfo ReqInfo) error {
	sql := `
	REPLACE INTO torrent (
		InfoHash,
		Host,
		ReportUploaded,
		Uploaded,
		Downloaded,
		Epoch,
		Incomplete
	) values (?, ?, ?, ?, ?, ?, ?)
	`

	if _, err := db.Exec(
		sql,
		reqInfo.InfoHash,
		reqInfo.Host,
		reqInfo.ReportUploaded,
		reqInfo.Uploaded,
		reqInfo.Downloaded,
		reqInfo.Epoch,
		reqInfo.Incomplete,
	); err != nil {
		return err
	}
	return nil
}

func saveIncomplete(db *sql.DB, infoHash string, incomplete int64) error {
	sql := `UPDATE torrent SET Incomplete=? WHERE InfoHash=?`
	_, err := db.Exec(sql, incomplete, infoHash)
	return err
}

func loadReqInfo(db *sql.DB, infoHash string) (ReqInfo, error) {
	sql := `SELECT * FROM torrent WHERE InfoHash=?`
	var reqInfo ReqInfo
	err := db.QueryRow(sql, infoHash).Scan(
		&reqInfo.InfoHash,
		&reqInfo.Host,
		&reqInfo.ReportUploaded,
		&reqInfo.Uploaded,
		&reqInfo.Downloaded,
		&reqInfo.Epoch,
		&reqInfo.Incomplete,
	)
	return reqInfo, err
}

func loadAllReqInfo(db *sql.DB) ([]ReqInfo, error) {
	sql := `SELECT * FROM torrent`
	var result []ReqInfo
	rows, err := db.Query(sql)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var reqInfo ReqInfo
		err := rows.Scan(
			&reqInfo.InfoHash,
			&reqInfo.Host,
			&reqInfo.ReportUploaded,
			&reqInfo.Uploaded,
			&reqInfo.Downloaded,
			&reqInfo.Epoch,
			&reqInfo.Incomplete,
		)
		if err != nil {
			return result, err
		}
		result = append(result, reqInfo)
	}
	return result, err
}

func cleanup(db *sql.DB) {
	sql := `DELETE FROM torrent WHERE Epoch < ?`
	for {
		result, err := db.Exec(sql, time.Now().Unix()-86400)
		if err != nil {
			log.Print(err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			log.Print(err)
		}
		log.Printf("CLEANUP: %d", count)
		time.Sleep(24 * time.Hour)
	}
}

func format(num int64) string {
	float := float64(num)
	for _, unit := range []string{"B", "K", "M", "G"} {
		if math.Abs(float) < float64(1024) {
			return fmt.Sprintf("%3.1f%s", float, unit)
		}
		float /= float64(1024)
	}
	return fmt.Sprintf("%3.1f%s", float, "T")
}

func ago(epoch int64) int64 {
	return (time.Now().Unix() - epoch) / 60
}

func main() {
	isTerminal := terminal.IsTerminal(int(os.Stdout.Fd()))
	if !isTerminal {
		log.SetFlags(0)
	}

	arg := parseArg()

	if *arg.Version {
		log.SetFlags(0)
		log.Fatal(Version)
	}

	db, err := sql.Open("sqlite3", *arg.DbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	config := loadConfig(*arg.Conf)
	if *arg.Verbose {
		log.Printf("CONFIG: %# v", pretty.Formatter(config))
	}

	loadCA()
	rand.Seed(time.Now().UnixNano())

	proxy := goproxy.NewProxyHttpServer()
	proxy.HandleConnect(goproxy.AlwaysMitm)

	if !isTerminal {
		proxy.Logger.SetFlags(0)
	}

	templates, err := template.New("").Funcs(template.FuncMap{
		"format": format,
		"ago":    ago,
	}).ParseFS(embedFS, "templates/*")
	if err != nil {
		log.Fatal(err)
	}

	lastModified := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
	    name := strings.TrimPrefix(r.URL.Path, "/")
		file, err := embedFS.Open(name)
		if err != nil {
			http.NotFound(w, r)
		} else {
			defer file.Close()
			http.ServeContent(w, r, name, lastModified, file.(io.ReadSeeker))
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", "/index.html":
			reqInfos, err := loadAllReqInfo(db)
			if err != nil {
				log.Print(err)
			}
			if strings.HasPrefix(r.UserAgent(), "Mozilla/") {
				if err := templates.ExecuteTemplate(w, "index.html", reqInfos); err != nil {
					log.Print(err)
				}
			} else {
				fmt.Fprintln(w, "hash     up            down   ann. inc. host")
				for _, reqInfo := range reqInfos {
					fmt.Fprintf(w, "%.4x %-6s %-6s %-6s %-4d %-4d %s\n",
						reqInfo.InfoHash,
						format(reqInfo.ReportUploaded),
						format(reqInfo.Uploaded),
						format(reqInfo.Downloaded),
						ago(reqInfo.Epoch),
						reqInfo.Incomplete,
						reqInfo.Host)
				}
			}
		default:
			http.NotFound(w, r)
		}
	})
	proxy.NonProxyHandler = mux

	proxy.HandleRequestFunc(func(ctx *goproxy.ProxyCtx) goproxy.Next {
		req := ctx.Req
		var reqInfo ReqInfo
		query := req.URL.Query()
		reqInfo.InfoHash = query.Get("info_hash")
		reqInfo.Uploaded = queryInt64(ctx, "uploaded")
		reqInfo.Downloaded = queryInt64(ctx, "downloaded")
		if reqInfo.InfoHash == "" || reqInfo.Uploaded < 0 || reqInfo.Downloaded < 0 {
			return goproxy.NEXT
		}
		reqInfo.Host = req.URL.Hostname()
		setting := config["default"]
		if hostSetting, ok := config[reqInfo.Host]; ok {
			setting = hostSetting
		}
		// ctx.Warnf("setting: %+v", setting)
		if setting.Port > 0 && setting.Port < 65536 {
			req.URL.RawQuery = portMatcher.ReplaceAllString(req.URL.RawQuery,
				fmt.Sprintf("${1}port=%d${2}", setting.Port))
		}
		reqInfo.Epoch = time.Now().Unix()
		reqInfo.ReportUploaded = reqInfo.Uploaded
		reqInfo.Incomplete = int64(-2)
		init := ""
		if prevReqInfo, err := loadReqInfo(db, reqInfo.InfoHash); err != nil {
			if err != sql.ErrNoRows {
				ctx.Warnf("%s", err)
			} else {
				init = "init, "
			}
		} else {
			if query.Get("event") != "started" {
				// ctx.Warnf("prevReqInfo: %+v", prevReqInfo)
				deltaUploaded := reqInfo.Uploaded - prevReqInfo.Uploaded
				deltaDownloaded := reqInfo.Downloaded - prevReqInfo.Downloaded
				deltaEpoch := reqInfo.Epoch - prevReqInfo.Epoch
				if deltaUploaded >= 0 && deltaDownloaded >= 0 && deltaEpoch <= 10800 {
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
					// query.Set("uploaded", strconv.FormatInt(reqInfo.ReportUploaded, 10))
					// req.URL.RawQuery = query.Encode()
					req.URL.RawQuery = uploadedMatcher.ReplaceAllString(req.URL.RawQuery,
						fmt.Sprintf("${1}uploaded=%d${2}", reqInfo.ReportUploaded))
				}
			}
		}
		if err := saveReqInfo(db, reqInfo); err != nil {
			ctx.Warnf("%s", err)
		}
		ctx.Logf("%x %sup: %s/%s, down: %s, host: %s, epoch: %d",
			reqInfo.InfoHash,
			init,
			format(reqInfo.ReportUploaded),
			format(reqInfo.Uploaded),
			format(reqInfo.Downloaded),
			reqInfo.Host,
			reqInfo.Epoch)
		return goproxy.NEXT
	})

	proxy.HandleResponseFunc(func(ctx *goproxy.ProxyCtx) goproxy.Next {
		resp := ctx.Resp
		if resp != nil && resp.StatusCode == http.StatusOK {
			if bodyBytes, err := ioutil.ReadAll(resp.Body); err != nil {
				ctx.Warnf("%s", err)
			} else {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				if match := incompleteMatcher.FindSubmatch(bodyBytes); match != nil {
					query := ctx.Req.URL.Query()
					infoHash := query.Get("info_hash")
					incomplete, _ := strconv.ParseInt(string(match[1]), 10, 64)
					if queryInt64(ctx, "left") > 0 || query.Get("event") == "completed" {
						incomplete--
					}
					if err := saveIncomplete(db, infoHash, incomplete); err != nil {
						ctx.Warnf("%s", err)
					}
					ctx.Logf("%x incomplete: %d", infoHash, incomplete)
				}
			}
		}
		return goproxy.NEXT
	})

	go cleanup(db)
	proxy.Verbose = *arg.Verbose
	log.Fatal(proxy.ListenAndServe(*arg.Addr))
}
