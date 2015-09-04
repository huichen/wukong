// 一个微博搜索的例子。
package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"flag"
	"github.com/huichen/wukong/engine"
	"github.com/huichen/wukong/types"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
)

const (
	SecondsInADay     = 86400
	MaxTokenProximity = 2
)

var (
	searcher = engine.Engine{}
	wbs      = map[uint64]Weibo{}
)

type Weibo struct {
	Id           uint64 `json:"id"`
	Timestamp    uint64 `json:"timestamp"`
	UserName     string `json:"user_name"`
	RepostsCount uint64 `json:"reposts_count"`
	Text         string `json:"text"`
}

/*******************************************************************************
    索引
*******************************************************************************/
func indexWeibo() {
	// 读入微博数据
	file, err := os.Open("../../testdata/weibo_data.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := strings.Split(scanner.Text(), "||||")
		if len(data) != 10 {
			continue
		}
		wb := Weibo{}
		wb.Id, _ = strconv.ParseUint(data[0], 10, 64)
		wb.Timestamp, _ = strconv.ParseUint(data[1], 10, 64)
		wb.UserName = data[3]
		wb.RepostsCount, _ = strconv.ParseUint(data[4], 10, 64)
		wb.Text = data[9]
		wbs[wb.Id] = wb
	}

	log.Print("添加索引")
	for docId, weibo := range wbs {
		searcher.IndexDocument(docId, types.DocumentIndexData{
			Content: weibo.Text,
			Fields: WeiboScoringFields{
				Timestamp:    weibo.Timestamp,
				RepostsCount: weibo.RepostsCount,
			},
		})
	}

	searcher.FlushIndex()
	log.Printf("索引了%d条微博\n", len(wbs))
}

/*******************************************************************************
    评分
*******************************************************************************/
type WeiboScoringFields struct {
	Timestamp    uint64
	RepostsCount uint64
}

type WeiboScoringCriteria struct {
}

func (criteria WeiboScoringCriteria) Score(
	doc types.IndexedDocument, fields interface{}) []float32 {
	if reflect.TypeOf(fields) != reflect.TypeOf(WeiboScoringFields{}) {
		return []float32{}
	}
	wsf := fields.(WeiboScoringFields)
	output := make([]float32, 3)
	if doc.TokenProximity > MaxTokenProximity {
		output[0] = 1.0 / float32(doc.TokenProximity)
	} else {
		output[0] = 1.0
	}
	output[1] = float32(wsf.Timestamp / (SecondsInADay * 3))
	output[2] = float32(doc.BM25 * (1 + float32(wsf.RepostsCount)/10000))
	return output
}

/*******************************************************************************
    JSON-RPC
*******************************************************************************/
type JsonResponse struct {
	Docs []*Weibo `json:"docs"`
}

func JsonRpcServer(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query().Get("query")
	output := searcher.Search(types.SearchRequest{
		Text: query,
		RankOptions: &types.RankOptions{
			ScoringCriteria: &WeiboScoringCriteria{},
			OutputOffset:    0,
			MaxOutputs:      100,
		},
	})

	// 整理为输出格式
	docs := []*Weibo{}
	for _, doc := range output.Docs {
		wb := wbs[doc.DocId]
		for _, t := range output.Tokens {
			wb.Text = strings.Replace(wb.Text, t, "<font color=red>"+t+"</font>", -1)
		}
		docs = append(docs, &wb)
	}
	response, _ := json.Marshal(&JsonResponse{Docs: docs})

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(response))
}

/*******************************************************************************
	主函数
*******************************************************************************/
func main() {
	// 解析命令行参数
	flag.Parse()

	// 初始化
	gob.Register(WeiboScoringFields{})
	searcher.Init(types.EngineInitOptions{
		SegmenterDictionaries: "../../data/dictionary.txt",
		StopTokenFile:         "../../data/stop_tokens.txt",
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
	})
	wbs = make(map[uint64]Weibo)

	// 索引
	go indexWeibo()

	// 捕获ctrl-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Print("捕获Ctrl-c，退出服务器")
			searcher.Close()
			os.Exit(0)
		}
	}()

	http.HandleFunc("/json", JsonRpcServer)
	http.Handle("/", http.FileServer(http.Dir("static")))
	log.Print("服务器启动")
	http.ListenAndServe(":8080", nil)
}
