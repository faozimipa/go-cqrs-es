package esclient

import (
	"bytes"
	"context"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/faozimipa/go-cqrs-es/pkg/es/serializer"
)

func Update(ctx context.Context, transport esapi.Transport, index, documentID string, document any) (*esapi.Response, error) {
	doc := Doc{Doc: document}
	reqBytes, err := serializer.Marshal(&doc)
	if err != nil {
		return nil, err
	}

	request := esapi.UpdateRequest{
		Index:      index,
		DocumentID: documentID,
		Body:       bytes.NewReader(reqBytes),
		Refresh:    "true",
	}

	return request.Do(ctx, transport)
}
