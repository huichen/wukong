package engine

import (
	"fmt"
	"github.com/huichen/wukong/types"
)

func indexedDocIdsToString(docs types.SearchResponse) (output string) {
	for _, doc := range docs.Docs {
		output += fmt.Sprintf("[%d] ",
			doc.DocId)
	}
	return
}
