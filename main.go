package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"

	"github.com/gorilla/mux"
)

type DirectoryItem struct {
	Name  string `json:"name,omitemty"`
	IsDir bool   `json:"isDir,omitempty"`
	Size  int64  `json:"size,omitempty"`
}

type DirectoryInfo struct {
	Path string          `json:"path,omitemty"`
	Dirs []DirectoryItem `json:"dirs,omitempty"`
}

type DirectoryTraversalInfo struct {
	Path      string `json:"path,omitemty"`
	DirCount  int    `json:"dirCount,omitemty"`
	FileCount int    `json:"fileCount,omitemty"`
	TotalSize int64  `json:"totalSize,omitemty"`
}

func GetOneDirItems(w http.ResponseWriter, req *http.Request) {
	var dirInfo DirectoryInfo
	fpath := rootPath

	query := req.URL.Query()
	path := query["path"][0]

	fpath = fpath + path

	dirInfo, err := CheckEachItem(fpath)

	if err != nil {
		panic(err)
	}

	json.NewEncoder(w).Encode(dirInfo)
}

func CheckEachItem(dirPath string) (directory DirectoryInfo, err error) {
	var items []DirectoryItem

	dir, err := ioutil.ReadDir(dirPath)

	if err != nil {
		return directory, err
	}

	for _, fi := range dir {
		if fi.IsDir() {
			items = append(items, DirectoryItem{Name: fi.Name(), IsDir: true, Size: 0})

		} else {
			items = append(items, DirectoryItem{Name: fi.Name(), IsDir: false, Size: fi.Size()})
		}
	}
	directory = DirectoryInfo{Path: dirPath, Dirs: items}

	return directory, nil
}

func GetDirInfo(w http.ResponseWriter, req *http.Request) {
	baseUrl := "http://localhost:8080/api/v1/directory-items?path="
	query := req.URL.Query()
	path := query["path"][0]
	baseUrl += path

	dirTotalCount := 0
	fileTotalCount := 0
	var fileTotalSizes int64 = 0
	doneTotalCount := 0

	dirCount := make(chan int)
	fileCount := make(chan int)
	fileSizes := make(chan int64)
	doneCount := make(chan int)

	go TraversingDir(baseUrl, dirCount, fileCount, doneCount, fileSizes)

out:
	for {
		select {
		case dir := <-dirCount:
			dirTotalCount += dir

		case f := <-fileCount:
			fileTotalCount += f

		case fsize := <-fileSizes:
			fileTotalSizes += fsize

		case done := <-doneCount:
			doneTotalCount += done

			if doneTotalCount == dirTotalCount+1 {
				break out
			}
		}
	}

	var dirTravInfo *DirectoryTraversalInfo = new(DirectoryTraversalInfo)
	dirTravInfo.Path = path
	dirTravInfo.DirCount = dirTotalCount
	dirTravInfo.FileCount = fileTotalCount
	dirTravInfo.TotalSize = fileTotalSizes

	json.NewEncoder(w).Encode(dirTravInfo)
}

func TraversingDir(url string, dirCount chan int, fileCount chan int, doneCount chan int, fileSizes chan int64) {
	resp, _ := http.Get(url)

	dirInfo := DirectoryInfo{}

	body, _ := ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()

	json.Unmarshal([]byte(body), &dirInfo)

	for _, itm := range dirInfo.Dirs {
		if itm.IsDir {
			newUrl := url + "/" + itm.Name
			dirCount <- 1
			go TraversingDir(newUrl, dirCount, fileCount, doneCount, fileSizes)
		} else {
			fileSizes <- itm.Size
			fileCount <- 1
		}
	}
	doneCount <- 1
}

var rootPath string

func main() {

	flag.Parse()
	rootPaths := flag.Args()

	if len(rootPaths) == 0 {
		rootPaths = []string{"E:\\"}
	}
	// e.g. E:\
	rootPath = rootPaths[0]

	runtime.GOMAXPROCS(5)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/directory-items", GetOneDirItems).Methods("GET")
	router.HandleFunc("/api/v1/directory-items/statistics", GetDirInfo).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", router))
}
