Wukong Engine Codelab
====

**don't read. translation in progress**

At the end of this codelab, you will be able to write a simple full-text search website using Wukong engine.

If you do not know Go yet, [here](http://tour.golang.org/#1) is a tutorial.

## Engine basics

Engine handles user requests, segmentation, indexing and sorting in their own goroutines.

1. main goroutines, are responsible for sending and receiving user requests
2. segmenter goroutines
3. indexer goroutines, for building inverted index and lookup
4. ranker goroutines, for scoring and ranking documents

![](https://raw.github.com/huichen/wukong/master/docs/en/wukong_en.png)

**Indexing pipeline**

When a document is added, the main goroutine sends the doc through a channel to a segmenter goroutine, which segments the text and passes it to a indexer. Indexer build doc index from search keyword. Inverted index table is stored in memory for quick lookups.

**Search pipeline**

Main goroutine receives a user request, segments the query and passes it to an indexer goroutine. The indexer looks up corresponding documents for each search keyword and applies logic operation (list intersection) to get a set of docs that have all keywords. The list is then passed to a ranker to be scored, filtered and ranked. Finally the ranked docs are passed to the main goroutine through callback channel and returned to user .

In order to improve concurrency at search time, documents are sharded based on content and docid (number of shards can be specified by user) -- indexing and search requests are sent to all shards in parallel, and ranked docs are merged in the main goroutine before returning to user.

Above gives you basic idea on how Wukong works. A search system often consists of four parts, **document crawling**, **indexing**, **search** and **rendering**. I will explain them in following sections.

## Doc crawling

The technology of doc crawling deserve a full article. Fortunately for Weibo fetching, Sina provides simple APIs to get Weibo data and there's an easy-to-use [Go language SDK](http://github.com/huichen/gobo) which allows fetching with high concurrency.

I've downloaded about one hundred thousand weibo posts and stored them in testdata/weibo_data.txt, so you do not need to do it yourself. One line of the file has following format:

    <Weibo id>||||<timestamp>||||<User id>||||<user name>||||<share counts>||||<comment counts>||||<fav counts>||||<thumbnail URL>||||<large image URL>||||<body>

Weibos are saved in following struct (only stored fields used by this codelab)

```Go
type Weibo struct {
    Id           uint64
    Timestamp    uint64
    UserName     string
    RepostsCount uint64
    Text         string
}
```

However, if you are interested in the crawling details, see this script [testdata/crawl_weibo_data.go](/testdata/crawl_weibo_data.go).

## Indexing

To use Wukong engine you need to import two packages

```Go
import (
    "Github.com/huichen/wukong/engine"
    "Github.com/huichen/wukong/types"
)
```
The first one implements the engine, and the second package defines common structs. The engine has to be initialized before use, for example,

```Go
var search engine.Engine
searcher.Init(types.EngineInitOptions{
    SegmenterDictionaries: "../../data/dictionary.txt",
    StopTokenFile: "../../data/stop_tokens.txt",
    IndexerInitOptions: & types.IndexerInitOptions {
        IndexType: types.LocationsIndex,
    },
})
```

[Types.EngineInitOptions](/types/engine_init_options.go) defines options in engine initialization, such as where the to load dictionary file for Chinese word segmentation, stop word lists, type of inverted index table, BM25 parameters, and the default search and pagination options.

You must choose IndexerInitOptions.IndexType carefully. There are three different types of index table:

1. DocIdsIndex, provides the most basic index where only docids are stored.
2. FrequenciesIndex, in addition to DocIdsIndex, frequency of each tokens are stored to compute BM25 relevance.
3. LocationsIndex, which additionally stores each token's locations in a document, in order to compute token proximity.

From top to bottom, at the same time of providing more computing power, it also consumes more memory, especially for LocationsIndex when documents are long. Please choose accordingly.

After initialization you can add documents to the index now. Following example shows you how to do so,

```Go
searcher.IndexDocument(docId, types.DocumentIndexData{
    Content: weibo.Text, // Weibo struct is defined above. Content must be in UTF-8.
    Fields: WeiboScoringFields {
        Timestamp: weibo.Timestamp,
        RepostsCount: weibo.RepostsCount,
    },
})
```

DocId must be unique for each doc. Wukong engine allows you to use three types of index data:

1. Body of the document (Content field), which will be tokenized and added to the index.
2. You can also add tokens (Tokens field)directly when the body is empty, it allows user to bypass the built-in tokenzier and use any tokenization scheme he wants.
3. Document labels (Labels field), such as Weibo author, category, etc. Labels can be words that don't appear in the body of the doc.
4. Custom scoring fields (Fields), which allows you to add any type of data for ranking purpose. The 'Search' section will talk more on this.

Note that **tokens** and **labels** make up search **keywords**. Please remember the difference of the three concepts since documentation and code will mention them many times. Searching in the index table is a logical operation on docs. For example, say a document has "bicycle" the token and a "fitness" label, but "fitness" doesn't appear in text, when the query "bicycle" + "fitness" combination is searched, the article will be looked up. The purpose of having document labels is to quickly reduce number of documents from non-literal scopes.

The Engine adds indices asynchronously, so when IndexDocument returns the doc may not yet be actually indexed. If you need to wait before futher operation, call following function

```Go
searcher.FlushIndex ()
```

## Search

Search process in two steps, the first step is to look in the index table containing the search key documents, which has been introduced in the last one before. The second step is an index of all documents to be sorted.

Sort the core of the document score. Wukong engine allows you to customize any of the scoring rules (scoring criteria). Search the microblogging example, we define scoring rules are as follows:

1. first sort by keywords close distance, for example search for "cycling", the phrase will be cut into two words, "bicycle" and "movement", there are two words next to the article should be at two key separate article in front of the word.
2. and then follow the microblogging Published roughly sort, and every three days as a team, later articles echelon top surface.
3. score was finally given microblogging BM25 * (1 + forwarding number / 10000)

Such rules need to save some of the score for each document data, such as microblogging Published, microblogging forwarding number and so on. The data is stored in the following structure body
```Go
type WeiboScoringFields struct {
    Timestamp uint64
    RepostsCount uint64
}
```
You may have noticed, this is the last one when the document is added to the index passed to the function call IndexDocument parameter type (in fact, that argument is the interface {} type, so you can pass any type of structure). The data stored in the memory sequencer waits for the call.

With these data, we can score, the code is as follows:
```Go
type WeiboScoringCriteria struct {
}

func (criteria WeiboScoringCriteria) Score (
        doc types.IndexedDocument, fields interface {}) [] float32 {
        if reflect.TypeOf (fields)! = reflect.TypeOf (WeiboScoringFields {}) {
                return [] float32 {}
        }
        wsf: = fields. (WeiboScoringFields)
        output: = make ([] float32, 3)
        if doc.TokenProximity> MaxTokenProximity {/ / Step
                output [0] = 1.0 / float32 (doc.TokenProximity)
        } Else {
                output [0] = 1.0
        }
        output [1] = float32 (wsf.Timestamp / (SecondsInADay * 3)) / / Step
        output [2] = float32 (doc.BM25 * (1 + float32 (wsf.RepostsCount) / 10000)) / / The third step
        return output
}
```
WeiboScoringCriteria actually inherited types.ScoringCriteria interface that implements Score function. This function takes two parameters:

1. Types.IndexedDocument indexer parameters passed from the data obtained, for example, word frequency, word specific location, BM25 value, close to the degrees and other information, see specific [types / index.go] (/ types / index.go) of comments.
(2) The second parameter is the type of interface {}, you can put this type of understanding into the C language void pointer, it can point to any data type. In our example, the point is WeiboScoringFields structure, and through reflection mechanism checks the correct type.

With custom scoring data and custom scoring rules, we will be able to search, and see the code below

```Go
response: = searcher.Search (types.SearchRequest {
Text: "cycling"
RankOptions: & types.RankOptions {
ScoringCriteria: & WeiboScoringCriteria {},
OutputOffset: 0,
MaxOutputs: 100,
},
})
```

Which, Text is entered search phrase (must be UTF-8 format), will be sub-word as a keyword. And the same index, Wukong engine allows to bypass the built-in word documents directly enter keywords and labels, see types.SearchRequest structure annotation. RankOptions defines the sorting options. WeiboScoringCriteria is our scoring rules defined above. In addition, you can also OutputOffset and MaxOutputs parameters control the paging output. Search results in response variable, the specific content [types / search_response.go] (/ types / search_response.go) SearchResponse defined in the file structure, such as the structure returned keywords appear in the document location, you can used to generate the document summary.

## Rendering

The final step to complete the search is to render the ranked docs to use. The usual practice is to make a backend service with the search engine, and then let the frontend calls the BE through JSON. Since it's not directly related to Wukong, I's skipping the details here.

## Summary

Now you should have gained a basic understanding of how Wukong engine works. I suggest you finishing the example by yourself. However if you do not have patience, here's the completed code [examples/codelab/search_server.go](/examples/codelab/search_server.go), which has less than 200 lines. Run this example is very simple: just enter examples/codelab directory and

    go run search_server.go

Wait for a couple seconds until indexing is down and then open [http://localhost:8080](http://localhost:8080) in your browser. This implements a simplified version of following site

http://soooweibo.com

If you want to learn more about Wukong engine, I suggest you read the code:

    /core       core components, including the indexer and ranker
    /data       tokenization dictionary and stopword file
    /docs       documentation
    /engine     engine, implementation of main/segmenter/indexer/ranker goroutines
    /examples   examples and benchmark code
    /testdata   test data
    /types      common structs
    /utils      common functions
