package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// FAQChunkMetadata defines the shape of a FAQ entry inside Chunk.Metadata.
type FAQChunkMetadata struct {
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions,omitempty"`
	NegativeQuestions []string       `json:"negative_questions,omitempty"`
	Answers           []string       `json:"answers,omitempty"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy,omitempty"`
	Version           int            `json:"version,omitempty"`
	Source            string         `json:"source,omitempty"`
}

// GeneratedQuestion is a single AI-generated question.
type GeneratedQuestion struct {
	ID              string `json:"id"`
	Question        string `json:"question"`
	ContentRevision *int   `json:"content_revision,omitempty"`
}

const maxGeneratedQuestionSourceIDLength = 64

// GeneratedQuestionSourceID builds the retrieval source identifier for a
// generated question. PostgreSQL stores source_id as varchar(64), while a
// chunk UUID plus a question UUID would be 73 bytes. Preserve the historical
// representation for short IDs and hash only oversized question IDs so
// existing index rows remain addressable by delete/reindex operations.
func GeneratedQuestionSourceID(chunkID, questionID string) string {
	candidate := chunkID + "-" + questionID
	if len(candidate) <= maxGeneratedQuestionSourceIDLength {
		return candidate
	}
	digest := sha256.Sum256([]byte(questionID))
	// UUID chunk IDs use 36 bytes; "-q" plus 24 hex characters keeps the
	// complete identifier at 62 bytes while retaining ample collision space.
	return chunkID + "-q" + hex.EncodeToString(digest[:12])
}

// DocumentChunkMetadata defines the metadata structure of a document chunk.
// It stores enrichment data such as AI-generated questions.
type DocumentChunkMetadata struct {
	// GeneratedQuestions holds the related questions the AI generated for this
	// chunk. They are indexed separately to improve recall.
	GeneratedQuestions []GeneratedQuestion `json:"generated_questions,omitempty"`
	// GeneratedQuestionsRevision ties the questions to Chunk.ContentRevision.
	GeneratedQuestionsRevision int `json:"generated_questions_revision,omitempty"`
}

// IsQuestionCurrent reports whether a generated question was authored for the
// current chunk body. This is advisory metadata for the UI: questions remain
// valid retrieval aliases across chunk edits. Legacy rows fall back to the
// metadata-level revision.
func (m *DocumentChunkMetadata) IsQuestionCurrent(question GeneratedQuestion, chunkRevision int) bool {
	if question.ContentRevision != nil {
		return *question.ContentRevision == chunkRevision
	}
	return m != nil && m.GeneratedQuestionsRevision == chunkRevision
}

// GetQuestionStrings returns the question texts as a string list (kept for older callers).
func (m *DocumentChunkMetadata) GetQuestionStrings() []string {
	if m == nil || len(m.GeneratedQuestions) == 0 {
		return nil
	}
	result := make([]string, len(m.GeneratedQuestions))
	for i, q := range m.GeneratedQuestions {
		result[i] = q.Question
	}
	return result
}

// DocumentMetadata parses the document metadata out of a chunk.
func (c *Chunk) DocumentMetadata() (*DocumentChunkMetadata, error) {
	if c == nil || len(c.Metadata) == 0 {
		return nil, nil
	}
	var meta DocumentChunkMetadata
	if err := json.Unmarshal(c.Metadata, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SetDocumentMetadata sets the document metadata on a chunk.
func (c *Chunk) SetDocumentMetadata(meta *DocumentChunkMetadata) error {
	if c == nil {
		return nil
	}
	if meta == nil {
		c.Metadata = nil
		return nil
	}
	bytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	c.Metadata = JSON(bytes)
	return nil
}

// Sanitize performs basic cleanup of the metadata (trimming whitespace and
// de-duplicating) while preserving the original content.
// It is used for DB storage and does no semantic normalisation.
func (m *FAQChunkMetadata) Sanitize() {
	if m == nil {
		return
	}
	m.StandardQuestion = strings.TrimSpace(m.StandardQuestion)
	m.SimilarQuestions = SanitizeStrings(m.SimilarQuestions)
	m.NegativeQuestions = SanitizeStrings(m.NegativeQuestions)
	m.Answers = SanitizeStrings(m.Answers)
	if m.Version <= 0 {
		m.Version = 1
	}
}

// Normalize returns a normalised copy, used for hash computation and vector
// indexing. The original data is unchanged; a new normalised copy is returned.
func (m *FAQChunkMetadata) Normalize() *FAQChunkMetadata {
	if m == nil {
		return nil
	}
	return &FAQChunkMetadata{
		StandardQuestion:  NormalizeQuestion(m.StandardQuestion),
		SimilarQuestions:  normalizeQuestionStrings(m.SimilarQuestions),
		NegativeQuestions: normalizeQuestionStrings(m.NegativeQuestions),
		Answers:           SanitizeStrings(m.Answers),
		AnswerStrategy:    m.AnswerStrategy,
		Version:           m.Version,
		Source:            m.Source,
	}
}

// SanitizeStrings performs basic cleanup of a string list (TrimSpace plus de-duplication).
func SanitizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	dedup := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		dedup = append(dedup, trimmed)
	}
	if len(dedup) == 0 {
		return nil
	}
	return dedup
}

// FAQMetadata parses the FAQ metadata out of a chunk.
// It returns the original data, with basic cleanup only.
func (c *Chunk) FAQMetadata() (*FAQChunkMetadata, error) {
	if c == nil || len(c.Metadata) == 0 {
		return nil, nil
	}
	var meta FAQChunkMetadata
	if err := json.Unmarshal(c.Metadata, &meta); err != nil {
		return nil, err
	}
	meta.Sanitize()
	return &meta, nil
}

// SetFAQMetadata sets the FAQ metadata on a chunk.
// The DB stores the original data; ContentHash is computed from the normalised data.
func (c *Chunk) SetFAQMetadata(meta *FAQChunkMetadata) error {
	if c == nil {
		return nil
	}
	if meta == nil {
		c.Metadata = nil
		c.ContentHash = ""
		return nil
	}
	// Store to the DB after basic cleanup (the original content is preserved)
	meta.Sanitize()
	bytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	c.Metadata = JSON(bytes)
	// ContentHash is computed from the normalised data and used for de-duplication matching
	normalized := meta.Normalize()
	c.ContentHash = CalculateFAQContentHash(normalized)
	return nil
}

// CalculateFAQContentHash computes the hash of a FAQ entry's content.
// The hash is built from: standard question + similar questions (sorted) +
// negative examples (sorted) + answers (sorted).
// It is used for fast matching and de-duplication.
func CalculateFAQContentHash(meta *FAQChunkMetadata) string {
	if meta == nil {
		return ""
	}

	// Normalize() returns a new copy; the old code discarded the return value.
	normalized := meta.Normalize()
	if normalized == nil {
		return ""
	}

	// Sort the arrays so identical content always produces the same hash
	similarQuestions := make([]string, len(normalized.SimilarQuestions))
	copy(similarQuestions, normalized.SimilarQuestions)
	sort.Strings(similarQuestions)

	negativeQuestions := make([]string, len(normalized.NegativeQuestions))
	copy(negativeQuestions, normalized.NegativeQuestions)
	sort.Strings(negativeQuestions)

	answers := make([]string, len(normalized.Answers))
	copy(answers, normalized.Answers)
	sort.Strings(answers)

	// Build the hash input: standard question + similar questions + negative examples + answers
	var builder strings.Builder
	builder.WriteString(normalized.StandardQuestion)
	builder.WriteString("|")
	builder.WriteString(strings.Join(similarQuestions, ","))
	builder.WriteString("|")
	builder.WriteString(strings.Join(negativeQuestions, ","))
	builder.WriteString("|")
	builder.WriteString(strings.Join(answers, ","))

	// Compute the SHA256 hash
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// AnswerStrategy defines how answers are returned.
type AnswerStrategy string

const (
	// AnswerStrategyAll returns every answer
	AnswerStrategyAll AnswerStrategy = "all"
	// AnswerStrategyRandom returns one answer at random
	AnswerStrategyRandom AnswerStrategy = "random"
)

// FAQEntry is a FAQ entry as returned to the SPA.
type FAQEntry struct {
	ID                int64          `json:"id"`
	ChunkID           string         `json:"chunk_id"`
	KnowledgeID       string         `json:"knowledge_id"`
	KnowledgeBaseID   string         `json:"knowledge_base_id"`
	TagID             int64          `json:"tag_id"`
	TagName           string         `json:"tag_name"`
	IsEnabled         bool           `json:"is_enabled"`
	IsRecommended     bool           `json:"is_recommended"`
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions"`
	NegativeQuestions []string       `json:"negative_questions"`
	Answers           []string       `json:"answers"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy"`
	IndexMode         FAQIndexMode   `json:"index_mode"`
	UpdatedAt         time.Time      `json:"updated_at"`
	CreatedAt         time.Time      `json:"created_at"`
	Score             float64        `json:"score,omitempty"`
	MatchType         MatchType      `json:"match_type,omitempty"`
	ChunkType         ChunkType      `json:"chunk_type"`
	// MatchedQuestion is the actual question text that was matched in FAQ search
	// Could be the standard question or one of the similar questions
	MatchedQuestion string `json:"matched_question,omitempty"`
}

// FAQExportEntry is a FAQ entry in the JSON export, compatible with the import
// format of FAQEntryPayload so that an "export, edit, re-import" cycle works.
// Always keep omitempty when adding a field, so historical export files stay
// compatible.
type FAQExportEntry struct {
	ID                int64          `json:"id"`
	TagName           string         `json:"tag_name,omitempty"`
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions,omitempty"`
	NegativeQuestions []string       `json:"negative_questions,omitempty"`
	Answers           []string       `json:"answers,omitempty"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy,omitempty"`
	IsEnabled         bool           `json:"is_enabled"`
	IsRecommended     bool           `json:"is_recommended"`
}

// FAQEntryPayload is the payload for creating or updating a FAQ entry.
type FAQEntryPayload struct {
	// ID is optional; it pins seq_id during data migration (it must be below the
	// auto-increment start value of 100000000)
	ID                *int64          `json:"id,omitempty"`
	StandardQuestion  string          `json:"standard_question"    binding:"required"`
	SimilarQuestions  []string        `json:"similar_questions"`
	NegativeQuestions []string        `json:"negative_questions"`
	Answers           []string        `json:"answers"`
	AnswerStrategy    *AnswerStrategy `json:"answer_strategy,omitempty"`
	TagID             int64           `json:"tag_id"`
	TagName           string          `json:"tag_name"`
	IsEnabled         *bool           `json:"is_enabled,omitempty"`
	IsRecommended     *bool           `json:"is_recommended,omitempty"`
}

const (
	FAQBatchModeAppend  = "append"
	FAQBatchModeReplace = "replace"
)

// FAQBatchUpsertPayload imports FAQ entries in bulk.
type FAQBatchUpsertPayload struct {
	Entries     []FAQEntryPayload `json:"entries"      binding:"required"`
	Mode        string            `json:"mode"         binding:"oneof=append replace"`
	KnowledgeID string            `json:"knowledge_id"`
	TaskID      string            `json:"task_id"`
	DryRun      bool              `json:"dry_run"`
}

// FAQFailedEntry is an entry that failed import or validation.
type FAQFailedEntry struct {
	Index             int      `json:"index"`
	Reason            string   `json:"reason"`
	FailureType       string   `json:"failure_type,omitempty"`
	IsPartialFailure  bool     `json:"is_partial_failure,omitempty"`
	TagName           string   `json:"tag_name,omitempty"`
	StandardQuestion  string   `json:"standard_question"`
	SimilarQuestions  []string `json:"similar_questions,omitempty"`
	NegativeQuestions []string `json:"negative_questions,omitempty"`
	Answers           []string `json:"answers,omitempty"`
	AnswerAll         bool     `json:"answer_all,omitempty"`
	IsDisabled        bool     `json:"is_disabled,omitempty"`
	// Partial failure details (set when IsPartialFailure is true)
	RemovedSimilarQuestions  []string `json:"removed_similar_questions,omitempty"`
	RemovedNegativeQuestions []string `json:"removed_negative_questions,omitempty"`
}

// FAQMergeDetail summarises how one FAQ was merged into an existing chunk in append mode.
type FAQMergeDetail struct {
	Index            int    `json:"index"`
	StandardQuestion string `json:"standard_question"`
	AnswerChanged    bool   `json:"answer_changed"`
	NewSimilarCount  int    `json:"new_similar_count"`
	NewNegativeCount int    `json:"new_negative_count"`
}

// FAQSuccessEntry carries brief details of a successfully imported entry.
type FAQSuccessEntry struct {
	Index            int    `json:"index"`
	SeqID            int64  `json:"seq_id"`
	TagID            int64  `json:"tag_id,omitempty"`
	TagName          string `json:"tag_name,omitempty"`
	StandardQuestion string `json:"standard_question"`
}

// FAQDryRunResult is the validation result of a dry_run import.
type FAQDryRunResult struct {
	TaskID        string           `json:"task_id,omitempty"`
	Total         int              `json:"total"`
	SuccessCount  int              `json:"success_count"`
	FailedCount   int              `json:"failed_count"`
	FailedEntries []FAQFailedEntry `json:"failed_entries"`
}

// FAQSearchRequest holds the FAQ search request parameters.
type FAQSearchRequest struct {
	QueryText            string  `json:"query_text"             binding:"required"`
	VectorThreshold      float64 `json:"vector_threshold"`
	MatchCount           int     `json:"match_count"`
	FirstPriorityTagIDs  []int64 `json:"first_priority_tag_ids"`
	SecondPriorityTagIDs []int64 `json:"second_priority_tag_ids"`
	OnlyRecommended      bool    `json:"only_recommended"`
}

// UntaggedTagName is the default tag name for entries without a tag
const UntaggedTagName = "Uncategorised"

// FAQEntryFieldsUpdate is a field update for a single FAQ entry.
type FAQEntryFieldsUpdate struct {
	IsEnabled     *bool  `json:"is_enabled,omitempty"`
	IsRecommended *bool  `json:"is_recommended,omitempty"`
	TagID         *int64 `json:"tag_id,omitempty"`
	// More fields can be added later
}

// FAQEntryFieldsBatchUpdate is the request to update FAQ entry fields in bulk.
// Two modes are supported:
//  1. By entry ID: use the ByID field
//  2. By tag: use the ByTag field, applying the same update to every entry under that tag
type FAQEntryFieldsBatchUpdate struct {
	// ByID updates by entry ID; the key is the entry ID (seq_id)
	ByID map[int64]FAQEntryFieldsUpdate `json:"by_id,omitempty"`
	// ByTag updates in bulk by tag; the key is the tag ID (seq_id)
	ByTag map[int64]FAQEntryFieldsUpdate `json:"by_tag,omitempty"`
	// ExcludeIDs lists the IDs to exclude from a ByTag operation (seq_id)
	ExcludeIDs []int64 `json:"exclude_ids,omitempty"`
}

// FAQImportTaskStatus is the status of an import task.
type FAQImportTaskStatus string

const (
	// FAQImportStatusPending represents the pending status of the FAQ import task
	FAQImportStatusPending FAQImportTaskStatus = "pending"
	// FAQImportStatusProcessing represents the processing status of the FAQ import task
	FAQImportStatusProcessing FAQImportTaskStatus = "processing"
	// FAQImportStatusCompleted represents the completed status of the FAQ import task
	FAQImportStatusCompleted FAQImportTaskStatus = "completed"
	// FAQImportStatusFailed represents the failed status of the FAQ import task
	FAQImportStatusFailed FAQImportTaskStatus = "failed"
)

// FAQImportProgress represents the progress of an FAQ import task stored in Redis
// When Status is "completed", the result fields (SkippedCount, ImportMode, ImportedAt, DisplayStatus, ProcessingTime) are populated.
type FAQImportProgress struct {
	TaskID             string              `json:"task_id"`      // UUID for the import task
	KBID               string              `json:"kb_id"`        // Knowledge Base ID
	KnowledgeID        string              `json:"knowledge_id"` // FAQ Knowledge ID
	Status             FAQImportTaskStatus `json:"status"`       // Task status
	Progress           int                 `json:"progress"`     // 0-100 percentage
	Total              int                 `json:"total"`        // Total entries to import
	Processed          int                 `json:"processed"`    // Entries processed so far
	SuccessCount       int                 `json:"success_count"`
	FailedCount        int                 `json:"failed_count"`
	PartialFailedCount int                 `json:"partial_failed_count,omitempty"`
	SkippedCount       int                 `json:"skipped_count,omitempty"`
	FailedEntries      []FAQFailedEntry    `json:"failed_entries,omitempty"`
	FailedEntriesURL   string              `json:"failed_entries_url,omitempty"`
	SuccessEntries     []FAQSuccessEntry   `json:"success_entries,omitempty"`
	ValidEntryIndices  []int               `json:"valid_entry_indices,omitempty"`
	MergeEntryIndices  []int               `json:"merge_entry_indices,omitempty"`
	MergedCount        int                 `json:"merged_count,omitempty"`
	AddedCount         int                 `json:"added_count,omitempty"`
	MergeDetails       []FAQMergeDetail    `json:"merge_details,omitempty"`
	Message            string              `json:"message"`    // Status message
	Error              string              `json:"error"`      // Error message if failed
	CreatedAt          int64               `json:"created_at"` // Task creation timestamp
	UpdatedAt          int64               `json:"updated_at"` // Last update timestamp
	DryRun             bool                `json:"dry_run,omitempty"`

	// Result fields (populated when Status == "completed")
	ImportMode     string    `json:"import_mode,omitempty"`
	ImportedAt     time.Time `json:"imported_at,omitempty"`
	DisplayStatus  string    `json:"display_status,omitempty"`
	ProcessingTime int64     `json:"processing_time,omitempty"`
}

// FAQImportMetadata is the FAQ import task information stored in Knowledge.Metadata.
// Deprecated: Use FAQImportProgress with Redis storage instead
type FAQImportMetadata struct {
	ImportProgress  int `json:"import_progress"` // 0-100
	ImportTotal     int `json:"import_total"`
	ImportProcessed int `json:"import_processed"`
}

// FAQImportResult holds the statistics of a completed FAQ import.
// It is persisted independently of the progress state and stays until the next
// import replaces it.
type FAQImportResult struct {
	// Import statistics
	TotalEntries       int `json:"total_entries"`
	SuccessCount       int `json:"success_count"`
	FailedCount        int `json:"failed_count"`
	PartialFailedCount int `json:"partial_failed_count"`
	SkippedCount       int `json:"skipped_count"`
	MergedCount        int `json:"merged_count"`
	AddedCount         int `json:"added_count"`

	// Import mode and timing
	ImportMode string    `json:"import_mode"`
	ImportedAt time.Time `json:"imported_at"`
	TaskID     string    `json:"task_id"`

	// Failure detail URL (a download link, provided when there are many failed entries)
	FailedEntriesURL string `json:"failed_entries_url,omitempty"`

	// Display control
	DisplayStatus string `json:"display_status"`

	// Extra statistics
	ProcessingTime int64 `json:"processing_time"`
}

// ToJSON converts the metadata to JSON type.
func (m *FAQImportMetadata) ToJSON() (JSON, error) {
	if m == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return JSON(bytes), nil
}

// ToJSON converts the import result to JSON type.
func (r *FAQImportResult) ToJSON() (JSON, error) {
	if r == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return JSON(bytes), nil
}

// ParseFAQImportMetadata parses FAQ import metadata from Knowledge.
func ParseFAQImportMetadata(k *Knowledge) (*FAQImportMetadata, error) {
	if k == nil || len(k.Metadata) == 0 {
		return nil, nil
	}
	var metadata FAQImportMetadata
	if err := json.Unmarshal(k.Metadata, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// normalizeQuestionStrings normalises a list of questions: full-width to
// half-width conversion, trailing punctuation removal, whitespace collapsing and
// de-duplication.
func normalizeQuestionStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	dedup := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		normalized := NormalizeQuestion(v)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		dedup = append(dedup, normalized)
	}
	if len(dedup) == 0 {
		return nil
	}
	return dedup
}

// multiSpaceRegex matches runs of consecutive whitespace.
var multiSpaceRegex = regexp.MustCompile(`\s+`)

// urlRegex matches a URL.
var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

// URLNormMode defines the URL normalisation mode.
type URLNormMode int

const (
	// URLRemove strips the URL entirely
	URLRemove URLNormMode = iota
	// URLPlaceholder replaces it with the <URL> placeholder
	URLPlaceholder
	// URLKeepDomain keeps the domain only
	URLKeepDomain
	// URLKeepDomainAndPath keeps the domain and the path
	URLKeepDomainAndPath
)

// NormalizeQuestion normalises question text to improve vector match recall.
// Processing order, for reference: query = convert_st(trim_url(query.lower().strip().strip("？。，；、：""！?.,;!:'\"")), 1)
//  1. Trim leading and trailing whitespace
//  2. Remove URLs
//  3. Lower-case
//  4. Strip leading and trailing punctuation
//  5. Convert full-width symbols to half-width
//  6. Smart whitespace handling (drop spaces between CJK characters, keep them
//     between letters and digits)
func NormalizeQuestion(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}

	// 1. Remove URLs
	q = trimURL(q)

	// 2. Lower-case (effective for Latin text)
	q = strings.ToLower(q)

	// 3. Strip leading and trailing punctuation
	q = strings.Trim(q, `？。，；、：""！?.,;!:'""`)

	// 4. Convert full-width characters to half-width
	q = toHalfWidth(q)

	// 5. Smart whitespace: drop spaces between CJK characters, keep them between letters and digits
	q = normalizeSpaces(q)

	return strings.TrimSpace(q)
}

// normalizeSpaces handles whitespace intelligently.
// Rules:
//   - drop redundant spaces in CJK context
//   - keep the spaces that matter between letters and digits
//
// For example, spaces separating CJK characters are removed, while
// "iphone 15" keeps its space.
func normalizeSpaces(s string) string {
	// Collapse runs of spaces into a single space first
	s = multiSpaceRegex.ReplaceAllString(s, " ")

	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(s))

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Not a space: write it straight through
		if r != ' ' {
			builder.WriteRune(r)
			continue
		}

		// Space: decide whether to keep it from the surrounding characters.
		// Find the previous non-space character
		var prevRune rune
		if i > 0 {
			prevRune = runes[i-1]
		}

		// Find the next non-space character
		var nextRune rune
		for j := i + 1; j < len(runes); j++ {
			if runes[j] != ' ' {
				nextRune = runes[j]
				break
			}
		}

		// Keep the space only when both sides are ASCII letters or digits;
		// drop it otherwise (including around CJK characters)
		if isASCIIAlphaNum(prevRune) && isASCIIAlphaNum(nextRune) {
			builder.WriteRune(' ')
		}
		// Otherwise skip the space
	}

	return builder.String()
}

// isASCIIAlphaNum reports whether the rune is an ASCII letter or digit.
func isASCIIAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// trimURL removes URLs from a string (using the default URLRemove mode).
func trimURL(s string) string {
	return NormalizeURL(s, URLRemove)
}

// NormalizeURL processes the URLs in a text according to the given mode.
func NormalizeURL(text string, mode URLNormMode) string {
	return urlRegex.ReplaceAllStringFunc(text, func(raw string) string {
		switch mode {
		case URLRemove:
			return ""
		case URLPlaceholder:
			return "<URL>"
		case URLKeepDomain:
			domain, _ := parseURL(raw)
			if domain != "" {
				return domain
			}
			return "<URL>"
		case URLKeepDomainAndPath:
			domain, path := parseURL(raw)
			if domain != "" {
				return domain + path
			}
			return "<URL>"
		default:
			return "<URL>"
		}
	})
}

// parseURL parses a URL and returns its domain and path.
func parseURL(raw string) (domain, path string) {
	// Strip the scheme prefix
	u := raw
	if strings.HasPrefix(u, "https://") {
		u = u[8:]
	} else if strings.HasPrefix(u, "http://") {
		u = u[7:]
	}

	// Split the domain from the path
	slashIdx := strings.Index(u, "/")
	if slashIdx == -1 {
		// No path: the whole string is the domain (possibly with query parameters)
		queryIdx := strings.Index(u, "?")
		if queryIdx != -1 {
			domain = u[:queryIdx]
		} else {
			domain = u
		}
		return domain, ""
	}

	domain = u[:slashIdx]
	path = u[slashIdx:]

	// Strip the query string and fragment
	if queryIdx := strings.Index(path, "?"); queryIdx != -1 {
		path = path[:queryIdx]
	}
	if fragIdx := strings.Index(path, "#"); fragIdx != -1 {
		path = path[:fragIdx]
	}

	return domain, path
}

// toHalfWidth converts full-width characters to half-width.
// It mainly handles the full-width space and full-width ASCII characters,
// including punctuation.
func toHalfWidth(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))

	for _, r := range s {
		switch {
		// full-width space -> half-width space
		case r == '\u3000':
			builder.WriteRune(' ')
		// full-width ASCII characters (range 0xFF01-0xFF5E) -> half-width (0x0021-0x007E)
		case r >= 0xFF01 && r <= 0xFF5E:
			builder.WriteRune(r - 0xFF01 + 0x21)
		// everything else is unchanged
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// NormalizeQueryText normalises search query text.
// It applies exactly the same processing as NormalizeQuestion, for normalising
// the query text at search time.
func NormalizeQueryText(q string) string {
	return NormalizeQuestion(q)
}

// IsChineseChar reports whether the rune is a Chinese character.
func IsChineseChar(r rune) bool {
	return unicode.Is(unicode.Han, r)
}
