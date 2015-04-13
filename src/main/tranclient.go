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
	"runtime"
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
	File     string
	Proxy    bool
}

var client = &http.Client{Timeout: time.Duration(10) * time.Second}
var localFileSize int64
var serverFileSize int64

func main() {
	var settingFile string
	if runtime.GOOS == "windows" {
		settingFile = os.Getenv("USERPROFILE") + "/go-tran.json"
	} else {
		settingFile = os.Getenv("HOME") + "/go-tran.json"
	}

	useGlobalLogger()
	s := initialize(settingFile)

	dlUrl := s.Scheme + "://" + s.Host + ":" + s.Port + s.Path

	// ダウンロードしきるまでスリープしながらループ
	for {
		var file string
		file = s.File

		// リソースサイズ、最終更新日取得
		responseHead := do("HEAD", dlUrl+file, nil)
		i, _ := strconv.Atoi(responseHead.Header.Get("Content-Length"))
		contentLength := int64(i)
		serverFileSize = contentLength
		serverModTime, err := time.Parse(time.RFC1123, responseHead.Header.Get("Last-Modified"))
		if err != nil {
		}
		serverModTime = serverModTime.UTC()

		// ローカルにあるファイルの情報取得
		info := readLocalFileInfo(file)
		localFileSize = info.Size
		localModTime, err := time.Parse(time.RFC1123, info.ModTime.Format(time.RFC1123))
		if err != nil {
		}
		localModTime = localModTime.UTC()

		if isNewerServerFile(serverModTime, localModTime) {
			// サーバのファイルの方が新しい場合ファイル削除
			fmt.Println("")
			log.Println(fmt.Sprintf("Timestamp server: %v", serverModTime))
			log.Println(fmt.Sprintf("Timestamp local : %v", localModTime))
			log.Println("Change server file ? And delete file")

			if err := os.Remove(file); err != nil {
			}

			// プログレスバーの更新のため、ファイルを消したタイミングでファイルサイズクリア
			localFileSize = 0
		} else {
			if localFileSize == serverFileSize {
				fmt.Println("")
				log.Println(fmt.Sprintf("Timestamp server: %v", serverModTime))
				log.Println(fmt.Sprintf("Timestamp local : %v", localModTime))
				log.Println("Download Complete")
				break
			}
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
		// log.Println(res.Header.Get("Content-Range"))

		time.Sleep(time.Duration(s.Interval) * time.Second)
	}
}

func useGlobalLogger() {
	log.SetFlags(log.Ldate | log.Ltime)
	//log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("[go-tran] ")
}

// 初期設定
func initialize(settingFile string) Setting {
	log.Println("Initialize.")
	log.Println("Read setting file: " + settingFile)
	s := Setting{}

	// 設定ファイル存在チェック、なければ生成
	_, err := os.Stat(settingFile)
	if os.IsNotExist(err) {
		log.Println(err)
		log.Println("Create json file as default value: " + settingFile)
		createDefaultSetting(settingFile)
	}

	// 設定ファイル読み込み構造体に格納
	s = readSettingFile(settingFile)

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
	log.Println("  Url          :" + s.Scheme + "://" + s.Host + ":" + s.Port + s.Path + s.File)
	log.Println("  Scheme       :" + s.Scheme)
	log.Println("  Host         :" + s.Host)
	log.Println("  Port         :" + s.Port)
	log.Println("  Path         :" + s.Path)
	log.Println("  File         :" + s.File)
	log.Println(fmt.Sprintf("  GetRange     :%d byte", s.GetRange))
	log.Println(fmt.Sprintf("  Interval     :%d sec", s.Interval))

	log.Println("Read setting file end.")

	return s
}

// ~/ にデフォルトの設定ファイルを生成する
func createDefaultSetting(settingFile string) {
	file, err := os.Create(settingFile)
	if err != nil {
	}

	s := Setting{}

	s.Proxy = false
	s.GetRange = 2048
	s.Interval = 3
	downloadUrl, err := url.Parse("http://ftp.kddilabs.jp:80/infosystems/apache/httpd/httpd-2.4.12.tar.gz")
	//downloadUrl, err := url.Parse("https://github.com:443/gosyujin/gosyujin.github.com/archive/v1.0.tar.gz")
	if err != nil {
	}
	s.Scheme = downloadUrl.Scheme
	fmt.Println(downloadUrl.Host)
	fmt.Println(net.SplitHostPort(downloadUrl.Host))
	host, port, _ := net.SplitHostPort(downloadUrl.Host)
	s.Host = host
	s.Port = port
	s.Path, s.File = path.Split(downloadUrl.Path)

	encoder := json.NewEncoder(file)
	encoder.Encode(s)
}

// 設定ファイル読み込み
func readSettingFile(settingFile string) Setting {
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
		log.Println("NOT found: " + path)
		i.Name = path
		i.Size = 0
		i.ModTime = time.Now()
	} else {
		// log.Println("found: " + path)
		i.Name = fileInfo.Name()
		i.Size = fileInfo.Size()
		i.Mode = fileInfo.Mode()
		i.ModTime = fileInfo.ModTime()
		i.IsDir = fileInfo.IsDir()
	}

	//log.Printf("Client Name: %s, Size: %d, ModTime: %s, Mode: %s, IsDir: %t", i.Name, i.Size, i.ModTime, i.Mode, i.IsDir)
	return i
}

// サーバのファイルが更新されているか比較する
func isNewerServerFile(server time.Time, local time.Time) bool {
	if server.Sub(local) > 0 {
		//log.Printf("Server is new")
		return true
	} else {
		//log.Printf("Server is old")
		return false
	}
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

		if res.StatusCode != http.StatusPartialContent {
			log.Println("This url is NOT supported partial GET request")
			log.Println("Exit")
			os.Exit(1)
		}

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
