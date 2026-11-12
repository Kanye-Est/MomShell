package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/momshell/backend/internal/config"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/openai"
	"github.com/pgvector/pgvector-go"
)

type RAGService struct {
	client  *openai.Client
	ragRepo *repository.RAGRepo
	cfg     ragConfig
}

// ragConfig holds RAG tuning parameters.
type ragConfig struct {
	SimilarityThreshold float64
	TopK                int
	RerankEnabled       bool
}

func NewRAGService(client *openai.Client, ragRepo *repository.RAGRepo, cfg *config.Config) *RAGService {
	return &RAGService{
		client:  client,
		ragRepo: ragRepo,
		cfg: ragConfig{
			SimilarityThreshold: cfg.RAGSimilarityThreshold,
			TopK:                cfg.RAGTopK,
			RerankEnabled:       cfg.RAGRerankEnabled,
		},
	}
}

// IndexText creates and stores an embedding for the given text.
func (s *RAGService) IndexText(ctx context.Context, source model.KnowledgeSource, sourceID string, userID *string, content string) error {
	if !s.client.Enabled() {
		return nil
	}
	exists, err := s.ragRepo.Exists(source, sourceID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	embedding, err := s.client.CreateEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	ke := &model.KnowledgeEmbedding{
		Source:    source,
		SourceID:  sourceID,
		UserID:    userID,
		Content:   content,
		Embedding: pgvector.NewVector(embedding),
	}

	return s.ragRepo.CreateEmbedding(ke)
}

// queryTransformResult holds the LLM-extracted keywords and rewritten query.
type queryTransformResult struct {
	Keywords       string `json:"keywords"`
	RewrittenQuery string `json:"rewritten_query"`
}

// transformQuery uses a single LLM call to extract keywords and rewrite the query.
func (s *RAGService) transformQuery(ctx context.Context, query string) queryTransformResult {
	prompt := fmt.Sprintf(`分析以下用户问题，提取关键词并重写为更适合语义搜索的查询。
只返回JSON，格式：{"keywords":"关键词1 关键词2","rewritten_query":"重写后的查询"}
不要返回其他内容。

用户问题：%s`, query)

	messages := []openai.Message{
		{Role: "system", Content: "你是一个查询分析助手。只返回JSON。"},
		{Role: "user", Content: prompt},
	}

	rawContent, err := s.client.Chat(ctx, messages)
	if err != nil {
		log.Printf("[RAG] query transform failed: %v, using original query", err)
		return queryTransformResult{Keywords: query, RewrittenQuery: query}
	}

	var result queryTransformResult
	rawContent = strings.TrimSpace(rawContent)
	if err := json.Unmarshal([]byte(rawContent), &result); err != nil {
		// Try extracting JSON from response
		start := strings.Index(rawContent, "{")
		end := strings.LastIndex(rawContent, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(rawContent[start:end+1]), &result); err2 != nil {
				log.Printf("[RAG] query transform parse failed: %v, using original", err2)
				return queryTransformResult{Keywords: query, RewrittenQuery: query}
			}
		} else {
			log.Printf("[RAG] query transform parse failed: %v, using original", err)
			return queryTransformResult{Keywords: query, RewrittenQuery: query}
		}
	}

	if result.Keywords == "" {
		result.Keywords = query
	}
	if result.RewrittenQuery == "" {
		result.RewrittenQuery = query
	}

	log.Printf("[RAG] query transform: keywords=%q rewritten=%q", result.Keywords, result.RewrittenQuery)
	return result
}

// rrfCandidate is used for Reciprocal Rank Fusion scoring.
type rrfCandidate struct {
	embedding  model.KnowledgeEmbedding
	vectorRank int // -1 means not present
	kwRank     int // -1 means not present
	rrfScore   float64
}

const rrfK = 60 // RRF constant

// collectVectorCandidates adds vector search results to the candidate map, filtering by threshold.
func (s *RAGService) collectVectorCandidates(candidates map[string]*rrfCandidate, vectorResults []repository.ScoredEmbedding) {
	for rank, vr := range vectorResults {
		if vr.Distance > s.cfg.SimilarityThreshold {
			continue
		}
		candidates[vr.ID] = &rrfCandidate{
			embedding:  vr.KnowledgeEmbedding,
			vectorRank: rank,
			kwRank:     -1,
		}
	}
}

// collectKeywordCandidates merges keyword search results into the candidate map.
func collectKeywordCandidates(candidates map[string]*rrfCandidate, kwResults []repository.KeywordResult) {
	for rank, kr := range kwResults {
		if c, ok := candidates[kr.ID]; ok {
			c.kwRank = rank
		} else {
			candidates[kr.ID] = &rrfCandidate{
				embedding:  kr.KnowledgeEmbedding,
				vectorRank: -1,
				kwRank:     rank,
			}
		}
	}
}

// computeRRFScores calculates Reciprocal Rank Fusion scores for all candidates.
func computeRRFScores(candidates map[string]*rrfCandidate) {
	for _, c := range candidates {
		if c.vectorRank >= 0 {
			c.rrfScore += 0.7 / float64(rrfK+c.vectorRank)
		}
		if c.kwRank >= 0 {
			c.rrfScore += 0.3 / float64(rrfK+c.kwRank)
		}
	}
}

// resolveMatchType determines the match type based on which retrieval paths found the candidate.
func resolveMatchType(vectorRank, kwRank int) model.MatchType {
	if vectorRank >= 0 && kwRank < 0 {
		return model.MatchVector
	}
	if vectorRank < 0 && kwRank >= 0 {
		return model.MatchKeyword
	}
	return model.MatchHybrid
}

// sortAndConvertCandidates sorts candidates by RRF score and converts to ScoredResult.
func sortAndConvertCandidates(candidates map[string]*rrfCandidate) []model.ScoredResult {
	sorted := make([]*rrfCandidate, 0, len(candidates))
	for _, c := range candidates {
		sorted = append(sorted, c)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].rrfScore > sorted[j].rrfScore
	})

	results := make([]model.ScoredResult, 0, len(sorted))
	for _, c := range sorted {
		results = append(results, model.ScoredResult{
			KnowledgeEmbedding: c.embedding,
			Score:              c.rrfScore,
			MatchType:          resolveMatchType(c.vectorRank, c.kwRank),
		})
	}
	return results
}

// hybridRetrieve performs vector + keyword search and merges via RRF.
func (s *RAGService) hybridRetrieve(ctx context.Context, qt queryTransformResult, userID *string) ([]model.ScoredResult, error) {
	embedding, err := s.client.CreateEmbedding(ctx, qt.RewrittenQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	const candidateLimit = 20

	vectorResults, vecErr := s.ragRepo.FindSimilarWithScore(embedding, candidateLimit, userID)
	if vecErr != nil {
		log.Printf("[RAG] vector search failed: %v", vecErr)
	}

	kwResults, kwErr := s.ragRepo.FindByKeyword(qt.Keywords, candidateLimit, userID)
	if kwErr != nil {
		log.Printf("[RAG] keyword search failed: %v", kwErr)
	}

	candidates := make(map[string]*rrfCandidate)
	s.collectVectorCandidates(candidates, vectorResults)
	collectKeywordCandidates(candidates, kwResults)
	computeRRFScores(candidates)

	return sortAndConvertCandidates(candidates), nil
}

// applyRerankScores updates results with rerank scores and re-sorts by score descending.
func applyRerankScores(results []model.ScoredResult, scores []openai.RerankResult) []model.ScoredResult {
	scoreMap := make(map[string]float64, len(scores))
	for _, s := range scores {
		scoreMap[s.ID] = s.Score
	}
	for i := range results {
		if score, ok := scoreMap[results[i].ID]; ok {
			results[i].Score = score
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

// rerank uses the LLM to re-score candidates by relevance.
func (s *RAGService) rerank(ctx context.Context, query string, results []model.ScoredResult) []model.ScoredResult {
	if !s.cfg.RerankEnabled || len(results) == 0 {
		return results
	}

	candidates := make([]openai.RerankCandidate, len(results))
	for i, r := range results {
		candidates[i] = openai.RerankCandidate{
			ID:      r.ID,
			Content: r.Content,
		}
	}

	scores, err := s.client.Rerank(ctx, query, candidates)
	if err != nil {
		log.Printf("[RAG] rerank failed, using RRF scores: %v", err)
		return results
	}

	return applyRerankScores(results, scores)
}

// Search is the main entry point for RAG retrieval.
// It performs query transformation, hybrid retrieval, reranking, and top-K filtering.
func (s *RAGService) Search(ctx context.Context, query string, userID *string) ([]model.ScoredResult, error) {
	if !s.client.Enabled() {
		return nil, nil
	}
	// Step 1: Query transformation
	qt := s.transformQuery(ctx, query)

	// Step 2: Hybrid retrieval (vector + keyword with RRF merge)
	results, err := s.hybridRetrieve(ctx, qt, userID)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	// Step 3: Reranking
	results = s.rerank(ctx, query, results)

	// Step 4: Top-K filtering
	if len(results) > s.cfg.TopK {
		results = results[:s.cfg.TopK]
	}

	log.Printf("[RAG] search complete: query=%q candidates=%d returned=%d",
		query, len(results), min(len(results), s.cfg.TopK))

	return results, nil
}

// FormatContext returns a formatted string of retrieved knowledge for the LLM prompt.
func (s *RAGService) FormatContext(results []model.ScoredResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, res := range results {
		relevance := "★"
		if res.Score > 5 {
			relevance = "★★★"
		} else if res.Score > 2 {
			relevance = "★★"
		}
		fmt.Fprintf(&sb, "[%s|%s|%s] %s\n", res.Source, res.MatchType, relevance, res.Content)
	}
	return sb.String()
}

// FormatContextLegacy formats results in the old format (for backward compatibility during transition).
func (s *RAGService) FormatContextLegacy(results []model.KnowledgeEmbedding) string {
	if len(results) == 0 {
		return ""
	}

	var contextStr string
	for _, res := range results {
		contextStr += fmt.Sprintf("[%s] %s\n", res.Source, res.Content)
	}
	return contextStr
}

// BackgroundReindexAll reindexes existing content in the background.
func (s *RAGService) BackgroundReindexAll(questionRepo *repository.QuestionRepo, answerRepo *repository.AnswerRepo, whisperRepo *repository.WhisperRepo, echoRepo *repository.EchoRepo) {
	go func() {
		ctx := context.Background()
		log.Println("[RAGService] starting background reindexing...")

		// Questions
		questions, _, err := questionRepo.FindAll("", "", model.StatusPublished, "created_at", "asc", 0, 1000)
		if err == nil {
			for _, q := range questions {
				_ = s.IndexText(ctx, model.SourceQuestion, q.ID, nil, q.Title+"\n"+q.Content)
			}
		}

		// Answers
		answers, _, err := answerRepo.FindByQuestionID("", nil, "created_at", "asc", 0, 1000)
		if err == nil {
			for _, a := range answers {
				if a.Status == model.StatusPublished {
					_ = s.IndexText(ctx, model.SourceAnswer, a.ID, nil, a.Content)
				}
			}
		}

		log.Println("[RAGService] background reindexing completed.")
	}()
}
