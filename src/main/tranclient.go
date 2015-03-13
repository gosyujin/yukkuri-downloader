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

func main() {
	dlUrl := "http://localhost:3000/dlFileList"
	//dlUrl := "http://localhost/sample.zip"
	response, err := http.Get(dlUrl)

	if err != nil {
		panic(err)
	}

	defer response.Body.Close()

	fmt.Println(response.Header)

	// File open
	file, err := os.OpenFile("hoge.html", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// create multi writer
	i, _ := strconv.Atoi(response.Header.Get("Content-Length"))
	sourceSize := int64(i)

	progressBar := pb.New(int(sourceSize)).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
	progressBar.ShowSpeed = true
	progressBar.Start()

	writer := io.MultiWriter(file, progressBar)

	source := response.Body

	io.Copy(writer, source)
}
