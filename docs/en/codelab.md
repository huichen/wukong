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

Search is done in two steps. The first step is to look up documents in inverted index table which contains the keywords. This part has been explained in details before. The second step is to score and to rank the documents returned from indexer.

Wukong engine allows you to use any scoring criteria you want in ranking. Taking the Weibo search as an example, we define scoring rules as follow,

1. first sort by token proximity, for example for search query "bicycle sports", the phrase will be cut into two tokens, "bicycle" and "sports". A document with the two tokens next to each other in the same order will appear above other docs.
2. and rank by Weibo's creation timestamp roughly (3 days per bucket).
3. finally sort by Weibo's BM25 * (1 + <shard count> / 10000)

To use such rule, we need to store scoring fields in ranker. Following structure is defined
```Go
type WeiboScoringFields struct {
    Timestamp uint64
    RepostsCount uint64
}
```
You may have noticed, this is exactly the same type of argument passed to the indexer when calling IndexDocument function (in fact, the function argument has the type of interface{}, so you can pass in anything you want).

Now you can define the scoring criteria in code,
```Go
type WeiboScoringCriteria struct {
}

func (criteria WeiboScoringCriteria) Score (
    doc types.IndexedDocument, fields interface {}) []float32 {
    if reflect.TypeOf(fields)! = reflect.TypeOf(WeiboScoringFields{}) {
        return []float32{}
    }
    wsf: = fields.(WeiboScoringFields)
    output: = make([]float32, 3)
    if doc.TokenProximity > MaxTokenProximity { // step 1
        output[0] = 1.0 / float32(doc.TokenProximity)
    } else {
        output[0] = 1.0
    }
    output[1] = float32(wsf.Timestamp / (SecondsInADay * 3)) // step 2
    output[2] = float32(doc.BM25 * (1 + float32(wsf.RepostsCount) / 10000)) // step 3
    return output
}
```
WeiboScoringCriteria inherits types.ScoringCriteria interface and implements the Score function. This function takes two arguments:

1. A types.IndexedDocument slice with information on indexed docs, for example, token frequencies, token locations, BM25 score, token proximity, etc.
2. The second argument has the type of interface{}. You can think this type as the void pointer in C language, which allows it to point to any data type. In our example, the data is a WeiboScoringFields struct, whose type was checked in the Score function using reflection mechanism.

With custom scoring criteria, search is made easy and customizable. Code example for doing searching,

```Go
response: = searcher.Search(types.SearchRequest {
    Text: "cycling sports",
    RankOptions: &types.RankOptions{
        ScoringCriteria: & WeiboScoringCriteria {},
        OutputOffset: 0,
        MaxOutputs: 100,
    },
})
```

In which, Text is the search phrase (must be in UTF-8). And the same as in indexing, Wukong engine allows you to bypass the built-in segmenter and to enter tokens and labels directly. See types.SearchRequest struct's comment. RankOptions defines options used in scoring and ranking. WeiboScoringCriteria is our scoring criteria defined above. OutputOffset and MaxOutputs parameters are set to control output pagination. Search results are returned in a types.SearchResponse type, which contains query's segmented tokens and their locations in each doc, which can be used to generate search snippets.

## Rendering

The final step to complete the search is rendering the ranked docs in front of the user. The usual practice is to make a backend service with the search engine, and then let the frontend calls the BE through JSON. Since it's not directly related to Wukong, I'm skipping the details here.

## Summary

Now you should have gained a basic understanding on how Wukong engine works. I suggest you finishing the example by yourself. However if you do not have patience, here's the completed code [examples/codelab/search_server.go](/examples/codelab/search_server.go), which has less than 200 lines. Running this example is simple. Just enter examples/codelab directory and type

    go run search_server.go

Wait for a couple seconds until indexing is done and then open [http://localhost:8080](http://localhost:8080) in your browser. The page implements a simplified version of following site

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
