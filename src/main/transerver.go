package main

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"os"
	"path/filepath"
	"fmt"
	"bytes"
	"encoding/binary"
)

func main() {
	m := martini.Classic()
	
	m.Use(render.Renderer(render.Options{
		Directory: "templates",
//		Layout: "layout",
		Extensions: []string{".tmpl"},
		Charset: "utf-8",
	}))
	
	m.Get("/", IndexRender)
	m.Get("/dlFileList", dlFileList)
	m.Run()
}

func dlFileList(params martini.Params) (int, string) {
	root := "./public"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
	
		rel, err := filepath.Rel(root, path)
		fmt.Println(rel)
		
		file, err := os.Open(path)
		
		if err != nil {
			fmt.Println(1, err)
		}
		
		b := make([]byte, 2)
		file.Read(b)
		
		var val uint8
		err2 := binary.Read(bytes.NewBuffer(b), binary.BigEndian, &val)

		if err2 != nil {
			fmt.Println(1, err)
		}
		
		fmt.Println("readdata:", val)

		return nil
	})
	
	if err != nil {
		fmt.Println(1, err)
	}
	
	return 200, "End!"
}

