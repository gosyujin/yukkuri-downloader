package main

import (
	"fmt"
	"github.com/cheggaaa/pb"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

var client = &http.Client{Timeout: time.Duration(10) * time.Second}

// ファイル情報
type Info struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

func main() {
	// 1.ファイルリスト取得
	// 2.いっこずつ投げる
	// 5.Content-Length分落としてきたら次へ
	//dlUrl := "http://localhost:3000/dlFileList"

	// ダウンロードターゲット
	dlUrl := "http://ftp.kddilabs.jp/infosystems/apache//httpd/httpd-2.4.12.tar.gz"
	//dlUrl := "http://localhost:3000/sakura.exe"
	// 一度に取得するサイズ
	var getRange int64
	getRange = 256
	// ダウンロード間隔
	var interval time.Duration
	interval = 5

	// ダウンロードしきるまでスリープしながらループ
	for {
		// リソースサイズ、最終更新日取得
		responseHead := do("HEAD", dlUrl, nil)
		i, _ := strconv.Atoi(responseHead.Header.Get("Content-Length"))
		contentLength := int64(i)
		//lastModified := headMap["Last-Modified"]

		// カレントにあるファイルのサイズ、最終更新日取得
		info := readFileInfo("sakura.exe")

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
		res := do("GET", dlUrl, header)
		fmt.Println(res.Header.Get("Content-Range"))

		time.Sleep(interval * time.Second)
	}
}

// ファイル情報取得
func readFileInfo(path string) Info {
	i := Info{}

	if fileInfo, err := os.Stat(path); err != nil {
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

	//fmt.Println("Target file is...")
	//fmt.Printf("Name: %s\n", i.Name)
	fmt.Printf("Size: %d\n", i.Size)
	//fmt.Printf("Mode: %s\n", i.Mode)
	//fmt.Printf("ModTime: %s\n", i.ModTime)
	//fmt.Printf("IsDir: %t\n", i.IsDir)
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

	if method == "GET" {
		// File open
		file, err := os.OpenFile("sakura.exe", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
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
