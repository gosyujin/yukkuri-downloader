package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"os"
	"path/filepath"
)

func main() {
	m := martini.Classic()

	m.Use(render.Renderer(render.Options{
		Directory: "templates",
		//		Layout: "layout",
		Extensions: []string{".tmpl"},
		Charset:    "utf-8",
	}))

	m.Get("/", IndexRender)
	m.Get("/dlFileList", dlFileList)
	m.Run()
}

func dlFileList(params martini.Params) (int, []byte) {
	root := "./public"
	var files [1]string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)

		fmt.Println(rel)
		files[0] = path

		return nil

	})

	if err != nil {
		fmt.Println(1, err)
	}

	file, err := os.Open(files[0])

	if err != nil {
		fmt.Println(1, err)
	}

	fileStat, _ := file.Stat()
	fileSize := fileStat.Size()

	var buffer int64
	buffer = 10

	var val uint8
	var i int64
	var ret []byte
	for i = 0; i <= (fileSize / buffer); i++ {
		b := make([]byte, buffer)
		file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &val)
		for j := 0; j < len(b); j++ {
			fmt.Printf("%X ", b[j])
		}
		fmt.Println("")
		ret = b
	}

	return 200, ret
}
