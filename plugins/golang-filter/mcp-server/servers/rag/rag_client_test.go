package rag

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

func getRAGClient() (*RAGClient, error) {
	config := &config.Config{
		RAG: config.RAGConfig{
			Splitter: config.SplitterConfig{
				Provider:     "recursive",
				ChunkSize:    200,
				ChunkOverlap: 20,
			},
			Threshold: 0.5,
			TopK:      10,
		},

		LLM: config.LLMConfig{
			Provider: "openai",
			APIKey:   "sk-xxxx",
			BaseURL:  "https://openrouter.ai/api/v1",
			Model:    "openai/gpt-4o",
		},

		Embedding: config.EmbeddingConfig{
			Provider: "dashscope",
			APIKey:   "sk-xxxx",
			Model:    "text-embedding-v4",
		},

		VectorDB: config.VectorDBConfig{
			Provider:   "milvus",
			Host:       "localhost",
			Port:       19530,
			Database:   "default",
			Collection: "test_collection",
		},
	}

	ragClient, err := NewRAGClient(config)
	if err != nil {
		return nil, err
	}

	return ragClient, nil

}

func TestNewRAGClient(t *testing.T) {
	_, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}
}

func TestRAGClient_CreateChunkFromText(t *testing.T) {
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}
	text := "The multi-agent interaction technology competition based on the openKylin desktop environment aims to promote the development of agent applications on the openKylin open-source OS, using the Kirin AI inference framework and the UKUI desktop environment. These applications should have autonomous planning and decision-making capabilities, access to system resources, and the ability to call system and desktop environment interfaces and tools, with memory functions. They should also be able to collaborate with other agent applications. The competition aims to deeply explore the integration of operating systems and AI and help enhance the international competitiveness of domestic open-source operating systems."
	chunkName := "test_chunk3"
	docs, err := ragClient.CreateChunkFromText(text, chunkName)
	if err != nil {
		t.Errorf("CreateChunkFromText() error = %v", err)
		return
	}
	if len(docs) != 1 {
		t.Errorf("CreateChunkFromText() docs len = %d, want 1", len(docs))
		return
	}

}

func TestRAGClient_ListChunks(t *testing.T) {
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}

	docs, err := ragClient.ListChunks()
	if err != nil {
		t.Errorf("ListChunks() error = %v", err)
		return
	}
	if len(docs) == 0 {
		t.Errorf("ListChunks() docs len = %d, want > 0", len(docs))
		return
	}
}

func TestRAGClient_DeleteChunk(t *testing.T) {
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}

	chunk_id := "63ee25d7-41b9-4455-8066-075ca5c803b2"
	err = ragClient.DeleteChunk(chunk_id)
	if err != nil {
		t.Errorf("DeleteChunk() error = %v", err)
		return
	}
}

func TestRAGClient_SearchChunks(t *testing.T) {
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}
	topk := 2
	threshold := 0.5
	query := "multi-agent"
	docs, err := ragClient.SearchChunks(query, topk, threshold)
	if err != nil {
		t.Errorf("SearchChunks() error = %v", err)
		return
	}
	if len(docs) != topk {
		t.Errorf("SearchChunks() docs len = %d, want %d", len(docs), topk)
		return
	}

}

func TestRAGClient_Chat(t *testing.T) {
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}
	// query := "Who is the individual associated with the cryptocurrency industry facing a criminal trial on fraud and conspiracy charges, as reported by both The Verge and TechCrunch, and is accused by prosecutors of committing fraud for personal gain?"
	// query := "Which individual is implicated in both inflating the value of a Manhattan apartment to a figure not yet achieved in New York City's real estate history, according to 'Fortune', and is also accused of adjusting this apartment's valuation to compensate for a loss in another asset's worth, as reported by 'The Age'?"
	// query := "Who is the figure associated with generative AI technology whose departure from OpenAI was considered shocking according to Fortune, and is also the subject of a prevailing theory suggesting a lack of full truthfulness with the board as reported by TechCrunch?"
	// query := "Do the TechCrunch article on software companies and the Hacker News article on The Epoch Times both report an increase in revenue related to payment and subscription models, respectively?"
	query := "Which online betting platform provides a welcome bonus of up to $1000 in bonus bets for new customers' first losses, runs NBA betting promotions, and is anticipated to extend the same sign-up offer to new users in Vermont, as reported by both CBSSports.com and Sporting News?"
	resp, err := ragClient.Chat(query)
	if err != nil {
		t.Errorf("Chat() error = %v", err)
		return
	}
	if resp == "" {
		t.Errorf("Chat() resp = %s, want not empty", resp)
		return
	}
	t.Logf("Chat() resp = %s", resp)
}

func TestRAGClient_LoadChunks(t *testing.T) {
	t.Logf("TestRAGClient_LoadChunks")
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}
	// load json output/corpus.json and then call ragclient CreateChunkFromText to insert chunks
	file, err := os.Open("/dataset/corpus.json")
	if err != nil {
		t.Errorf("LoadData() error = %v", err)
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var data []struct {
		Body  string `json:"body"`
		Title string `json:"title"`
		Url   string `json:"url"`
	}
	if err := decoder.Decode(&data); err != nil {
		t.Errorf("LoadData() error = %v", err)
		return
	}

	for _, item := range data {
		t.Logf("LoadData() url = %s", item.Url)
		t.Logf("LoadData() title = %s", item.Title)
		t.Logf("LoadData() len body = %d", len(item.Body))
		chunks, err := ragClient.CreateChunkFromText(item.Body, item.Title)
		if err != nil {
			t.Errorf("LoadData() error = %v", err)
			continue
		} else {
			t.Logf("LoadData() chunks len = %d", len(chunks))
		}
	}
	t.Logf("TestRAGClient_LoadChunks done")
}
