package main

import (
	"fmt"
	"github.com/cheggaaa/pb"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

var client = &http.Client{Timeout: time.Duration(10) * time.Second}

// ファイル情報構造体
type Info struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

func main() {
	// 設定ファイルに追い出したい
	// 一度に取得するサイズ
	var getRange int64
	getRange = 256
	// ダウンロード間隔
	var interval time.Duration
	interval = 5
	// ダウンロードターゲット
	var protcol string
	protcol = "http"
	var host string
	host = "ftp.kddilabs.jp"
	var port string
	port = "80"
	var address string
	address = "infosystems/apache/httpd"
	dlUrl := protcol + "://" + host + ":" + port + "/" + address + "/"

	// ダウンロードしきるまでスリープしながらループ
	for {
		// TODO transerverから落としたいファイルの名前一覧引いてくる
		var file string
		file = "httpd-2.4.12.tar.gz"

		// リソースサイズ、最終更新日取得
		responseHead := do("HEAD", dlUrl+file, nil)
		i, _ := strconv.Atoi(responseHead.Header.Get("Content-Length"))
		contentLength := int64(i)
		//lastModified := headMap["Last-Modified"]
		fmt.Printf("Server Size:%d\n", contentLength)

		// カレントにあるファイルのサイズ、最終更新日取得
		info := readLocalFileInfo(file)

		if info.Size == contentLength {
			break
		}

		// ヘッダのRange組み立て
		var start string
		if info.Size == 0 {
			start = "0"
		} else {
			start = strconv.FormatInt(info.Size, 10)
		}
		next := strconv.FormatInt((info.Size + getRange), 10)
		header := map[string]string{"Range": "bytes=" + start + "-" + next}

		// リクエスト
		res := do("GET", dlUrl+file, header)
		fmt.Println(res.Header.Get("Content-Range"))

		time.Sleep(interval * time.Second)
	}
}

// ファイル情報取得
func readLocalFileInfo(path string) Info {
	i := Info{}

	if fileInfo, err := os.Stat(path); err != nil {
		// ローカルに同一ファイルが存在しない場合0バイトからDL開始する
		fmt.Println("NOT FOUND and NEW CREATE")
		i.Name = path
		i.Size = 0
	} else {
		i.Name = fileInfo.Name()
		i.Size = fileInfo.Size()
		i.Mode = fileInfo.Mode()
		i.ModTime = fileInfo.ModTime()
		i.IsDir = fileInfo.IsDir()
	}

	fmt.Printf("Client Name:%s,Size:%d,ModTime:%s,Mode:%s,IsDir:%t\n", i.Name, i.Size, i.ModTime, i.Mode, i.IsDir)
	return i
}

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

		// create multi writer
		i, _ := strconv.Atoi(res.Header.Get("Content-Length"))
		sourceSize := int64(i)

		progressBar := pb.New(int(sourceSize)).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
		progressBar.ShowSpeed = true
		progressBar.Start()

		writer := io.MultiWriter(file, progressBar)

		source := res.Body

		io.Copy(writer, source)
	}

	return res
}
