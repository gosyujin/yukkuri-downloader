package main

import (
	"encoding/json"
	"fmt"
	"github.com/cheggaaa/pb"
	// 使いたいけどWindowsだとめんどくさそう
	_ "github.com/mitchellh/colorstring"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
)

// ファイル情報構造体
type Info struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

// セッティングファイル情報
type Setting struct {
	GetRange int64
	Interval int64
	Scheme   string
	Host     string
	Port     string
	Path     string
	Proxy    bool
}

// Windows以外はos.Getenv("HOME")？
var settingFile = os.Getenv("USERPROFILE") + "/go-tran.json"
var client = &http.Client{Timeout: time.Duration(10) * time.Second}
var localFileSize int64
var serverFileSize int64

func main() {
	useGlobalLogger()
	s := initialize()

	// 設定ファイルに追い出したい
	var address string
	address = "infosystems/apache/httpd"
	dlUrl := s.Scheme + "://" + s.Host + "/" + address + "/"

	// ダウンロードしきるまでスリープしながらループ
	for {
		// TODO transerverから落としたいファイルの名前一覧引いてくる
		var file string
		file = "httpd-2.4.12.tar.gz"

		// リソースサイズ、最終更新日取得
		responseHead := do("HEAD", dlUrl+file, nil)
		i, _ := strconv.Atoi(responseHead.Header.Get("Content-Length"))
		contentLength := int64(i)
		serverFileSize = contentLength
		//lastModified := responseHead.Header.Get("Last-Modified")

		// ローカルにあるファイルの情報取得
		info := readLocalFileInfo(file)
		localFileSize = info.Size

		if localFileSize == serverFileSize {
			break
		}

		// ヘッダのRange組み立て
		var start string
		if info.Size == 0 {
			start = "0"
		} else {
			start = strconv.FormatInt(info.Size, 10)
		}
		next := strconv.FormatInt((info.Size + s.GetRange), 10)
		header := map[string]string{"Range": "bytes=" + start + "-" + next}

		// リクエスト
		_ = do("GET", dlUrl+file, header)
		//log.Println(res.Header.Get("Content-Range"))

		time.Sleep(time.Duration(s.Interval) * time.Second)
	}
}

func useGlobalLogger() {
	log.SetFlags(log.Ldate | log.Ltime)
	//log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("[go-tran]")
}

// 初期設定
func initialize() Setting {
	log.Println("Initialize.")
	log.Println("Read setting file: " + settingFile)
	s := Setting{}

	// 設定ファイル存在チェック、なければ生成
	_, err := os.Stat(settingFile)
	if os.IsNotExist(err) {
		log.Println(err)
		log.Println("Create json file as default value: " + settingFile)
		createDefaultSetting()
	}

	// 設定ファイル読み込み構造体に格納
	s = readSettingFile()

	if s.Proxy {
		log.Println("USE system proxy HTTP_PROXY and HTTPS_PROXY")
	} else {
		log.Println("CLEAR system proxy HTTP_PROXY and HTTPS_PROXY")
		os.Setenv("http_proxy", "")
		os.Setenv("HTTP_PROXY", "")
		os.Setenv("https_proxy", "")
		os.Setenv("HTTPS_PROXY", "")
	}

	log.Println("  HTTP_PROXY   :" + os.Getenv("http_proxy"))
	log.Println("  HTTPS_PROXY  :" + os.Getenv("https_proxy"))
	log.Println("  Scheme       :" + s.Scheme)
	log.Println("  Host         :" + s.Host)
	log.Println("  Port         :" + s.Port)
	log.Println("  Path         :" + s.Path)
	log.Println(fmt.Sprintf("  GetRange     :%d", s.GetRange))
	log.Println(fmt.Sprintf("  Interval     :%d", s.Interval))

	log.Println("Read setting file end.")

	return s
}

// ~/ にデフォルトの設定ファイルを生成する
func createDefaultSetting() {
	file, err := os.Create(settingFile)
	if err != nil {
	}

	s := Setting{}

	s.Proxy = true
	s.GetRange = 1024
	s.Interval = 5
	downloadUrl, err := url.Parse("http://ftp.kddilabs.jp:80/infosystems/apache/httpd/httpd-2.4.12.tar.gz")
	if err != nil {
	}
	s.Scheme = downloadUrl.Scheme
	host, port, _ := net.SplitHostPort(downloadUrl.Host)
	s.Host = host
	s.Port = port
	s.Path = downloadUrl.Path

	encoder := json.NewEncoder(file)
	encoder.Encode(s)
}

// 設定ファイル読み込み
func readSettingFile() Setting {
	jsonString, err := ioutil.ReadFile(settingFile)
	if err != nil {
	}
	s := Setting{}
	json.Unmarshal(jsonString, &s)

	return s
}

// ダウンロードファイル情報取得
func readLocalFileInfo(path string) Info {
	i := Info{}

	if fileInfo, err := os.Stat(path); err != nil {
		// ローカルに同一ファイルが存在しない場合0バイトからDL開始する
		log.Println("NOT FOUND and NEW CREATE: " + path)
		i.Name = path
		i.Size = 0
	} else {
		i.Name = fileInfo.Name()
		i.Size = fileInfo.Size()
		i.Mode = fileInfo.Mode()
		i.ModTime = fileInfo.ModTime()
		i.IsDir = fileInfo.IsDir()
	}

	//fmt.Printf("Client Name:%s,Size:%d,ModTime:%s,Mode:%s,IsDir:%t\n", i.Name, i.Size, i.ModTime, i.Mode, i.IsDir)
	return i
}

// リクエストと、ファイル書き込み
func do(method string, url string, header map[string]string) *http.Response {
	req, err := http.NewRequest(method, url, nil)
	if header != nil {
		req.Header.Add("Range", header["Range"])
	}
	if err != nil {
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	// GETの場合、Bodyの内容をファイルに書き込み
	if method == "GET" {
		// File open
		_, fileName := path.Split(url)
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
		}
		defer file.Close()

		progressBar := pb.New(int(serverFileSize))
		progressBar.SetUnits(pb.U_BYTES)
		progressBar.SetRefreshRate(time.Millisecond * 10)
		progressBar.ShowCounters = true
		progressBar.ShowTimeLeft = true
		progressBar.ShowSpeed = true
		progressBar.SetMaxWidth(80)
		// 続きから
		progressBar.Set(int(localFileSize))
		progressBar.Start()

		writer := io.MultiWriter(file, progressBar)

		source := res.Body
		io.Copy(writer, source)
	}

	return res
}
