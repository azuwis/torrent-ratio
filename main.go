package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// Arg ...
type Arg struct {
	Addr    *string
	DbPath  *string
	Verbose *bool
}

// Setting ...
type Setting struct {
	Uploaded    [2]float64
	Downloaded  [2]float64
	PercentMin  float64
	PercentMax  float64
	PercentStep float64
	Speed       int64
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
	queryMatcher      = regexp.MustCompile(`(^|&)uploaded=\d+(&|$)`)
)

func parseArg() Arg {
	var arg Arg
	defaultDbPath := ":memory:"
	if usr, err := user.Current(); err != nil {
		log.Print(err)
	} else {
		defaultDbPath = filepath.Join(usr.HomeDir, ".torrent-ratio.db")
	}
	arg.Verbose = flag.Bool("v", false, "enable verbose logging")
	arg.Addr = flag.String("addr", "127.0.0.1:8082", "proxy listen address")
	arg.DbPath = flag.String("db", defaultDbPath, "database path")
	flag.Parse()
	return arg
}

func getInt64(query *url.Values, key string) int64 {
	i, err := strconv.ParseInt(query.Get(key), 10, 64)
	if err != nil {
		log.Print(err)
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

func format(num int64) string {
	float := float64(num)
	for _, unit := range []string{"", "K", "M", "G"} {
		if math.Abs(float) < float64(1024) {
			return fmt.Sprintf("%3.1f%s", float, unit)
		}
		float /= float64(1024)
	}
	return fmt.Sprintf("%3.1f%s", float, "T")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	arg := parseArg()

	db, err := sql.Open("sqlite3", *arg.DbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	config := map[string]Setting{
		"default": {
			Uploaded:    [2]float64{0.1, 0.6},
			Downloaded:  [2]float64{0, 0.07},
			PercentMin:  0.2,
			PercentMax:  0.5,
			PercentStep: 0.02,
			Speed:       51200,
		},
		"127.0.0.1": {
			Uploaded:    [2]float64{2, 2},
			Downloaded:  [2]float64{1, 1},
			PercentMin:  0,
			PercentMax:  0,
			PercentStep: 0.02,
			Speed:       51200,
		},
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		reqInfos, err := loadAllReqInfo(db)
		if err != nil {
			log.Print(err)
		}
		epoch := time.Now().Unix()
		fmt.Fprintln(w, "info_hash                                uploaded     downloaded announced incomplete host")
		for _, reqInfo := range reqInfos {
			fmt.Fprintf(w, "%x %-6s%-6s %-10s %-9d %-10d %s\n",
				reqInfo.InfoHash,
				format(reqInfo.ReportUploaded),
				format(reqInfo.Uploaded),
				format(reqInfo.Downloaded),
				(epoch-reqInfo.Epoch)/60,
				reqInfo.Incomplete,
				reqInfo.Host)
		}
	})

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		var reqInfo ReqInfo
		query := req.URL.Query()
		reqInfo.InfoHash = query.Get("info_hash")
		reqInfo.Uploaded = getInt64(&query, "uploaded")
		reqInfo.Downloaded = getInt64(&query, "downloaded")
		if reqInfo.InfoHash == "" || reqInfo.Uploaded < 0 || reqInfo.Downloaded < 0 {
			return req, nil
		}
		reqInfo.Host = req.URL.Hostname()
		setting := config["default"]
		if hostSetting, ok := config[reqInfo.Host]; ok {
			setting = hostSetting
		}
		// ctx.Warnf("setting: %+v", setting)
		reqInfo.Epoch = time.Now().Unix()
		reqInfo.ReportUploaded = reqInfo.Uploaded
		reqInfo.Incomplete = int64(-2)
		init := ""
		if query.Get("event") != "started" {
			if prevReqInfo, err := loadReqInfo(db, reqInfo.InfoHash); err != nil {
				if err != sql.ErrNoRows {
					ctx.Warnf("%s", err)
				} else {
					init = "init, "
				}
			} else {
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
					req.URL.RawQuery = queryMatcher.ReplaceAllString(req.URL.RawQuery,
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
		return req, nil
	})

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp.StatusCode == http.StatusOK {
			if bodyBytes, err := ioutil.ReadAll(resp.Body); err != nil {
				ctx.Warnf("%s", err)
			} else {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				if match := incompleteMatcher.FindSubmatch(bodyBytes); match != nil {
					query := ctx.Req.URL.Query()
					infoHash := query.Get("info_hash")
					incomplete, _ := strconv.ParseInt(string(match[1]), 10, 64)
					if getInt64(&query, "left") > 0 || query.Get("event") == "completed" {
						incomplete--
					}
					if err := saveIncomplete(db, infoHash, incomplete); err != nil {
						ctx.Warnf("%s", err)
					}
					ctx.Logf("%x incomplete: %d", infoHash, incomplete)
				}
			}
		}
		return resp
	})

	proxy.Verbose = *arg.Verbose
	log.Fatal(http.ListenAndServe(*arg.Addr, proxy))
}
