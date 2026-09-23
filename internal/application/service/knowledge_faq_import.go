package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/application/access"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/application/repository"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/application/service/retriever"
	werrors "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/errors"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/logger"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/embedding"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/tracing/langfuse"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	secutils "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/utils"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// UpsertFAQEntries imports or appends FAQ entries asynchronously.
// Returns task ID (UUID) for tracking import progress.
func (s *knowledgeService) UpsertFAQEntries(ctx context.Context,
	kbID string, payload *types.FAQBatchUpsertPayload,
) (string, error) {
	if payload == nil || len(payload.Entries) == 0 {
		return "", werrors.NewBadRequestError("FAQ entries must not be empty")
	}
	if payload.Mode == "" {
		payload.Mode = types.FAQBatchModeAppend
	}
	if payload.Mode != types.FAQBatchModeAppend && payload.Mode != types.FAQBatchModeReplace {
		return "", werrors.NewBadRequestError("Mode must be either append or replace")
	}

	kb, ctx, err := s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return "", err
	}
	if err := s.validateFAQImportTags(ctx, kb, payload.Entries); err != nil {
		return "", err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	// Use the supplied TaskID, or generate an enhanced one when none was supplied.
	// A client-supplied task_id ends up in file names and Redis keys, so it must be an
	// identifier with no path separators.
	taskID := strings.TrimSpace(payload.TaskID)
	if taskID == "" {
		taskID = secutils.GenerateTaskID("faq_import", tenantID, kbID)
	} else if err := secutils.ValidateTaskID(taskID); err != nil {
		return "", werrors.NewBadRequestError("task_id has an invalid format")
	}

	var knowledgeID string

	runningTaskID, err := s.getRunningFAQImportTaskID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to check running import task: %v", err)
		// A failed check does not block the import; carry on.
	} else if runningTaskID != "" {
		logger.Warnf(ctx, "Import task already running for KB %s: %s", kbID, runningTaskID)
		return "", werrors.NewBadRequestError(fmt.Sprintf("An import task is already running for this knowledge base (task ID: %s). Please wait for it to finish and try again", runningTaskID))
	}

	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return "", fmt.Errorf("failed to ensure FAQ knowledge: %w", err)
	}
	knowledgeID = faqKnowledge.ID

	enqueuedAt := time.Now().Unix()
	instanceID := uuid.NewString()
	runningInfoSet := false
	enqueueSucceeded := false

	if err := s.setRunningFAQImportInfo(ctx, kbID, &runningFAQImportInfo{
		TaskID:     taskID,
		EnqueuedAt: enqueuedAt,
		InstanceID: instanceID,
	}); err != nil {
		logger.Errorf(ctx, "Failed to set running FAQ import task info: %v", err)
		// Does not affect task execution; carry on.
	} else {
		runningInfoSet = true
	}
	defer func() {
		if !runningInfoSet || enqueueSucceeded {
			return
		}
		if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, kbID, taskID, instanceID, enqueuedAt); clearErr != nil {
			logger.Warnf(ctx, "Failed to clear FAQ import running info after setup failure: %v", clearErr)
		}
	}()

	progress := &types.FAQImportProgress{
		TaskID:        taskID,
		KBID:          kbID,
		KnowledgeID:   knowledgeID,
		Status:        types.FAQImportStatusPending,
		Progress:      0,
		Total:         len(payload.Entries),
		Processed:     0,
		SuccessCount:  0,
		FailedCount:   0,
		FailedEntries: make([]types.FAQFailedEntry, 0),
		Message:       "Task created, waiting to be processed",
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
		DryRun:        payload.DryRun,
	}
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Errorf(ctx, "Failed to initialize FAQ import task status: %v", err)
		return "", fmt.Errorf("failed to initialize task: %w", err)
	}

	logger.Infof(ctx, "FAQ import task initialized: %s, kb_id: %s, total entries: %d, dry_run: %v",
		taskID, kbID, len(payload.Entries), payload.DryRun)

	// Enqueue FAQ import task to Asynq
	logger.Info(ctx, "Enqueuing FAQ import task to Asynq")

	taskPayload := types.FAQImportPayload{
		TenantID:    tenantID,
		TaskID:      taskID,
		KBID:        kbID,
		KnowledgeID: knowledgeID,
		Mode:        payload.Mode,
		DryRun:      payload.DryRun,
		EnqueuedAt:  enqueuedAt,
		InstanceID:  instanceID,
		Initiator:   types.TaskInitiatorFromContext(ctx),
	}

	// Threshold: use object storage beyond 200 entries, or 50KB once serialised.
	const (
		entryCountThreshold  = 200
		payloadSizeThreshold = 50 * 1024 // 50KB
	)

	entryCount := len(payload.Entries)
	if entryCount > entryCountThreshold {
		entriesData, err := json.Marshal(payload.Entries)
		if err != nil {
			logger.Errorf(ctx, "Failed to marshal FAQ entries: %v", err)
			return "", fmt.Errorf("failed to marshal entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries size: %d bytes, uploading to object storage", len(entriesData))

		// Upload to the private (primary) bucket; cleaned up once the task finishes.
		fileName, err := faqImportEntriesFileName(taskID, enqueuedAt)
		if err != nil {
			return "", fmt.Errorf("invalid task id for object name: %w", err)
		}
		entriesURL, err := s.fileSvc.SaveBytes(ctx, entriesData, tenantID, fileName, false)
		if err != nil {
			logger.Errorf(ctx, "Failed to upload FAQ entries to object storage: %v", err)
			return "", fmt.Errorf("failed to upload entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries uploaded to: %s", entriesURL)
		taskPayload.EntriesURL = entriesURL
		taskPayload.EntryCount = entryCount
	} else {
		taskPayload.Entries = payload.Entries
	}

	langfuse.InjectTracing(ctx, &taskPayload)
	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal FAQ import task payload: %v", err)
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	if len(payloadBytes) > payloadSizeThreshold && taskPayload.EntriesURL == "" {
		// The payload is too large but has not been uploaded yet, so upload it now.
		entriesData, _ := json.Marshal(payload.Entries)
		fileName, nameErr := faqImportEntriesFileName(taskID, enqueuedAt)
		if nameErr != nil {
			return "", fmt.Errorf("invalid task id for object name: %w", nameErr)
		}
		entriesURL, err := s.fileSvc.SaveBytes(ctx, entriesData, tenantID, fileName, false)
		if err != nil {
			logger.Errorf(ctx, "Failed to upload FAQ entries to object storage: %v", err)
			return "", fmt.Errorf("failed to upload entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries uploaded to (size exceeded): %s", entriesURL)
		taskPayload.Entries = nil
		taskPayload.EntriesURL = entriesURL
		taskPayload.EntryCount = entryCount

		payloadBytes, _ = json.Marshal(taskPayload)
	}

	logger.Infof(ctx, "FAQ import task payload size: %d bytes", len(payloadBytes))

	maxRetry := 5
	if payload.DryRun {
		maxRetry = 3 // fewer retries for a dry run
	}

	// taskID:instanceID is the unique asynq task identifier.
	asynqTaskID := fmt.Sprintf("%s:%s", taskID, instanceID)

	task := asynq.NewTask(
		types.TypeFAQImport,
		payloadBytes,
		asynq.TaskID(asynqTaskID),
		asynq.Queue(types.QueueMaintenance),
		asynq.MaxRetry(maxRetry),
		asynq.Timeout(2*time.Hour),
	)
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue FAQ import task: %v", err)
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}
	logger.Infof(ctx, "Enqueued FAQ import task: id=%s queue=%s task_id=%s dry_run=%v", info.ID, info.Queue, taskID, payload.DryRun)
	enqueueSucceeded = true

	if !payload.DryRun {
		recordKBActivity(ctx, s.audit, tenantID, kbID, types.AuditActionFAQImportStarted,
			"faq_entry", knowledgeID, types.AuditOutcomeAccepted,
			map[string]any{
				"task_id": taskID, "mode": payload.Mode, "total": len(payload.Entries),
				"trigger": kbActivityTrigger(ctx), "processing_status": "pending",
			})
	}

	return taskID, nil
}

func faqImportEntriesFileName(taskID string, enqueuedAt int64) (string, error) {
	return secutils.SafeFileName(fmt.Sprintf("faq_import_entries_%s_%d.json", taskID, enqueuedAt))
}

// generateFailedEntriesCSV builds the CSV of failed entries and uploads it.
func (s *knowledgeService) generateFailedEntriesCSV(ctx context.Context,
	tenantID uint64, taskID string, failedEntries []types.FAQFailedEntry,
) (string, error) {
	var buf strings.Builder

	// Write a BOM so Excel detects UTF-8 correctly.
	buf.WriteString("\xEF\xBB\xBF")

	buf.WriteString("Error Reason,Tag (required),Question (required),Similar Questions (optional - separate with ##),Negative Questions (optional - separate with ##),Bot Answers (required - separate with ##),Reply All (optional - default FALSE),Disabled (optional - default FALSE)\n")

	for _, entry := range failedEntries {
		// CSV escaping: wrap in quotes and double the inner quotes when the value contains
		// a comma, a quote or a newline.
		reason := csvEscape(entry.Reason)
		tagName := csvEscape(entry.TagName)
		standardQ := csvEscape(entry.StandardQuestion)
		similarQs := ""
		if len(entry.SimilarQuestions) > 0 {
			similarQs = csvEscape(strings.Join(entry.SimilarQuestions, "##"))
		}
		negativeQs := ""
		if len(entry.NegativeQuestions) > 0 {
			negativeQs = csvEscape(strings.Join(entry.NegativeQuestions, "##"))
		}
		answers := ""
		if len(entry.Answers) > 0 {
			answers = csvEscape(strings.Join(entry.Answers, "##"))
		}
		answerAll := "false"
		if entry.AnswerAll {
			answerAll = "true"
		}
		isDisabled := "false"
		if entry.IsDisabled {
			isDisabled = "true"
		}

		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
			reason, tagName, standardQ, similarQs, negativeQs, answers, answerAll, isDisabled))
	}

	// Upload the CSV to temporary storage (auto-expiring).
	fileName, err := secutils.SafeFileName(fmt.Sprintf("faq_dryrun_failed_%s.csv", taskID))
	if err != nil {
		return "", fmt.Errorf("invalid task id for object name: %w", err)
	}
	filePath, err := s.fileSvc.SaveBytes(ctx, []byte(buf.String()), tenantID, fileName, true)
	if err != nil {
		return "", fmt.Errorf("failed to save CSV file: %w", err)
	}

	fileURL, err := s.fileSvc.GetFileURL(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file URL: %w", err)
	}

	logger.Infof(ctx, "Generated failed entries CSV: %s, entries: %d", fileURL, len(failedEntries))
	return fileURL, nil
}

// csvEscape escapes a CSV field.
func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		// Double the inner quotes and wrap the whole field in quotes.
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// saveFAQImportResultToDatabase persists the FAQ import result statistics.
func (s *knowledgeService) saveFAQImportResultToDatabase(ctx context.Context,
	payload *types.FAQImportPayload, progress *types.FAQImportProgress, originalTotalEntries int,
) error {
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, payload.KnowledgeID)
	if err != nil {
		return fmt.Errorf("failed to get FAQ knowledge: %w", err)
	}

	// Skipped entries = total - fully successful - partially failed - fully failed.
	skippedCount := originalTotalEntries - progress.SuccessCount - progress.PartialFailedCount - progress.FailedCount
	if skippedCount < 0 {
		skippedCount = 0
	}

	importResult := &types.FAQImportResult{
		TotalEntries:       originalTotalEntries,
		SuccessCount:       progress.SuccessCount,
		FailedCount:        progress.FailedCount,
		PartialFailedCount: progress.PartialFailedCount,
		SkippedCount:       skippedCount,
		MergedCount:        progress.MergedCount,
		AddedCount:         progress.AddedCount,
		ImportMode:         payload.Mode,
		ImportedAt:         time.Now(),
		TaskID:             payload.TaskID,
		ProcessingTime:     time.Now().Unix() - progress.CreatedAt, // processing time in seconds
		DisplayStatus:      "open",                                 // a fresh import result is shown by default
	}

	// Record the CSV download URL when there are failed or partially failed entries
	// (FailedEntries covers both).
	if progress.FailedEntriesURL != "" {
		importResult.FailedEntriesURL = progress.FailedEntriesURL
	}

	if err := knowledge.SetLastFAQImportResult(importResult); err != nil {
		return fmt.Errorf("failed to set FAQ import result: %w", err)
	}

	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		return fmt.Errorf("failed to update knowledge with import result: %w", err)
	}

	logger.Infof(ctx, "Saved FAQ import result to database: knowledge_id=%s, task_id=%s, total=%d, success=%d, added=%d, merged=%d, failed=%d, partial_failed=%d, skipped=%d",
		payload.KnowledgeID, payload.TaskID, originalTotalEntries, progress.SuccessCount, progress.AddedCount, progress.MergedCount, progress.FailedCount, progress.PartialFailedCount, skippedCount)

	return nil
}

// buildFAQFailedEntry builds an FAQFailedEntry.
func buildFAQFailedEntry(idx int, reason string, entry *types.FAQEntryPayload) types.FAQFailedEntry {
	answerAll := false
	if entry.AnswerStrategy != nil && *entry.AnswerStrategy == types.AnswerStrategyAll {
		answerAll = true
	}
	isDisabled := false
	if entry.IsEnabled != nil && !*entry.IsEnabled {
		isDisabled = true
	}
	return types.FAQFailedEntry{
		Index:             idx,
		Reason:            reason,
		TagName:           entry.TagName,
		StandardQuestion:  strings.TrimSpace(entry.StandardQuestion),
		SimilarQuestions:  entry.SimilarQuestions,
		NegativeQuestions: entry.NegativeQuestions,
		Answers:           entry.Answers,
		AnswerAll:         answerAll,
		IsDisabled:        isDisabled,
	}
}

func buildFAQPartialFailedEntry(idx int, entry *types.FAQEntryPayload,
	removedSimilarQuestions, removedNegativeQuestions []string,
) types.FAQFailedEntry {
	answerAll := false
	if entry.AnswerStrategy != nil && *entry.AnswerStrategy == types.AnswerStrategyAll {
		answerAll = true
	}
	isDisabled := false
	if entry.IsEnabled != nil && !*entry.IsEnabled {
		isDisabled = true
	}

	// Build the failure reason: summary plus detail.
	var summary []string
	if len(removedSimilarQuestions) > 0 {
		summary = append(summary, fmt.Sprintf("%d similar question(s) removed", len(removedSimilarQuestions)))
	}
	if len(removedNegativeQuestions) > 0 {
		summary = append(summary, fmt.Sprintf("%d negative example(s) removed", len(removedNegativeQuestions)))
	}

	// Full reason: summary | similar-question detail | negative-example detail
	var reasonParts []string
	reasonParts = append(reasonParts, "Partially successful: "+strings.Join(summary, ", "))
	if len(removedSimilarQuestions) > 0 {
		reasonParts = append(reasonParts, strings.Join(removedSimilarQuestions, "; "))
	}
	if len(removedNegativeQuestions) > 0 {
		reasonParts = append(reasonParts, strings.Join(removedNegativeQuestions, "; "))
	}

	return types.FAQFailedEntry{
		Index:                    idx,
		Reason:                   strings.Join(reasonParts, " | "),
		IsPartialFailure:         true,
		TagName:                  entry.TagName,
		StandardQuestion:         strings.TrimSpace(entry.StandardQuestion),
		SimilarQuestions:         entry.SimilarQuestions,
		NegativeQuestions:        entry.NegativeQuestions,
		Answers:                  entry.Answers,
		AnswerAll:                answerAll,
		IsDisabled:               isDisabled,
		RemovedSimilarQuestions:  removedSimilarQuestions,
		RemovedNegativeQuestions: removedNegativeQuestions,
	}
}

// executeFAQDryRunValidation runs the FAQ dry-run validation and returns the indices of the entries that passed.
func (s *knowledgeService) executeFAQDryRunValidation(ctx context.Context,
	payload *types.FAQImportPayload, progress *types.FAQImportProgress,
) []int {
	entries := payload.Entries

	// Indices of entries that passed basic validation and the duplicate checks; a safety check follows.
	validEntryIndices := make([]int, 0, len(entries))

	if payload.Mode == types.FAQBatchModeAppend {
		validEntryIndices = s.validateEntriesForAppendModeWithProgress(ctx, payload.TenantID, payload.KBID, entries, progress)
	} else {
		validEntryIndices = s.validateEntriesForReplaceModeWithProgress(ctx, entries, progress)
	}

	return validEntryIndices
}

// validateEntriesForAppendModeWithProgress validates entries in Append mode (with progress updates).
// Note: the validation stage does not update Processed; only the actual import does.
//
// Validation runs in four phases:
// Phase one (pre-validation):
//  1. Standard question - de-duplicated within the file only; if the KB already has it,
//     the entry is marked as a merge candidate
//  2. Similar questions - compared against the file and the KB (a merge candidate excludes
//     its own existing questions) -> conflicting similar questions are dropped one by one
//  3. Negative examples - compared only against this entry's own standard and similar
//     questions -> conflicting negative examples are dropped one by one
//
// Phase two (post-validation, merge candidates only):
//  4. Re-run the negative example check on the merged data -> on a conflict the whole entry
//     is rolled back to its pre-merge state
func (s *knowledgeService) validateEntriesForAppendModeWithProgress(ctx context.Context,
	tenantID uint64, kbID string, entries []types.FAQEntryPayload, progress *types.FAQImportProgress,
) []int {
	totalEntries := len(entries)

	// Load the metadata of every FAQ chunk already in the knowledge base.
	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		logger.Warnf(ctx, "Failed to list existing FAQ chunks for dry run: %v", err)
	}

	// Map existing standard question -> chunk (used to spot merge candidates).
	existingStdQToChunk := make(map[string]*types.Chunk)
	// Map every existing question -> owning chunkID (used for similar-question conflict detection).
	existingQuestionToChunkID := make(map[string]string)
	// Set of questions each chunk owns (used to exclude itself when merging).
	existingChunkQuestions := make(map[string]map[string]bool)
	// Map chunkID -> standard question (used when reporting a conflict failure).
	existingChunkIDToStdQ := make(map[string]string)

	for _, chunk := range existingChunks {
		meta, err := chunk.FAQMetadata()
		if err != nil || meta == nil {
			continue
		}
		qs := make(map[string]bool)
		if meta.StandardQuestion != "" {
			existingStdQToChunk[meta.StandardQuestion] = chunk
			existingQuestionToChunkID[meta.StandardQuestion] = chunk.ID
			qs[meta.StandardQuestion] = true
		}
		for _, q := range meta.SimilarQuestions {
			if q != "" {
				existingQuestionToChunkID[q] = chunk.ID
				qs[q] = true
			}
		}
		existingChunkQuestions[chunk.ID] = qs
		existingChunkIDToStdQ[chunk.ID] = meta.StandardQuestion
	}

	// Merge candidate tracking: entry index -> existing target chunk.
	mergeChunkMap := make(map[int]*types.Chunk)

	// ==================== Pass one: basic format validation + in-file standard question de-duplication + merge candidate detection ====================
	batchStandardQuestions := make(map[string]int) // value is the index of the first occurrence
	validIndicesAfterStdQ := make([]int, 0, totalEntries)

	for i, entry := range entries {
		if err := validateFAQEntryPayloadBasic(&entry); err != nil {
			progress.FailedCount++
			fe := buildFAQFailedEntry(i, err.Error(), &entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			continue
		}

		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// In-file standard question de-duplication.
		if firstIdx, exists := batchStandardQuestions[standardQ]; exists {
			progress.FailedCount++
			fe := buildFAQFailedEntry(i, fmt.Sprintf("Standard question conflict: duplicates entry %d in this batch", firstIdx+1), &entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			continue
		}

		// Decide whether this is a merge candidate.
		if chunk, exists := existingStdQToChunk[standardQ]; exists {
			// The standard question already exists in the KB -> mark as a merge candidate.
			mergeChunkMap[i] = chunk
			logger.Infof(ctx, "FAQ entry %d: standard question '%s' exists in KB, marking as merge candidate (chunk_id=%s)", i, standardQ, chunk.ID)
		} else if conflictChunkID, hit := existingQuestionToChunkID[standardQ]; hit {
			// The standard question does not duplicate an existing standard question, but it
			// collides with a similar question of another KB entry -> fail up front, so that
			// calculateAppendOperations downstream does not drop it silently and skew the counts.
			conflictStdQ := existingChunkIDToStdQ[conflictChunkID]
			progress.FailedCount++
			fe := buildFAQFailedEntry(i,
				fmt.Sprintf(`Standard question conflict: duplicates similar question "%s" of standard question "%s" in the knowledge base`, conflictStdQ, standardQ),
				&entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			logger.Infof(ctx,
				"FAQ entry %d: standard question '%s' conflicts with existing similar question of chunk_id=%s (std='%s')",
				i, standardQ, conflictChunkID, conflictStdQ)
			continue
		}

		batchStandardQuestions[standardQ] = i
		validIndicesAfterStdQ = append(validIndicesAfterStdQ, i)

		if (i+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating standard questions %d/%d...", i+1, totalEntries)
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Pass two: similar question conflict detection ====================
	// Build the set of every standard and similar question in the batch.
	batchAllQuestions := make(map[string]int)
	for _, i := range validIndicesAfterStdQ {
		standardQ := strings.TrimSpace(entries[i].StandardQuestion)
		batchAllQuestions[standardQ] = i
		for _, q := range entries[i].SimilarQuestions {
			q = strings.TrimSpace(q)
			if q != "" {
				if _, exists := batchAllQuestions[q]; !exists {
					batchAllQuestions[q] = i
				}
			}
		}
	}

	removedSimilarQuestionsMap := make(map[int][]string)
	removedNegativeQuestionsMap := make(map[int][]string)

	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Merge candidate: load the target chunk's own questions so they can be excluded.
		var ownChunkQuestions map[string]bool
		if mergeChunk, isMerge := mergeChunkMap[i]; isMerge {
			ownChunkQuestions = existingChunkQuestions[mergeChunk.ID]
		}

		validSimilarQuestions := make([]string, 0, len(entry.SimilarQuestions))
		var removedSimilarQuestions []string
		for _, q := range entry.SimilarQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// A similar question must not equal its own standard question.
			if q == standardQ {
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`[Similar question conflict]: "%s" conflicts with this entry's standard question`, q))
				continue
			}
			// Similar question check against the standard and similar questions already in the KB.
			if _, exists := existingQuestionToChunkID[q]; exists {
				// Merge candidate: allow it when the question belongs to the merge target chunk itself
				// (de-duplication happens during the merge).
				if ownChunkQuestions != nil && ownChunkQuestions[q] {
					validSimilarQuestions = append(validSimilarQuestions, q)
					continue
				}
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`[Similar question conflict]: "%s" conflicts with an existing standard or similar question in the knowledge base`, q))
				continue
			}
			// Similar question check against the standard and similar questions in this batch.
			if firstIdx, exists := batchAllQuestions[q]; exists && firstIdx != i {
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`[Similar question conflict]: "%s" conflicts with the standard or similar question on row %d`, q, firstIdx+1))
				continue
			}
			validSimilarQuestions = append(validSimilarQuestions, q)
		}
		entries[i].SimilarQuestions = validSimilarQuestions

		if len(removedSimilarQuestions) > 0 {
			removedSimilarQuestionsMap[i] = removedSimilarQuestions
		}

		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating similar questions %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Pass three: negative example conflict detection (pre-validation, new entry data only) ====================
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		currentQAQuestions := make(map[string]bool)
		currentQAQuestions[standardQ] = true
		for _, q := range entry.SimilarQuestions {
			currentQAQuestions[q] = true
		}

		validNegativeQuestions := make([]string, 0, len(entry.NegativeQuestions))
		var removedNegativeQuestions []string
		for _, q := range entry.NegativeQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			if currentQAQuestions[q] {
				removedNegativeQuestions = append(removedNegativeQuestions, fmt.Sprintf(`[Negative example conflict]: "%s" conflicts with this entry's standard or similar question`, q))
				continue
			}
			validNegativeQuestions = append(validNegativeQuestions, q)
		}
		entries[i].NegativeQuestions = validNegativeQuestions

		if len(removedNegativeQuestions) > 0 {
			removedNegativeQuestionsMap[i] = removedNegativeQuestions
		}

		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating negative examples %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Pass four: post-validation (merge candidates only) ====================
	// Re-run the negative example check on the merged data; on a conflict the whole entry rolls back to its pre-merge state.
	postValidationFailed := make(map[int]bool)
	mergeCount := 0
	for _, i := range validIndicesAfterStdQ {
		mergeChunk, isMerge := mergeChunkMap[i]
		if !isMerge {
			continue
		}

		existingMeta, err := mergeChunk.FAQMetadata()
		if err != nil || existingMeta == nil {
			logger.Warnf(ctx, "FAQ entry %d: failed to get merge target metadata, skipping post-validation", i)
			continue
		}

		entry := &entries[i]

		// Compute the merged data.
		mergedSimilar := unionStrings(existingMeta.SimilarQuestions, entry.SimilarQuestions)
		mergedNegative := unionStrings(existingMeta.NegativeQuestions, entry.NegativeQuestions)

		// Build the merged conflict set (standard question plus every merged similar question).
		mergedPositiveSet := make(map[string]bool)
		mergedPositiveSet[existingMeta.StandardQuestion] = true
		for _, q := range mergedSimilar {
			mergedPositiveSet[q] = true
		}

		// Check each merged negative example against the merged standard / similar questions.
		var conflictingNegatives []string
		for _, q := range mergedNegative {
			if mergedPositiveSet[q] {
				conflictingNegatives = append(conflictingNegatives, q)
			}
		}

		if len(conflictingNegatives) > 0 {
			// Post-validation failed -> roll the whole entry back to its pre-merge state.
			postValidationFailed[i] = true
			delete(mergeChunkMap, i)
			progress.FailedCount++
			reason := fmt.Sprintf("Post-validation failed: merged negative example \"%s\" conflicts with a similar question", strings.Join(conflictingNegatives, ", "))
			fe := buildFAQFailedEntry(i, reason, entry)
			fe.FailureType = "post_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			logger.Infof(ctx, "FAQ entry %d: post-validation failed, conflicting negatives after merge: %v", i, conflictingNegatives)
		} else {
			mergeCount++
		}
	}

	// Drop entries that failed post-validation from the valid index list.
	if len(postValidationFailed) > 0 {
		filtered := make([]int, 0, len(validIndicesAfterStdQ))
		for _, i := range validIndicesAfterStdQ {
			if !postValidationFailed[i] {
				filtered = append(filtered, i)
			}
		}
		validIndicesAfterStdQ = filtered
	}

	// Append the partial failure information to FailedEntries.
	for _, i := range validIndicesAfterStdQ {
		removedSimilar := removedSimilarQuestionsMap[i]
		removedNegative := removedNegativeQuestionsMap[i]
		if len(removedSimilar) > 0 || len(removedNegative) > 0 {
			pf := buildFAQPartialFailedEntry(i, &entries[i], removedSimilar, removedNegative)
			progress.FailedEntries = append(progress.FailedEntries, pf)
			progress.PartialFailedCount++
		}
	}

	// Record the merged entry indices on progress (used by the execution stage and retries).
	mergeIndices := make([]int, 0, mergeCount)
	for _, i := range validIndicesAfterStdQ {
		if _, isMerge := mergeChunkMap[i]; isMerge {
			mergeIndices = append(mergeIndices, i)
		}
	}
	progress.MergeEntryIndices = mergeIndices

	logger.Infof(ctx, "Append mode validation completed: total=%d, valid=%d, merge_candidates=%d, failed=%d, partial_failed=%d",
		totalEntries, len(validIndicesAfterStdQ), mergeCount, progress.FailedCount, progress.PartialFailedCount)

	return validIndicesAfterStdQ
}

// validateEntriesForReplaceModeWithProgress validates entries in Replace mode (with progress updates).
// Note: the validation stage does not update Processed; only the actual import does.
// Three passes, so data filtered out earlier is never considered later:
// 1. Standard question - compared against every standard question -> the whole QA fails with "standard question conflict"
// 2. Similar questions - compared against every standard and similar question -> individual phrasings fail with "similar question conflict" (only the conflicting similar questions are dropped)
// 3. Negative examples - compared against every standard and similar question of this QA -> individual phrasings fail with "negative example conflict" (only the conflicting negative examples are dropped)
func (s *knowledgeService) validateEntriesForReplaceModeWithProgress(ctx context.Context,
	entries []types.FAQEntryPayload, progress *types.FAQImportProgress,
) []int {
	totalEntries := len(entries)

	// ==================== Pass one: basic format validation + standard question conflict detection ====================
	// A standard question conflict fails the whole QA.
	batchStandardQuestions := make(map[string]int) // value is the index of the first occurrence
	validIndicesAfterStdQ := make([]int, 0, totalEntries)

	for i, entry := range entries {
		// Validate the basic entry format.
		if err := validateFAQEntryPayloadBasic(&entry); err != nil {
			progress.FailedCount++
			progress.FailedEntries = append(progress.FailedEntries, buildFAQFailedEntry(i, err.Error(), &entry))
			continue
		}

		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Standard question check against every standard question -> the whole QA fails with "standard question conflict".
		if firstIdx, exists := batchStandardQuestions[standardQ]; exists {
			progress.FailedCount++
			progress.FailedEntries = append(progress.FailedEntries, buildFAQFailedEntry(i, fmt.Sprintf("Standard question conflict: duplicates entry %d in this batch", firstIdx+1), &entry))
			continue
		}

		// Record the standard question.
		batchStandardQuestions[standardQ] = i
		validIndicesAfterStdQ = append(validIndicesAfterStdQ, i)

		// Refresh the progress message periodically.
		if (i+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating standard questions %d/%d...", i+1, totalEntries)
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Pass two: similar question conflict detection ====================
	// Only entries that passed pass one are considered; a conflict drops only the conflicting similar question.
	// Build the set of every standard and similar question (entries that passed pass one only).
	batchAllQuestions := make(map[string]int) // value is the index of the first occurrence
	for _, i := range validIndicesAfterStdQ {
		standardQ := strings.TrimSpace(entries[i].StandardQuestion)
		batchAllQuestions[standardQ] = i
		for _, q := range entries[i].SimilarQuestions {
			q = strings.TrimSpace(q)
			if q != "" {
				// Record only the first occurrence.
				if _, exists := batchAllQuestions[q]; !exists {
					batchAllQuestions[q] = i
				}
			}
		}
	}

	// Collects the similar questions and negative examples removed per entry.
	removedSimilarQuestionsMap := make(map[int][]string)  // key is the entry index
	removedNegativeQuestionsMap := make(map[int][]string) // key is the entry index

	// For each entry that passed pass one, filter out conflicting similar questions.
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		validSimilarQuestions := make([]string, 0, len(entry.SimilarQuestions))
		var removedSimilarQuestions []string
		for _, q := range entry.SimilarQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// Similar question check against every standard and similar question -> individual phrasings fail with "similar question conflict".
			// Drop the similar question when it conflicts with another entry's standard or similar question (and is not its own standard question).
			if firstIdx, exists := batchAllQuestions[q]; exists && firstIdx != i {
				logger.Infof(ctx, "FAQ entry %d: similar question '%s' conflicts with entry %d, removing", i, q, firstIdx+1)
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`[Similar question conflict]: "%s" conflicts with the standard or similar question on row %d`, q, firstIdx+1))
				continue
			}
			// A similar question must not equal its own standard question.
			if q == standardQ {
				logger.Infof(ctx, "FAQ entry %d: similar question '%s' same as standard question, removing", i, q)
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(`[Similar question conflict]: "%s" conflicts with this entry's standard question`, q))
				continue
			}
			validSimilarQuestions = append(validSimilarQuestions, q)
		}
		entries[i].SimilarQuestions = validSimilarQuestions

		// Record the removed similar questions.
		if len(removedSimilarQuestions) > 0 {
			removedSimilarQuestionsMap[i] = removedSimilarQuestions
		}

		// Refresh the progress message periodically.
		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating similar questions %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Pass three: negative example conflict detection ====================
	// Only entries that passed the first two passes are considered; a conflict drops only the conflicting negative example.
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Build the set of every question in this QA (standard question plus the similar questions that passed).
		currentQAQuestions := make(map[string]bool)
		currentQAQuestions[standardQ] = true
		for _, q := range entry.SimilarQuestions {
			currentQAQuestions[q] = true
		}

		validNegativeQuestions := make([]string, 0, len(entry.NegativeQuestions))
		var removedNegativeQuestions []string
		for _, q := range entry.NegativeQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// Negative example check against every standard and similar question of this QA -> individual phrasings fail with "negative example conflict".
			if currentQAQuestions[q] {
				logger.Infof(ctx, "FAQ entry %d: negative question '%s' conflicts with current QA's questions, removing", i, q)
				removedNegativeQuestions = append(removedNegativeQuestions, fmt.Sprintf(`[Negative example conflict]: "%s" conflicts with this entry's standard or similar question`, q))
				continue
			}
			validNegativeQuestions = append(validNegativeQuestions, q)
		}
		entries[i].NegativeQuestions = validNegativeQuestions

		// Record the removed negative examples.
		if len(removedNegativeQuestions) > 0 {
			removedNegativeQuestionsMap[i] = removedNegativeQuestions
		}

		// Refresh the progress message periodically.
		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf("Validating negative examples %d/%d...", idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// Append the partial failure information to FailedEntries.
	for _, i := range validIndicesAfterStdQ {
		removedSimilar := removedSimilarQuestionsMap[i]
		removedNegative := removedNegativeQuestionsMap[i]
		if len(removedSimilar) > 0 || len(removedNegative) > 0 {
			pf := buildFAQPartialFailedEntry(i, &entries[i], removedSimilar, removedNegative)
			progress.FailedEntries = append(progress.FailedEntries, pf)
			progress.PartialFailedCount++
		}
	}

	return validIndicesAfterStdQ
}

// unionStrings merges two string slices and removes exact duplicates.
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// validateFAQEntryPayloadBasic validates the basic format of an FAQ entry.
func validateFAQEntryPayloadBasic(entry *types.FAQEntryPayload) error {
	if entry == nil {
		return fmt.Errorf("entry must not be empty")
	}
	standardQ := strings.TrimSpace(entry.StandardQuestion)
	if standardQ == "" {
		return fmt.Errorf("standard question must not be empty")
	}
	if len(entry.Answers) == 0 {
		return fmt.Errorf("answer must not be empty")
	}
	hasValidAnswer := false
	for _, a := range entry.Answers {
		if strings.TrimSpace(a) != "" {
			hasValidAnswer = true
			break
		}
	}
	if !hasValidAnswer {
		return fmt.Errorf("answers must not all be empty")
	}
	return nil
}

type faqMergeOperation struct {
	Entry         types.FAQEntryPayload
	ExistingChunk *types.Chunk
	OldMeta       *types.FAQChunkMetadata
	MergedMeta    *types.FAQChunkMetadata
	Detail        types.FAQMergeDetail
}

// calculateAppendOperations works out the operations for Append mode (with smart merging).
// When an entry's standard question already exists in the KB, the entry is treated as a merge
// (union of similar questions / negative examples, with answers and strategy taken from the new
// entry); otherwise it is a create. A merge target with no change (same hash and unchanged
// operational flags) is skipped, avoiding a pointless DB write and index rebuild.
//
// FAQ import is commonly used as "export, edit, re-append", and this merge semantics is what
// lets new similar questions be layered on without losing the historical data.
func (s *knowledgeService) calculateAppendOperations(ctx context.Context,
	tenantID uint64, kbID string, entries []types.FAQEntryPayload,
) (newEntries []types.FAQEntryPayload, mergeOps []faqMergeOperation, skippedCount int, err error) {
	if len(entries) == 0 {
		return nil, nil, 0, nil
	}

	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to list existing FAQ chunks: %w", err)
	}

	existingStdQToChunk := make(map[string]*types.Chunk)
	existingQuestions := make(map[string]bool)
	for _, chunk := range existingChunks {
		meta, cErr := chunk.FAQMetadata()
		if cErr != nil || meta == nil {
			continue
		}
		if meta.StandardQuestion != "" {
			existingStdQToChunk[meta.StandardQuestion] = chunk
			existingQuestions[meta.StandardQuestion] = true
		}
		for _, q := range meta.SimilarQuestions {
			if q != "" {
				existingQuestions[q] = true
			}
		}
	}

	batchQuestions := make(map[string]bool)
	newEntries = make([]types.FAQEntryPayload, 0, len(entries))
	mergeOps = make([]faqMergeOperation, 0)

	for entryIdx, entry := range entries {
		meta, sErr := sanitizeFAQEntryPayload(&entry)
		if sErr != nil {
			skippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry: %v", sErr)
			continue
		}

		// Check whether the standard question exists in the KB as a standard question -> merge candidate.
		existingChunk, isMergeCandidate := existingStdQToChunk[meta.StandardQuestion]
		if isMergeCandidate {
			existingMeta, mErr := existingChunk.FAQMetadata()
			if mErr != nil || existingMeta == nil {
				isMergeCandidate = false
			} else {
				mergedSimilar := unionStrings(existingMeta.SimilarQuestions, meta.SimilarQuestions)
				mergedNegative := unionStrings(existingMeta.NegativeQuestions, meta.NegativeQuestions)

				mergedMeta := &types.FAQChunkMetadata{
					StandardQuestion:  existingMeta.StandardQuestion,
					SimilarQuestions:  mergedSimilar,
					NegativeQuestions: mergedNegative,
					Answers:           meta.Answers,
					AnswerStrategy:    meta.AnswerStrategy,
					Version:           existingMeta.Version + 1,
					Source:            existingMeta.Source,
				}

				newHash := types.CalculateFAQContentHash(mergedMeta)
				enabledChanged := entry.IsEnabled != nil && *entry.IsEnabled != existingChunk.IsEnabled
				recommendedChanged := entry.IsRecommended != nil &&
					*entry.IsRecommended != existingChunk.Flags.HasFlag(types.ChunkFlagRecommended)
				answerStrategyChanged := meta.AnswerStrategy != existingMeta.AnswerStrategy
				if existingChunk.ContentHash == newHash && !enabledChanged && !recommendedChanged && !answerStrategyChanged {
					skippedCount++
					logger.Infof(ctx, "Skipping merge for unchanged FAQ entry: %s", meta.StandardQuestion)
					continue
				}

				oldAnswerStr := strings.Join(existingMeta.Answers, "##")
				newAnswerStr := strings.Join(meta.Answers, "##")
				oldSimilarSet := make(map[string]bool, len(existingMeta.SimilarQuestions))
				for _, q := range existingMeta.SimilarQuestions {
					oldSimilarSet[q] = true
				}
				oldNegativeSet := make(map[string]bool, len(existingMeta.NegativeQuestions))
				for _, q := range existingMeta.NegativeQuestions {
					oldNegativeSet[q] = true
				}
				newSimilarCount := 0
				for _, q := range mergedSimilar {
					if !oldSimilarSet[q] {
						newSimilarCount++
					}
				}
				newNegativeCount := 0
				for _, q := range mergedNegative {
					if !oldNegativeSet[q] {
						newNegativeCount++
					}
				}

				mergeOps = append(mergeOps, faqMergeOperation{
					Entry:         entry,
					ExistingChunk: existingChunk,
					OldMeta:       existingMeta,
					MergedMeta:    mergedMeta,
					Detail: types.FAQMergeDetail{
						Index:            entryIdx,
						StandardQuestion: meta.StandardQuestion,
						AnswerChanged:    oldAnswerStr != newAnswerStr,
						NewSimilarCount:  newSimilarCount,
						NewNegativeCount: newNegativeCount,
					},
				})
				continue
			}
		}

		// Reaching here means this is neither a merge candidate nor free of conflicts with the
		// existing KB / current batch standard or similar questions. Normally such conflicts are
		// caught in executeFAQDryRunValidation, so this is only a defensive fallback; it logs at
		// Warn level to make a gap in the validation layer easy to spot.
		if existingQuestions[meta.StandardQuestion] || batchQuestions[meta.StandardQuestion] {
			skippedCount++
			logger.Warnf(ctx,
				"calculateAppendOperations: dropping FAQ entry with duplicate standard question %q "+
					"(should have been filtered at validation); entry_idx=%d",
				meta.StandardQuestion, entryIdx)
			continue
		}

		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}
		newEntries = append(newEntries, entry)
	}

	return newEntries, mergeOps, skippedCount, nil
}

// calculateReplaceOperations works out which entries to delete, create and update in Replace mode,
// filtering out entries whose standard or similar questions duplicate another entry in the batch.
func (s *knowledgeService) calculateReplaceOperations(ctx context.Context,
	tenantID uint64, knowledgeID string, newEntries []types.FAQEntryPayload,
) ([]types.FAQEntryPayload, []*types.Chunk, int, error) {
	// Resolve kbID so tags can be resolved.
	var kbID string
	if len(newEntries) > 0 {
		// Derive kbID from knowledgeID.
		knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to get knowledge: %w", err)
		}
		if knowledge != nil {
			kbID = knowledge.KnowledgeBaseID
		}
	}

	// Compute the content hash of every new entry and build the hash -> entry map at the same time.
	type entryWithHash struct {
		entry types.FAQEntryPayload
		hash  string
		meta  *types.FAQChunkMetadata
	}
	entriesWithHash := make([]entryWithHash, 0, len(newEntries))
	newHashSet := make(map[string]bool)
	// Used to de-duplicate standard and similar questions within the batch.
	batchQuestions := make(map[string]bool)
	batchSkippedCount := 0

	for _, entry := range newEntries {
		meta, err := sanitizeFAQEntryPayload(&entry)
		if err != nil {
			batchSkippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry in replace mode: %v", err)
			continue
		}

		// Check whether the standard question is duplicated within the batch.
		if batchQuestions[meta.StandardQuestion] {
			batchSkippedCount++
			logger.Infof(ctx, "Skipping FAQ entry with duplicate standard question in batch: %s", meta.StandardQuestion)
			continue
		}

		// Check whether the similar questions are duplicated within the batch.
		hasDuplicateSimilar := false
		for _, q := range meta.SimilarQuestions {
			if batchQuestions[q] {
				hasDuplicateSimilar = true
				logger.Infof(ctx, "Skipping FAQ entry with duplicate similar question in batch: %s (standard: %s)", q, meta.StandardQuestion)
				break
			}
		}
		if hasDuplicateSimilar {
			batchSkippedCount++
			continue
		}

		// Add this entry's standard and similar questions to the batch set.
		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}

		hash := types.CalculateFAQContentHash(meta)
		if hash != "" {
			entriesWithHash = append(entriesWithHash, entryWithHash{entry: entry, hash: hash, meta: meta})
			newHashSet[hash] = true
		}
	}

	// Load every existing chunk.
	allExistingChunks, err := s.chunkRepo.ListAllFAQChunksByKnowledgeID(ctx, tenantID, knowledgeID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to list existing chunks: %w", err)
	}

	// Filter in memory for chunks matching a new entry hash and build the map.
	existingHashMap := make(map[string]*types.Chunk)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash != "" && newHashSet[chunk.ContentHash] {
			existingHashMap[chunk.ContentHash] = chunk
		}
	}

	// Work out which chunks to delete (present in the DB but not in the new batch, or hash mismatch).
	chunksToDelete := make([]*types.Chunk, 0)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash == "" {
			// No hash means it must be deleted (probably legacy data).
			chunksToDelete = append(chunksToDelete, chunk)
		} else if !newHashSet[chunk.ContentHash] {
			// The hash is not in the new entries, so it must be deleted.
			chunksToDelete = append(chunksToDelete, chunk)
		}
	}

	// Preload tag information in bulk to avoid a DB query per iteration.
	// Collect every tag_id and tag_name that needs looking up.
	tagSeqIDSet := make(map[int64]bool)
	tagNameSet := make(map[string]bool)
	for _, ewh := range entriesWithHash {
		if ewh.entry.TagID != 0 {
			tagSeqIDSet[ewh.entry.TagID] = true
		} else if ewh.entry.TagName != "" {
			tagNameSet[ewh.entry.TagName] = true
		} else {
			tagNameSet[types.UntaggedTagName] = true
		}
	}

	// Bulk lookup of tags by seq_id.
	tagSeqIDToUUID := make(map[int64]string)
	if len(tagSeqIDSet) > 0 {
		seqIDs := make([]int64, 0, len(tagSeqIDSet))
		for seqID := range tagSeqIDSet {
			seqIDs = append(seqIDs, seqID)
		}
		tags, err := s.tagRepo.GetBySeqIDs(ctx, tenantID, seqIDs)
		if err != nil {
			logger.Warnf(ctx, "Failed to batch load tags by seq_ids: %v", err)
		} else {
			for _, tag := range tags {
				tagSeqIDToUUID[tag.SeqID] = tag.ID
			}
		}
	}

	// Bulk lookup of tags by name.
	tagNameToUUID := make(map[string]string)
	if len(tagNameSet) > 0 && kbID != "" {
		for name := range tagNameSet {
			if tag, err := s.tagRepo.GetByName(ctx, tenantID, kbID, name); err == nil && tag != nil {
				tagNameToUUID[name] = tag.ID
			}
		}
	}

	logger.Infof(ctx, "Preloaded %d tags by seq_id, %d tags by name for %d entries",
		len(tagSeqIDToUUID), len(tagNameToUUID), len(entriesWithHash))

	// resolveTagIDFromCache resolves a tag ID from the cache, falling back to a DB query on a miss.
	resolveTagIDFromCache := func(entry *types.FAQEntryPayload) (string, error) {
		if entry.TagID != 0 {
			if uuid, ok := tagSeqIDToUUID[entry.TagID]; ok {
				return uuid, nil
			}
			// Cache miss: fall back to a DB query (the tag may need creating).
			return s.resolveTagID(ctx, kbID, entry)
		}
		tagName := entry.TagName
		if tagName == "" {
			tagName = types.UntaggedTagName
		}
		if uuid, ok := tagNameToUUID[tagName]; ok {
			return uuid, nil
		}
		// Cache miss: fall back to a DB query (the tag may need creating).
		return s.resolveTagID(ctx, kbID, entry)
	}

	// Work out which entries need creating (reusing the hashes already computed).
	entriesToProcess := make([]types.FAQEntryPayload, 0, len(entriesWithHash))
	skippedCount := batchSkippedCount

	for idx, ewh := range entriesWithHash {
		// Log progress every 1000 entries.
		if idx > 0 && idx%1000 == 0 {
			logger.Infof(ctx, "calculateReplaceOperations progress: %d/%d entries processed", idx, len(entriesWithHash))
		}

		existingChunk := existingHashMap[ewh.hash]
		if existingChunk != nil {
			// Hash matches, so check whether the tag changed.
			newTagID, err := resolveTagIDFromCache(&ewh.entry)
			if err != nil {
				logger.Warnf(ctx, "Failed to resolve tag for entry, treating as new: %v", err)
				entriesToProcess = append(entriesToProcess, ewh.entry)
				continue
			}

			enabledChanged := ewh.entry.IsEnabled != nil && *ewh.entry.IsEnabled != existingChunk.IsEnabled
			recommendedChanged := ewh.entry.IsRecommended != nil &&
				*ewh.entry.IsRecommended != existingChunk.Flags.HasFlag(types.ChunkFlagRecommended)
			answerStrategyChanged := false
			if existingMeta, metaErr := existingChunk.FAQMetadata(); metaErr == nil && existingMeta != nil {
				answerStrategyChanged = ewh.meta.AnswerStrategy != existingMeta.AnswerStrategy
			}

			if existingChunk.TagID != newTagID || enabledChanged || recommendedChanged || answerStrategyChanged {
				if existingChunk.TagID != newTagID {
					logger.Infof(ctx, "FAQ entry tag changed from %s to %s, will update", existingChunk.TagID, newTagID)
				}
				if enabledChanged || recommendedChanged || answerStrategyChanged {
					logger.Infof(ctx, "FAQ entry operational fields changed (enabled: %v->%v, recommended: %v->%v, answerStrategy changed: %v), will update",
						existingChunk.IsEnabled, ewh.entry.IsEnabled,
						existingChunk.Flags.HasFlag(types.ChunkFlagRecommended), ewh.entry.IsRecommended,
						answerStrategyChanged)
				}
				chunksToDelete = append(chunksToDelete, existingChunk)
				entriesToProcess = append(entriesToProcess, ewh.entry)
			} else {
				// Hash, tag and operational state are all identical, so skip.
				skippedCount++
			}
			continue
		}

		// The hash does not match or does not exist, so the entry must be created.
		entriesToProcess = append(entriesToProcess, ewh.entry)
	}

	return entriesToProcess, chunksToDelete, skippedCount, nil
}

// executeFAQImport runs the actual FAQ import logic.
func (s *knowledgeService) executeFAQImport(ctx context.Context, taskID string, kbID string,
	payload *types.FAQBatchUpsertPayload, tenantID uint64, processedCount int,
	progress *types.FAQImportProgress,
) (err error) {
	// Keep the knowledge base and embedding model details for index cleanup.
	var kb *types.KnowledgeBase
	var embeddingModel embedding.Embedder
	totalEntries := len(payload.Entries) + processedCount

	// Recovery: on any error or panic, roll back every chunk and index entry created so far.
	defer func() {
		// Capture the panic.
		if r := recover(); r != nil {
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			stack := string(buf[:n])
			logger.Errorf(ctx, "FAQ import task %s panicked: %v\n%s", taskID, r, stack)
			err = fmt.Errorf("panic during FAQ import: %v", r)
		}
	}()

	kb, ctx, err = s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}

	kb.EnsureDefaults()

	// Load the embedding model, used for the index cleanup that follows.
	embeddingModel, err = s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return fmt.Errorf("failed to get embedding model: %w", err)
	}
	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return err
	}

	// Read the index mode.
	indexMode := types.FAQIndexModeQuestionOnly
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	// Incremental update logic: work out which entries need processing.
	var entriesToProcess []types.FAQEntryPayload
	var chunksToDelete []*types.Chunk
	var skippedCount int

	if payload.Mode == types.FAQBatchModeReplace {
		// Replace mode: work out which entries to delete, create and update.
		entriesToProcess, chunksToDelete, skippedCount, err = s.calculateReplaceOperations(
			ctx,
			tenantID,
			faqKnowledge.ID,
			payload.Entries,
		)
		if err != nil {
			return fmt.Errorf("failed to calculate replace operations: %w", err)
		}

		// Delete the chunks that must go (including the old chunks of entries being updated).
		if len(chunksToDelete) > 0 {
			chunkIDsToDelete := make([]string, 0, len(chunksToDelete))
			for _, chunk := range chunksToDelete {
				chunkIDsToDelete = append(chunkIDsToDelete, chunk.ID)
			}
			if err := s.chunkRepo.DeleteChunks(ctx, tenantID, chunkIDsToDelete); err != nil {
				return fmt.Errorf("failed to delete chunks: %w", err)
			}
			// Delete the index entries.
			if err := s.deleteFAQChunkVectors(ctx, kb, faqKnowledge, chunksToDelete); err != nil {
				return fmt.Errorf("failed to delete chunk vectors: %w", err)
			}
			logger.Infof(ctx, "FAQ import task %s: deleted %d chunks (including updates)", taskID, len(chunksToDelete))
		}
	} else {
		// Append mode (smart merge): entries whose standard question already exists go through
		// merge ops, the rest are treated as new.
		var mergeOps []faqMergeOperation
		entriesToProcess, mergeOps, skippedCount, err = s.calculateAppendOperations(ctx, tenantID, kb.ID, payload.Entries)
		if err != nil {
			return fmt.Errorf("failed to calculate append operations: %w", err)
		}

		if len(mergeOps) > 0 {
			mergedCount, mergeErr := s.executeFAQMergeOperations(ctx, taskID, kb, faqKnowledge, embeddingModel, indexMode, mergeOps, progress)
			if mergeErr != nil {
				return fmt.Errorf("failed to execute merge operations: %w", mergeErr)
			}
			logger.Infof(ctx, "FAQ import task %s: merged %d entries", taskID, mergedCount)
			progress.MergedCount = mergedCount
			for _, op := range mergeOps {
				progress.MergeDetails = append(progress.MergeDetails, op.Detail)
			}
		}
	}
	logger.Infof(
		ctx,
		"FAQ import task %s: total entries: %d, new to create: %d, skipped: %d, merged: %d",
		taskID,
		len(payload.Entries),
		len(entriesToProcess),
		skippedCount,
		progress.MergedCount,
	)

	// Nothing to process, so return straight away.
	if len(entriesToProcess) == 0 {
		logger.Infof(ctx, "FAQ import task %s: no new entries to create", taskID)
		return nil
	}

	// Process the entries to create in batches.
	remainingEntries := len(entriesToProcess)
	totalStartTime := time.Now()
	actualProcessed := skippedCount + processedCount + progress.MergedCount

	logger.Infof(
		ctx,
		"FAQ import task %s: starting batch processing, remaining entries: %d, total entries: %d, batch size: %d",
		taskID,
		remainingEntries,
		totalEntries,
		faqImportBatchSize,
	)

	for i := 0; i < remainingEntries; i += faqImportBatchSize {
		batchStartTime := time.Now()
		end := i + faqImportBatchSize
		if end > remainingEntries {
			end = remainingEntries
		}

		batch := entriesToProcess[i:end]
		logger.Infof(ctx, "FAQ import task %s: processing batch %d-%d (%d entries)", taskID, i+1, end, len(batch))

		// Build the chunks.
		buildStartTime := time.Now()
		chunks := make([]*types.Chunk, 0, len(batch))
		chunkIds := make([]string, 0, len(batch))
		for idx, entry := range batch {
			meta, err := sanitizeFAQEntryPayload(&entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to sanitize entry at index %d: %w", i+idx, err)
			}

			// Resolve the TagID.
			tagID, err := s.resolveTagID(ctx, kbID, &entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to resolve tag for entry at index %d: %w", i+idx, err)
			}

			isEnabled := true
			if entry.IsEnabled != nil {
				isEnabled = *entry.IsEnabled
			}
			// ChunkIndex = startChunkIndex + (i+idx) + initialProcessed
			chunk := &types.Chunk{
				ID:              uuid.New().String(),
				TenantID:        tenantID,
				KnowledgeID:     faqKnowledge.ID,
				KnowledgeBaseID: kb.ID,
				Content:         buildFAQChunkContent(meta, indexMode),
				// ChunkIndex:      0,
				IsEnabled: isEnabled,
				ChunkType: types.ChunkTypeFAQ,
				TagID:     tagID,
				Status:    int(types.ChunkStatusStored), // store but not indexed
			}
			// When an ID was supplied (used for data migration), set SeqID.
			if entry.ID != nil && *entry.ID > 0 {
				chunk.SeqID = *entry.ID
			}
			if err := chunk.SetFAQMetadata(meta); err != nil {
				return fmt.Errorf("failed to set FAQ metadata: %w", err)
			}
			chunks = append(chunks, chunk)
			chunkIds = append(chunkIds, chunk.ID)
		}
		buildDuration := time.Since(buildStartTime)
		logger.Debugf(ctx, "FAQ import task %s: batch %d-%d built %d chunks in %v, chunk IDs: %v",
			taskID, i+1, end, len(chunks), buildDuration, chunkIds)
		// Create the chunks.
		createStartTime := time.Now()
		if err := s.chunkService.CreateChunks(ctx, chunks); err != nil {
			return fmt.Errorf("failed to create chunks: %w", err)
		}
		createDuration := time.Since(createStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d created %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			createDuration,
		)

		// Index the chunks.
		indexStartTime := time.Now()
		// Note: if indexing fails, the recovery in the defer rolls back the chunks and index data already created.
		if err := s.indexFAQChunks(ctx, kb, faqKnowledge, chunks, embeddingModel, true, false); err != nil {
			return fmt.Errorf("failed to index chunks: %w", err)
		}
		indexDuration := time.Since(indexStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d indexed %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			indexDuration,
		)

		// Mark the chunks as indexed: every row gets the same value, so a single
		// UPDATE ... WHERE id IN is enough and content need not be sent back.
		for _, chunk := range chunks {
			chunk.Status = int(types.ChunkStatusIndexed) // indexed
		}
		if err := s.chunkRepo.UpdateChunkFieldsByIDs(ctx, tenantID, chunkIds, map[string]interface{}{
			"status": int(types.ChunkStatusIndexed),
		}); err != nil {
			return fmt.Errorf("failed to update chunks status: %w", err)
		}

		// Collect the successful entries (tag information is fetched once per batch).
		tagsByID := s.loadFAQTagsForChunks(ctx, tenantID, chunks)
		for idx, chunk := range chunks {
			entryIdx := i + idx + processedCount // index in the original entry list
			meta, _ := chunk.FAQMetadata()
			standardQ := ""
			if meta != nil {
				standardQ = meta.StandardQuestion
			}
			tagID, tagName := faqTagInfo(tagsByID, chunk.TagID)
			progress.SuccessEntries = append(progress.SuccessEntries, types.FAQSuccessEntry{
				Index:            entryIdx,
				SeqID:            chunk.SeqID,
				TagID:            tagID,
				TagName:          tagName,
				StandardQuestion: standardQ,
			})
		}

		actualProcessed += len(batch)
		// Update the task progress.
		progress := int(float64(actualProcessed) / float64(totalEntries) * 100)
		if err := s.updateFAQImportProgressStatus(ctx, taskID, "", 0, types.FAQImportStatusProcessing, progress, totalEntries, actualProcessed, fmt.Sprintf("Processing entry %d/%d", actualProcessed, totalEntries), ""); err != nil {
			logger.Errorf(ctx, "Failed to update task progress: %v", err)
		}

		batchDuration := time.Since(batchStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d completed in %v (build: %v, create: %v, index: %v), total progress: %d/%d (%d%%)",
			taskID,
			i+1,
			end,
			batchDuration,
			buildDuration,
			createDuration,
			indexDuration,
			actualProcessed,
			totalEntries,
			progress,
		)
	}

	totalDuration := time.Since(totalStartTime)
	var avgPerEntry time.Duration
	if actualProcessed > 0 {
		avgPerEntry = totalDuration / time.Duration(actualProcessed)
	}
	logger.Infof(
		ctx,
		"FAQ import task %s: all batches completed, processed: %d entries (skipped: %d) in %v, avg: %v per entry",
		taskID,
		actualProcessed,
		skippedCount,
		totalDuration,
		avgPerEntry,
	)

	return nil
}

// updateFAQImportProgressStatus updates the FAQ import progress in Redis
func (s *knowledgeService) updateFAQImportProgressStatus(
	ctx context.Context,
	taskID string,
	instanceID string,
	enqueuedAt int64,
	status types.FAQImportTaskStatus,
	progress, total, processed int,
	message, errorMsg string,
) error {
	// Get existing progress from Redis
	existingProgress, err := s.GetFAQImportProgress(ctx, taskID)
	if err != nil {
		// If not found, create a new progress entry
		existingProgress = &types.FAQImportProgress{
			TaskID:    taskID,
			CreatedAt: time.Now().Unix(),
		}
	}

	// Update progress fields
	existingProgress.Status = status
	existingProgress.Progress = progress
	existingProgress.Total = total
	existingProgress.Processed = processed
	if message != "" {
		existingProgress.Message = message
	}
	existingProgress.Error = errorMsg
	if status == types.FAQImportStatusCompleted {
		existingProgress.Error = ""
	}

	// Clear the running key when the task finishes or fails.
	if status == types.FAQImportStatusCompleted || status == types.FAQImportStatusFailed {
		if existingProgress.KBID != "" {
			if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, existingProgress.KBID, taskID, instanceID, enqueuedAt); clearErr != nil {
				logger.Errorf(ctx, "Failed to clear running FAQ import task ID: %v", clearErr)
			}
		}
	}

	return s.saveFAQImportProgress(ctx, existingProgress)
}

// cleanupFAQEntriesFileOnFinalFailure removes the entries file from object storage once the task has finally failed.
// Cleanup only runs when retryCount >= maxRetry, because a retry still needs that file.
func (s *knowledgeService) cleanupFAQEntriesFileOnFinalFailure(ctx context.Context, entriesURL string, retryCount, maxRetry int) {
	if entriesURL == "" || retryCount < maxRetry {
		return
	}
	if err := s.fileSvc.DeleteFile(ctx, entriesURL); err != nil {
		logger.Warnf(ctx, "Failed to delete FAQ entries file from object storage on final failure: %v", err)
	} else {
		logger.Infof(ctx, "Deleted FAQ entries file from object storage on final failure: %s", entriesURL)
	}
}

// runningFAQImportInfo stores the task ID and enqueued timestamp for uniquely identifying a task instance
type runningFAQImportInfo struct {
	TaskID     string `json:"task_id"`
	EnqueuedAt int64  `json:"enqueued_at"`
	InstanceID string `json:"instance_id,omitempty"`
}

// getRunningFAQImportInfo checks if there's a running FAQ import task for the given KB
// Returns the task info if found, nil otherwise
func (s *knowledgeService) getRunningFAQImportInfo(ctx context.Context, kbID string) (*runningFAQImportInfo, error) {
	if s.redisClient == nil {
		if v, ok := s.memFAQRunningImport.Load(kbID); ok {
			return v.(*runningFAQImportInfo), nil
		}
		return nil, nil
	}
	key := getFAQImportRunningKey(kbID)
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get running FAQ import task: %w", err)
	}

	// Try to parse as JSON first (new format)
	var info runningFAQImportInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		// Fallback: old format was just taskID string
		return &runningFAQImportInfo{TaskID: data, EnqueuedAt: 0}, nil
	}
	return &info, nil
}

// getRunningFAQImportTaskID checks if there's a running FAQ import task for the given KB
// Returns the task ID if found, empty string otherwise (for backward compatibility)
func (s *knowledgeService) getRunningFAQImportTaskID(ctx context.Context, kbID string) (string, error) {
	info, err := s.getRunningFAQImportInfo(ctx, kbID)
	if err != nil {
		return "", err
	}
	if info == nil {
		return "", nil
	}
	return info.TaskID, nil
}

// setRunningFAQImportInfo sets the running task info for a KB
func (s *knowledgeService) setRunningFAQImportInfo(ctx context.Context, kbID string, info *runningFAQImportInfo) error {
	if s.redisClient == nil {
		s.memFAQRunningImport.Store(kbID, info)
		return nil
	}
	key := getFAQImportRunningKey(kbID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal running info: %w", err)
	}
	return s.redisClient.Set(ctx, key, data, faqImportProgressTTL).Err()
}

// clearRunningFAQImportTaskID clears the running task ID for a KB
func (s *knowledgeService) clearRunningFAQImportTaskID(ctx context.Context, kbID string) error {
	if s.redisClient == nil {
		s.memFAQRunningImport.Delete(kbID)
		return nil
	}
	key := getFAQImportRunningKey(kbID)
	return s.redisClient.Del(ctx, key).Err()
}

func (s *knowledgeService) clearRunningFAQImportInfoIfMatches(ctx context.Context, kbID, taskID, instanceID string, enqueuedAt int64) error {
	if s.redisClient == nil {
		if v, ok := s.memFAQRunningImport.Load(kbID); ok {
			info, _ := v.(*runningFAQImportInfo)
			if runningFAQImportInfoMatches(info, taskID, instanceID, enqueuedAt) {
				s.memFAQRunningImport.Delete(kbID)
			}
		}
		return nil
	}

	info, err := s.getRunningFAQImportInfo(ctx, kbID)
	if err != nil {
		return err
	}
	if !runningFAQImportInfoMatches(info, taskID, instanceID, enqueuedAt) {
		return nil
	}

	key := getFAQImportRunningKey(kbID)
	return s.redisClient.Del(ctx, key).Err()
}

func runningFAQImportInfoMatches(info *runningFAQImportInfo, taskID, instanceID string, enqueuedAt int64) bool {
	if info == nil || info.TaskID != taskID {
		return false
	}
	if info.InstanceID != "" && instanceID != "" {
		return info.InstanceID == instanceID
	}
	return enqueuedAt == 0 || info.EnqueuedAt == 0 || info.EnqueuedAt == enqueuedAt
}

// incrementalIndexFAQEntry incrementally updates the index of an FAQ entry.
// Only the changed content is re-embedded and re-indexed; unchanged parts are skipped.
func (s *knowledgeService) incrementalIndexFAQEntry(
	ctx context.Context,
	kb *types.KnowledgeBase,
	knowledge *types.Knowledge,
	chunk *types.Chunk,
	embeddingModel embedding.Embedder,
	oldStandardQuestion string,
	oldSimilarQuestions []string,
	oldAnswers []string,
	newMeta *types.FAQChunkMetadata,
) error {
	indexStartTime := time.Now()
	logger.Debugf(ctx, "incrementalIndexFAQEntry: starting for chunk=%s, oldSimilarQuestions=%d, newSimilarQuestions=%d",
		chunk.ID, len(oldSimilarQuestions), len(newMeta.SimilarQuestions))

	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, types.MustTenantIDFromContext(ctx), kb.VectorStoreID)
	if err != nil {
		return err
	}

	indexMode := types.FAQIndexModeQuestionAnswer
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	// Normalise the old and new data so the behaviour matches buildFAQIndexInfoList.
	// Normalise the old data.
	oldStandardQuestion = types.NormalizeQuestion(oldStandardQuestion)
	normalizedOldSimilarQuestions := make([]string, 0, len(oldSimilarQuestions))
	for _, q := range oldSimilarQuestions {
		if nq := types.NormalizeQuestion(q); nq != "" {
			normalizedOldSimilarQuestions = append(normalizedOldSimilarQuestions, nq)
		}
	}
	oldSimilarQuestions = normalizedOldSimilarQuestions
	oldAnswers = types.SanitizeStrings(oldAnswers)
	// Normalise the new data.
	normalizedNewMeta := newMeta.Normalize()

	// Build the index content.
	buildContent := func(question string, answers []string) string {
		if indexMode == types.FAQIndexModeQuestionAnswer && len(answers) > 0 {
			var builder strings.Builder
			builder.WriteString(question)
			for _, ans := range answers {
				builder.WriteString("\n")
				builder.WriteString(ans)
			}
			return builder.String()
		}
		return question
	}

	// Check whether the answers changed (only affects the index in QuestionAnswer mode).
	answersChanged := indexMode == types.FAQIndexModeQuestionAnswer && !slices.Equal(oldAnswers, normalizedNewMeta.Answers)
	logger.Debugf(ctx, "incrementalIndexFAQEntry: answersChanged=%v (indexMode=%s), oldAnswers=%d, newAnswers=%d",
		answersChanged, indexMode, len(oldAnswers), len(normalizedNewMeta.Answers))

	// Collect the index entries that need updating.
	var indexInfoToUpdate []*types.IndexInfo

	// 1. Check whether the standard question needs updating.
	oldStdContent := buildContent(oldStandardQuestion, oldAnswers)
	newStdContent := buildContent(normalizedNewMeta.StandardQuestion, normalizedNewMeta.Answers)
	stdQuestionChanged := oldStdContent != newStdContent
	if stdQuestionChanged {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: standard question changed, sourceID=%s", chunk.ID)
		indexInfoToUpdate = append(indexInfoToUpdate, &types.IndexInfo{
			Content:         newStdContent,
			SourceID:        chunk.ID,
			SourceType:      types.ChunkSourceType,
			ChunkID:         chunk.ID,
			KnowledgeID:     chunk.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			KnowledgeType:   types.KnowledgeTypeFAQ,
			TagID:           chunk.TagID,
			IsEnabled:       chunk.IsEnabled,
			IsRecommended:   chunk.Flags.HasFlag(types.ChunkFlagRecommended),
		})
	}

	// 2. Handle similar question additions, removals and edits by content hash.
	// Build the old question set (question -> present).
	oldQuestionsSet := make(map[string]struct{}, len(oldSimilarQuestions))
	for _, q := range oldSimilarQuestions {
		oldQuestionsSet[q] = struct{}{}
	}

	// Build the new question set.
	newQuestionsSet := make(map[string]struct{}, len(normalizedNewMeta.SimilarQuestions))
	for _, q := range normalizedNewMeta.SimilarQuestions {
		newQuestionsSet[q] = struct{}{}
	}

	// Find the questions to delete (present in the old set but not the new one).
	var sourceIDsToDelete []string
	var deletedQuestions []string
	for oldQ := range oldQuestionsSet {
		if _, exists := newQuestionsSet[oldQ]; !exists {
			sourceID := fmt.Sprintf("%s-%s", chunk.ID, hashQuestion(oldQ))
			sourceIDsToDelete = append(sourceIDsToDelete, sourceID)
			deletedQuestions = append(deletedQuestions, oldQ)
		}
	}

	// Find the questions to add or update.
	var addedQuestions, updatedQuestions []string
	for newQ := range newQuestionsSet {
		_, existedBefore := oldQuestionsSet[newQ]
		// Update is needed when:
		// 1. the question is new (did not exist before), or
		// 2. the answers changed (so it needs re-embedding).
		if !existedBefore || answersChanged {
			sourceID := fmt.Sprintf("%s-%s", chunk.ID, hashQuestion(newQ))
			indexInfoToUpdate = append(indexInfoToUpdate, &types.IndexInfo{
				Content:         buildContent(newQ, normalizedNewMeta.Answers),
				SourceID:        sourceID,
				SourceType:      types.ChunkSourceType,
				ChunkID:         chunk.ID,
				KnowledgeID:     chunk.KnowledgeID,
				KnowledgeBaseID: chunk.KnowledgeBaseID,
				KnowledgeType:   types.KnowledgeTypeFAQ,
				TagID:           chunk.TagID,
				IsEnabled:       chunk.IsEnabled,
				IsRecommended:   chunk.Flags.HasFlag(types.ChunkFlagRecommended),
			})
			if !existedBefore {
				addedQuestions = append(addedQuestions, newQ)
			} else {
				updatedQuestions = append(updatedQuestions, newQ)
			}
		}
	}

	// Log the changes in detail.
	if len(deletedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: deleted similar questions: %v", deletedQuestions)
	}
	if len(addedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: added similar questions: %v", addedQuestions)
	}
	if len(updatedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: updated similar questions (answers changed): %v", updatedQuestions)
	}

	// 3. Delete the index entries of similar questions that no longer exist.
	if len(sourceIDsToDelete) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: deleting %d obsolete sourceIDs: %v", len(sourceIDsToDelete), sourceIDsToDelete)
		if delErr := retrieveEngine.DeleteBySourceIDList(ctx, sourceIDsToDelete, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); delErr != nil {
			logger.Warnf(ctx, "incrementalIndexFAQEntry: failed to delete obsolete source IDs: %v", delErr)
		}
	}

	// 4. Bulk index the content that needs updating.
	newCount := len(normalizedNewMeta.SimilarQuestions)
	if len(indexInfoToUpdate) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: updating %d index entries (skipped %d unchanged)",
			len(indexInfoToUpdate), 1+newCount-len(indexInfoToUpdate))
		if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoToUpdate); err != nil {
			return err
		}
	} else {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: all %d entries unchanged, skipping index update", 1+newCount)
	}

	// 5. Update the knowledge record.
	now := time.Now()
	knowledge.UpdatedAt = now
	knowledge.ProcessedAt = &now
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		return err
	}

	totalDuration := time.Since(indexStartTime)
	logger.Debugf(ctx, "incrementalIndexFAQEntry: completed in %v, updated %d/%d entries",
		totalDuration, len(indexInfoToUpdate), 1+newCount)

	return nil
}

func (s *knowledgeService) indexFAQChunks(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge,
	chunks []*types.Chunk, embeddingModel embedding.Embedder,
	adjustStorage bool, needDelete bool,
) error {
	if len(chunks) == 0 {
		return nil
	}
	indexStartTime := time.Now()
	logger.Debugf(ctx, "indexFAQChunks: starting to index %d chunks", len(chunks))

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
	if err != nil {
		return err
	}

	// Build the index info.
	buildIndexInfoStartTime := time.Now()
	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}
	buildIndexInfoDuration := time.Since(buildIndexInfoStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: built %d index info entries for %d chunks in %v",
		len(indexInfo),
		len(chunks),
		buildIndexInfoDuration,
	)

	var size int64
	if adjustStorage {
		estimateStartTime := time.Now()
		size = retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
		estimateDuration := time.Since(estimateStartTime)
		logger.Debugf(ctx, "indexFAQChunks: estimated storage size %d bytes in %v", size, estimateDuration)
		if tenantInfo.StorageQuota > 0 && tenantInfo.StorageUsed+size > tenantInfo.StorageQuota {
			return types.NewStorageQuotaExceededError()
		}
	}

	// Delete the old vectors.
	var deleteDuration time.Duration
	if needDelete {
		deleteStartTime := time.Now()
		if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
			logger.Warnf(ctx, "Delete FAQ vectors failed: %v", err)
		}
		deleteDuration = time.Since(deleteStartTime)
		if deleteDuration > 100*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: deleted old vectors for %d chunks in %v", len(chunkIDs), deleteDuration)
		}
	}

	// Bulk index (this can be the bottleneck).
	batchIndexStartTime := time.Now()
	if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfo); err != nil {
		return err
	}
	batchIndexDuration := time.Since(batchIndexStartTime)
	var avgPerEntry time.Duration
	if len(indexInfo) > 0 {
		avgPerEntry = batchIndexDuration / time.Duration(len(indexInfo))
	}
	logger.Debugf(ctx, "indexFAQChunks: batch indexed %d index info entries in %v (avg: %v per entry)",
		len(indexInfo), batchIndexDuration, avgPerEntry)

	if adjustStorage && size > 0 {
		adjustStartTime := time.Now()
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, size); err == nil {
			tenantInfo.StorageUsed += size
		}
		knowledge.StorageSize += size
		adjustDuration := time.Since(adjustStartTime)
		if adjustDuration > 50*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: adjusted storage in %v", adjustDuration)
		}
	}

	updateStartTime := time.Now()
	now := time.Now()
	knowledge.UpdatedAt = now
	knowledge.ProcessedAt = &now
	err = s.repo.UpdateKnowledge(ctx, knowledge)
	updateDuration := time.Since(updateStartTime)
	if updateDuration > 50*time.Millisecond {
		logger.Debugf(ctx, "indexFAQChunks: updated knowledge in %v", updateDuration)
	}

	totalDuration := time.Since(indexStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: completed indexing %d chunks in %v (build: %v, delete: %v, batchIndex: %v, update: %v)",
		len(chunks),
		totalDuration,
		buildIndexInfoDuration,
		deleteDuration,
		batchIndexDuration,
		updateDuration,
	)

	return err
}

func (s *knowledgeService) deleteFAQChunkVectors(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge, chunks []*types.Chunk,
) error {
	if len(chunks) == 0 {
		return nil
	}
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return err
	}
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
	if err != nil {
		return err
	}

	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}

	size := retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
	if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
		return err
	}
	if size > 0 {
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, -size); err == nil {
			tenantInfo.StorageUsed -= size
			if tenantInfo.StorageUsed < 0 {
				tenantInfo.StorageUsed = 0
			}
		}
		if knowledge.StorageSize >= size {
			knowledge.StorageSize -= size
		} else {
			knowledge.StorageSize = 0
		}
	}
	knowledge.UpdatedAt = time.Now()
	return s.repo.UpdateKnowledge(ctx, knowledge)
}

func faqImportCompletedOutcome(successCount, failedCount, skippedCount int) types.AuditOutcome {
	if successCount > 0 && (failedCount > 0 || skippedCount > 0) {
		return types.AuditOutcomePartial
	}
	if successCount > 0 {
		return types.AuditOutcomeSuccess
	}
	if failedCount > 0 && skippedCount > 0 {
		return types.AuditOutcomePartial
	}
	if failedCount > 0 || skippedCount > 0 {
		return types.AuditOutcomeFailed
	}
	return types.AuditOutcomeSuccess
}

func faqImportActivityDetails(payload *types.FAQImportPayload, progress *types.FAQImportProgress, totalEntries int) map[string]any {
	details := map[string]any{"mode": payload.Mode}
	if progress == nil {
		return details
	}
	total := totalEntries
	if total <= 0 {
		total = progress.Total
	}
	if total > 0 {
		details["total"] = total
	}
	details["count"] = progress.SuccessCount
	if progress.FailedCount > 0 {
		details["failed"] = progress.FailedCount
	}
	skipped := progress.SkippedCount
	if skipped <= 0 && total > 0 {
		skipped = total - progress.SuccessCount - progress.FailedCount
		if skipped < 0 {
			skipped = 0
		}
	}
	if skipped > 0 {
		details["skipped"] = skipped
	}
	return details
}

func (s *knowledgeService) recordFAQImportKBActivity(
	ctx context.Context,
	payload *types.FAQImportPayload,
	progress *types.FAQImportProgress,
	totalEntries int,
	action types.AuditAction,
	outcome types.AuditOutcome,
) {
	if s == nil || payload == nil || payload.DryRun || payload.KBID == "" {
		return
	}
	recordKBActivity(ctx, s.audit, payload.TenantID, payload.KBID, action,
		"faq_entry", payload.KnowledgeID, outcome, faqImportActivityDetails(payload, progress, totalEntries))
}

// ProcessFAQImport handles Asynq FAQ import tasks (including dry run mode)
func (s *knowledgeService) ProcessFAQImport(ctx context.Context, t *asynq.Task) error {
	var payload types.FAQImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "failed to unmarshal FAQ import task payload: %v", err)
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}
	ctx = payload.Initiator.Apply(ctx)
	ctx = withKBActivityTask(ctx, payload.TaskID, kbActivityTrigger(ctx))

	ctx = logger.WithRequestID(ctx, uuid.New().String())
	ctx = logger.WithField(ctx, "faq_import", payload.TaskID)
	ctx = types.WithExecutionTenant(ctx, payload.TenantID)
	kb, err := s.validateFAQKnowledgeBase(ctx, payload.KBID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeBaseNotFound) {
			return fmt.Errorf("%w: FAQ task KB no longer exists", asynq.SkipRetry)
		}
		return err
	}
	ctx, err = access.WithKBTaskWrite(ctx, kb, payload.TenantID)
	if err != nil {
		return fmt.Errorf("%w: FAQ task KB does not belong to its tenant", asynq.SkipRetry)
	}
	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeNotFound) {
			return fmt.Errorf("%w: FAQ task document no longer exists", asynq.SkipRetry)
		}
		return err
	}
	if knowledge == nil || knowledge.TenantID != payload.TenantID || knowledge.KnowledgeBaseID != payload.KBID ||
		knowledge.Type != types.KnowledgeTypeFAQ {
		return fmt.Errorf("%w: FAQ task document does not belong to its KB", asynq.SkipRetry)
	}

	// Read the task retry info to tell whether this is the final retry.
	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	isLastRetry := retryCount >= maxRetry

	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil {
		logger.Errorf(ctx, "failed to get tenant: %v", err)
		return nil
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	// Download the entries first when they live in object storage.
	if payload.EntriesURL != "" && len(payload.Entries) == 0 {
		logger.Infof(ctx, "Downloading FAQ entries from object storage: %s", payload.EntriesURL)
		reader, err := s.fileSvc.GetFile(ctx, payload.EntriesURL)
		if err != nil {
			logger.Errorf(ctx, "Failed to download FAQ entries from object storage: %v", err)
			return fmt.Errorf("failed to download entries: %w", err)
		}
		defer reader.Close()

		entriesData, err := io.ReadAll(reader)
		if err != nil {
			logger.Errorf(ctx, "Failed to read FAQ entries data: %v", err)
			return fmt.Errorf("failed to read entries data: %w", err)
		}

		var entries []types.FAQEntryPayload
		if err := json.Unmarshal(entriesData, &entries); err != nil {
			logger.Errorf(ctx, "Failed to unmarshal FAQ entries: %v", err)
			return fmt.Errorf("failed to unmarshal entries: %w", err)
		}

		payload.Entries = entries
		logger.Infof(ctx, "Downloaded %d FAQ entries from object storage", len(entries))
	}

	logger.Infof(ctx, "Processing FAQ import task: task_id=%s, kb_id=%s, total_entries=%d, dry_run=%v, retry=%d/%d",
		payload.TaskID, payload.KBID, len(payload.Entries), payload.DryRun, retryCount, maxRetry)
	if err := s.validateFAQImportTags(ctx, kb, payload.Entries); err != nil {
		return err
	}

	// Keep the original total count.
	originalTotalEntries := len(payload.Entries)

	// Initialise progress.
	// Check for an existing validation result (lets a retry skip validation).
	// Note: this must be read before the new progress is saved, otherwise it is overwritten.
	existingProgress, _ := s.GetFAQImportProgress(ctx, payload.TaskID)

	progress := &types.FAQImportProgress{
		TaskID:         payload.TaskID,
		KBID:           payload.KBID,
		KnowledgeID:    payload.KnowledgeID,
		Status:         types.FAQImportStatusProcessing,
		Progress:       0,
		Total:          originalTotalEntries,
		Processed:      0,
		SuccessCount:   0,
		FailedCount:    0,
		FailedEntries:  make([]types.FAQFailedEntry, 0),
		SuccessEntries: make([]types.FAQSuccessEntry, 0),
		Message:        "Validating entries...",
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
		DryRun:         payload.DryRun,
	}
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to save initial FAQ import progress: %v", err)
	}

	var validEntryIndices []int
	if existingProgress != nil && len(existingProgress.ValidEntryIndices) > 0 {
		// On a retry, reuse the previous validation result.
		validEntryIndices = existingProgress.ValidEntryIndices
		progress.FailedCount = existingProgress.FailedCount
		progress.FailedEntries = existingProgress.FailedEntries
		logger.Infof(ctx, "Reusing previous validation result: valid=%d, failed=%d",
			len(validEntryIndices), progress.FailedCount)
	} else {
		// Step one: validate (both dry run and import mode need this).
		validEntryIndices = s.executeFAQDryRunValidation(ctx, &payload, progress)
		// Keep the indices that passed so a retry can skip validation.
		progress.ValidEntryIndices = validEntryIndices
		if err := s.saveFAQImportProgress(ctx, progress); err != nil {
			logger.Warnf(ctx, "Failed to save validation result: %v", err)
		}
		logger.Infof(ctx, "FAQ validation completed: total=%d, valid=%d, failed=%d",
			originalTotalEntries, len(validEntryIndices), progress.FailedCount)
	}

	// Dry run mode: return the result as soon as validation completes.
	if payload.DryRun {
		return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
	}

	// Import mode: check whether there is anything valid to import.
	if len(validEntryIndices) == 0 {
		// Nothing valid, so finish here.
		return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
	}

	// Extract the valid entries.
	validEntries := make([]types.FAQEntryPayload, 0, len(validEntryIndices))
	for _, idx := range validEntryIndices {
		validEntries = append(validEntries, payload.Entries[idx])
	}

	// Update the progress message.
	progress.Message = fmt.Sprintf("Validation complete, importing %d valid entries...", len(validEntries))
	progress.UpdatedAt = time.Now().Unix()
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to update FAQ import progress: %v", err)
	}

	// Check the task status for idempotency (reusing the existingProgress read earlier).
	var processedCount int
	if existingProgress != nil {
		if existingProgress.Status == types.FAQImportStatusCompleted {
			logger.Infof(ctx, "FAQ import already completed, skipping: %s", payload.TaskID)
			if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, payload.KBID, payload.TaskID, payload.InstanceID, payload.EnqueuedAt); clearErr != nil {
				logger.Warnf(ctx, "Failed to clear running FAQ import info for completed task: %v", clearErr)
			}
			return nil // Idempotent: an already finished task returns straight away
		}
		// Read how many were processed (note: this index is relative to validEntries).
		processedCount = existingProgress.Processed - progress.FailedCount // processed - validation failures = valid entries already imported
		if processedCount < 0 {
			processedCount = 0
		}
		logger.Infof(ctx, "Resuming FAQ import from progress: %d/%d (valid entries)", processedCount, len(validEntries))
	}

	// Idempotency: clean up chunks and index data that may have been partially processed.
	chunksDeleted, err := s.chunkRepo.DeleteUnindexedChunks(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to delete unindexed chunks: %v", err)
		// On the final retry, mark the task as failed.
		if isLastRetry {
			if updateErr := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusFailed, 0, originalTotalEntries, 0, "Failed to clean up unindexed data", err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
			s.recordFAQImportKBActivity(ctx, &payload, progress, originalTotalEntries, types.AuditActionFAQImportFailed, types.AuditOutcomeFailed)
		}
		s.cleanupFAQEntriesFileOnFinalFailure(ctx, payload.EntriesURL, retryCount, maxRetry)
		return fmt.Errorf("failed to delete unindexed chunks: %w", err)
	}
	if len(chunksDeleted) > 0 {
		logger.Infof(ctx, "Deleted unindexed chunks: %d", len(chunksDeleted))

		// Delete the index data.
		embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
		if err == nil {
			retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
				ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
			if err == nil {
				chunkIDs := make([]string, 0, len(chunksDeleted))
				for _, chunk := range chunksDeleted {
					chunkIDs = append(chunkIDs, chunk.ID)
				}
				if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
					logger.Warnf(ctx, "Failed to delete index data for chunks (may not exist): %v", err)
				} else {
					logger.Infof(ctx, "Successfully deleted index data for %d chunks", len(chunksDeleted))
				}
			}
		}
	}

	// Carry on from where processing stopped when some valid entries were already handled.
	entriesToImport := validEntries
	importMode := payload.Mode
	if processedCount > 0 && processedCount < len(validEntries) {
		entriesToImport = validEntries[processedCount:]
		// On a retry, switch to Append mode when some data was already processed, because the
		// Replace-mode deletion already ran on the first attempt. Staying in Replace mode would
		// make calculateReplaceOperations mark the data imported earlier for deletion and lose it.
		if payload.Mode == types.FAQBatchModeReplace {
			importMode = types.FAQBatchModeAppend
			logger.Infof(ctx, "Switching to Append mode for retry, original mode was Replace")
		}
		logger.Infof(ctx, "Continuing FAQ import from entry %d, remaining: %d entries", processedCount, len(entriesToImport))
	}

	// Build the FAQBatchUpsertPayload from the entries that passed validation.
	faqPayload := &types.FAQBatchUpsertPayload{
		Entries: entriesToImport,
		Mode:    importMode,
	}

	// Run the FAQ import (passing the processed offset, used for progress).
	if err := s.executeFAQImport(ctx, payload.TaskID, payload.KBID, faqPayload, payload.TenantID, progress.FailedCount+processedCount, progress); err != nil {
		logger.Errorf(ctx, "FAQ import task failed: %s, error: %v", payload.TaskID, err)
		// On the final retry, mark the task as failed.
		if isLastRetry {
			if updateErr := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusFailed, 0, originalTotalEntries, len(validEntries), "Import failed", err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
			s.recordFAQImportKBActivity(ctx, &payload, progress, originalTotalEntries, types.AuditActionFAQImportFailed, types.AuditOutcomeFailed)
		}
		s.cleanupFAQEntriesFileOnFinalFailure(ctx, payload.EntriesURL, retryCount, maxRetry)
		return fmt.Errorf("FAQ import failed: %w", err)
	}

	// The task finished successfully.
	logger.Infof(ctx, "FAQ import task completed: %s, imported: %d, failed: %d",
		payload.TaskID, len(progress.SuccessEntries), progress.FailedCount)

	// Final processing (generating the failed entries CSV and so on).
	return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
}

// finalizeFAQValidation finishes an FAQ validation/import task, generating the failed entries CSV when needed.
func (s *knowledgeService) finalizeFAQValidation(ctx context.Context, payload *types.FAQImportPayload,
	progress *types.FAQImportProgress, originalTotalEntries int,
) error {
	// Clean up the entries file in object storage, if any.
	if payload.EntriesURL != "" {
		if err := s.fileSvc.DeleteFile(ctx, payload.EntriesURL); err != nil {
			logger.Warnf(ctx, "Failed to delete FAQ entries file from object storage: %v", err)
		} else {
			logger.Infof(ctx, "Deleted FAQ entries file from object storage: %s", payload.EntriesURL)
		}
	}
	progress.UpdatedAt = time.Now().Unix()

	// Generate the CSV when there are failed entries.
	if len(progress.FailedEntries) > 0 {
		csvURL, err := s.generateFailedEntriesCSV(ctx, payload.TenantID, payload.TaskID, progress.FailedEntries)
		if err != nil {
			logger.Warnf(ctx, "Failed to generate failed entries CSV: %v", err)
		} else {
			progress.FailedEntriesURL = csvURL
			progress.FailedEntries = nil // clear the inline data and use the URL instead
			progress.Message += " (failed records exported as CSV)"
		}
	}

	// The final statistics must be computed before saveFAQImportResultToDatabase.
	progress.Status = types.FAQImportStatusCompleted
	progress.Progress = 100
	progress.Processed = originalTotalEntries

	if len(progress.ValidEntryIndices) > 0 {
		progress.SuccessCount = len(progress.ValidEntryIndices) - progress.PartialFailedCount
	} else if len(progress.SuccessEntries) > 0 {
		progress.SuccessCount = len(progress.SuccessEntries) - progress.PartialFailedCount
	} else {
		progress.SuccessCount = originalTotalEntries - progress.FailedCount - progress.PartialFailedCount
	}
	if progress.SuccessCount < 0 {
		progress.SuccessCount = 0
	}

	if progress.AddedCount == 0 && progress.MergedCount > 0 {
		progress.AddedCount = progress.SuccessCount - progress.MergedCount
		if progress.AddedCount < 0 {
			progress.AddedCount = 0
		}
	} else if progress.AddedCount == 0 {
		progress.AddedCount = progress.SuccessCount
	}

	skippedCount := originalTotalEntries - progress.SuccessCount - progress.PartialFailedCount - progress.FailedCount
	if skippedCount < 0 {
		skippedCount = 0
	}
	progress.SkippedCount = skippedCount

	if payload.DryRun {
		progress.Message = s.buildFAQImportResultMessage("Validation complete", progress)
	} else {
		progress.Message = s.buildFAQImportResultMessage("Import complete", progress)
	}

	// Persist the import result statistics unless this is a dry run.
	if !payload.DryRun {
		if err := s.saveFAQImportResultToDatabase(ctx, payload, progress, originalTotalEntries); err != nil {
			logger.Warnf(ctx, "Failed to save FAQ import result to database: %v", err)
		}

		// Only replace mode cleans up unused tags.
		// Append mode must not delete empty tags the user created up front.
		if payload.Mode == types.FAQBatchModeReplace {
			deletedTags, err := s.tagRepo.DeleteUnusedTags(ctx, payload.TenantID, payload.KBID)
			if err != nil {
				logger.Warnf(ctx, "FAQ import task %s: failed to cleanup unused tags: %v", payload.TaskID, err)
			} else if deletedTags > 0 {
				logger.Infof(ctx, "FAQ import task %s: cleaned up %d unused tags after replace import", payload.TaskID, deletedTags)
			}
		}
	}

	// Go through updateFAQImportProgressStatus so the running key is cleared correctly,
	// but save the other fields first, because it does not persist every field.
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to save final FAQ import progress: %v", err)
	}

	// Then call the status update to clear the running key.
	if err := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusCompleted,
		100, originalTotalEntries, originalTotalEntries, progress.Message, ""); err != nil {
		logger.Warnf(ctx, "Failed to update final FAQ import status: %v", err)
	}

	logger.Infof(ctx, "FAQ task completed: %s, dry_run=%v, success: %d, added: %d, merged: %d, failed: %d, partial_failed: %d",
		payload.TaskID, payload.DryRun, progress.SuccessCount, progress.AddedCount, progress.MergedCount, progress.FailedCount, progress.PartialFailedCount)

	if !payload.DryRun {
		outcome := faqImportCompletedOutcome(progress.SuccessCount, progress.FailedCount, progress.SkippedCount)
		s.recordFAQImportKBActivity(ctx, payload, progress, originalTotalEntries,
			types.AuditActionFAQImportCompleted, outcome)
	}

	return nil
}

// executeFAQMergeOperations applies append-mode merge operations in bulk: updating the
// metadata / content / index of existing chunks. ListChunksByID loads them in bulk and
// SaveChunks saves them in one transaction, cutting DB round trips. Any batch failure
// returns immediately and executeFAQImport's defer recovery decides whether to roll the
// whole import back.
// Indexing fans out per entry here (EFPutDocument) rather than rebuilding the merged chunks
// in bulk, because the index layer overwrites by SourceID, so putting the final merged
// content directly is enough.
func (s *knowledgeService) executeFAQMergeOperations(
	ctx context.Context,
	taskID string,
	kb *types.KnowledgeBase,
	faqKnowledge *types.Knowledge,
	embeddingModel embedding.Embedder,
	indexMode types.FAQIndexMode,
	mergeOps []faqMergeOperation,
	progress *types.FAQImportProgress,
) (int, error) {
	if len(mergeOps) == 0 {
		return 0, nil
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	mergedCount := 0

	for batchStart := 0; batchStart < len(mergeOps); batchStart += faqImportBatchSize {
		batchEnd := batchStart + faqImportBatchSize
		if batchEnd > len(mergeOps) {
			batchEnd = len(mergeOps)
		}
		batch := mergeOps[batchStart:batchEnd]

		// 1. Load the full chunks in bulk (calculateAppendOperations loads only some fields,
		//    missing status/is_enabled/flags/seq_id, so a direct update would zero them out).
		chunkIDs := make([]string, len(batch))
		for i, op := range batch {
			chunkIDs[i] = op.ExistingChunk.ID
		}
		fullChunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, chunkIDs)
		if err != nil {
			logger.Errorf(ctx, "FAQ import task %s: failed to batch reload chunks for merge: %v", taskID, err)
			return mergedCount, fmt.Errorf("failed to batch reload chunks for merge: %w", err)
		}
		chunkMap := make(map[string]*types.Chunk, len(fullChunks))
		for _, c := range fullChunks {
			chunkMap[c.ID] = c
		}

		// 2. Apply the merged data entry by entry.
		mergedChunks := make([]*types.Chunk, 0, len(batch))
		for _, op := range batch {
			fullChunk, ok := chunkMap[op.ExistingChunk.ID]
			if !ok {
				logger.Errorf(ctx, "FAQ import task %s: chunk %s not found during batch reload", taskID, op.ExistingChunk.ID)
				return mergedCount, fmt.Errorf("chunk %s not found during batch reload", op.ExistingChunk.ID)
			}

			if err := fullChunk.SetFAQMetadata(op.MergedMeta); err != nil {
				logger.Errorf(ctx, "FAQ import task %s: failed to set merged metadata for chunk %s: %v", taskID, fullChunk.ID, err)
				return mergedCount, fmt.Errorf("failed to set merged FAQ metadata: %w", err)
			}

			fullChunk.Content = buildFAQChunkContent(op.MergedMeta, indexMode)
			fullChunk.ContentHash = types.CalculateFAQContentHash(op.MergedMeta)
			fullChunk.UpdatedAt = time.Now()

			// Overwrite the operational state with the new values.
			if op.Entry.IsEnabled != nil {
				fullChunk.IsEnabled = *op.Entry.IsEnabled
			}
			if op.Entry.IsRecommended != nil {
				if *op.Entry.IsRecommended {
					fullChunk.Flags = fullChunk.Flags.SetFlag(types.ChunkFlagRecommended)
				} else {
					fullChunk.Flags = fullChunk.Flags.ClearFlag(types.ChunkFlagRecommended)
				}
			}

			mergedChunks = append(mergedChunks, fullChunk)
		}

		// 3. Save in bulk inside a transaction (GORM Save updates every field, so metadata and
		//    content_hash are persisted).
		if err := s.chunkRepo.SaveChunks(ctx, mergedChunks); err != nil {
			logger.Errorf(ctx, "FAQ import task %s: failed to batch save merged chunks: %v", taskID, err)
			return mergedCount, fmt.Errorf("failed to batch save merged chunks: %w", err)
		}

		// 4. Rebuild the index (EFPutDocument overwrites the same SourceID automatically).
		if err := s.indexFAQChunks(ctx, kb, faqKnowledge, mergedChunks, embeddingModel, false, false); err != nil {
			return mergedCount, fmt.Errorf("failed to re-index merged chunks: %w", err)
		}

		// 5. Collect the successful entries.
		tagsByID := s.loadFAQTagsForChunks(ctx, tenantID, mergedChunks)
		for i, op := range batch {
			chunk := mergedChunks[i]
			meta := op.MergedMeta
			tagID, tagName := faqTagInfo(tagsByID, chunk.TagID)
			progress.SuccessEntries = append(progress.SuccessEntries, types.FAQSuccessEntry{
				Index:            op.Detail.Index,
				SeqID:            chunk.SeqID,
				TagID:            tagID,
				TagName:          tagName,
				StandardQuestion: meta.StandardQuestion,
			})
		}

		mergedCount += len(batch)

		logger.Infof(ctx, "FAQ import task %s: merged batch %d-%d (%d chunks)", taskID, batchStart+1, batchEnd, len(mergedChunks))
	}

	return mergedCount, nil
}

// loadFAQTagsForChunks resolves every distinct tag referenced by chunks with a
// single query. Lookup failures are logged and yield an empty map so the
// import result degrades to "no tag info" instead of aborting the batch.
func (s *knowledgeService) loadFAQTagsForChunks(
	ctx context.Context, tenantID uint64, chunks []*types.Chunk,
) map[string]*types.KnowledgeTag {
	tagsByID := make(map[string]*types.KnowledgeTag)
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, chunk := range chunks {
		if chunk == nil || chunk.TagID == "" {
			continue
		}
		if _, ok := seen[chunk.TagID]; ok {
			continue
		}
		seen[chunk.TagID] = struct{}{}
		ids = append(ids, chunk.TagID)
	}
	if len(ids) == 0 {
		return tagsByID
	}
	tags, err := s.tagRepo.GetByIDs(ctx, tenantID, ids)
	if err != nil {
		logger.Warnf(ctx, "Failed to load FAQ tags for import result: %v", err)
		return tagsByID
	}
	for _, tag := range tags {
		if tag != nil {
			tagsByID[tag.ID] = tag
		}
	}
	return tagsByID
}

// faqTagInfo returns the external (seq_id, name) pair for tagID, or zero values
// when the chunk has no tag or the tag could not be loaded.
func faqTagInfo(tagsByID map[string]*types.KnowledgeTag, tagID string) (int64, string) {
	if tagID == "" {
		return 0, ""
	}
	if tag, ok := tagsByID[tagID]; ok && tag != nil {
		return tag.SeqID, tag.Name
	}
	return 0, ""
}

// buildFAQImportResultMessage builds the human-readable message for the final FAQ import / validation result.
// The frontend shows it directly in a toast or the task list, so keep it simple and clear:
//   - Default form: "Import complete / uploaded N / succeeded X [/ failed Y] [/ partially failed Z]"
//   - When MergedCount > 0 it switches to the split form: "/ added X / merged Y", so that in
//     append mode the user sees how many historical FAQs were merged rather than only the total.
func (s *knowledgeService) buildFAQImportResultMessage(prefix string, progress *types.FAQImportProgress) string {
	parts := []string{prefix}
	parts = append(parts, fmt.Sprintf("uploaded %d", progress.Total))

	if progress.MergedCount > 0 {
		parts = append(parts, fmt.Sprintf("added %d", progress.AddedCount))
		parts = append(parts, fmt.Sprintf("merged %d", progress.MergedCount))
	} else {
		parts = append(parts, fmt.Sprintf("succeeded %d", progress.SuccessCount))
	}

	if progress.FailedCount > 0 {
		parts = append(parts, fmt.Sprintf("failed %d", progress.FailedCount))
	}
	if progress.PartialFailedCount > 0 {
		parts = append(parts, fmt.Sprintf("partially failed %d", progress.PartialFailedCount))
	}

	return strings.Join(parts, " / ")
}

const (
	faqImportProgressKeyPrefix = "faq_import_progress:"
	faqImportRunningKeyPrefix  = "faq_import_running:"
	faqImportProgressTTL       = 3 * time.Hour
)

// getFAQImportProgressKey returns the Redis key for storing FAQ import progress
func getFAQImportProgressKey(taskID string) string {
	return faqImportProgressKeyPrefix + taskID
}

// getFAQImportRunningKey returns the Redis key for storing running task ID by KB ID
func getFAQImportRunningKey(kbID string) string {
	return faqImportRunningKeyPrefix + kbID
}

// saveFAQImportProgress saves the FAQ import progress to Redis
func (s *knowledgeService) saveFAQImportProgress(ctx context.Context, progress *types.FAQImportProgress) error {
	if s.redisClient == nil {
		progress.UpdatedAt = time.Now().Unix()
		s.memFAQProgress.Store(progress.TaskID, progress)
		return nil
	}
	key := getFAQImportProgressKey(progress.TaskID)
	progress.UpdatedAt = time.Now().Unix()
	data, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("failed to marshal FAQ import progress: %w", err)
	}
	return s.redisClient.Set(ctx, key, data, faqImportProgressTTL).Err()
}

// GetFAQImportProgress retrieves the progress of an FAQ import task
func (s *knowledgeService) GetFAQImportProgress(ctx context.Context, taskID string) (*types.FAQImportProgress, error) {
	if s.redisClient == nil {
		if v, ok := s.memFAQProgress.Load(taskID); ok {
			return v.(*types.FAQImportProgress), nil
		}
		return nil, werrors.NewNotFoundError("FAQ import task not found")
	}
	key := getFAQImportProgressKey(taskID)
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, werrors.NewNotFoundError("FAQ import task not found")
		}
		return nil, fmt.Errorf("failed to get FAQ import progress from Redis: %w", err)
	}

	var progress types.FAQImportProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FAQ import progress: %w", err)
	}

	// If task is completed, enrich with persisted result fields from database
	if progress.Status == types.FAQImportStatusCompleted && progress.KnowledgeID != "" {
		tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
		knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, progress.KnowledgeID)
		if err == nil && knowledge != nil {
			if result, err := knowledge.GetLastFAQImportResult(); err == nil && result != nil {
				progress.SuccessCount = result.SuccessCount
				progress.FailedCount = result.FailedCount
				progress.PartialFailedCount = result.PartialFailedCount
				progress.SkippedCount = result.SkippedCount
				progress.MergedCount = result.MergedCount
				progress.AddedCount = result.AddedCount
				progress.ImportMode = result.ImportMode
				progress.ImportedAt = result.ImportedAt
				progress.DisplayStatus = result.DisplayStatus
				progress.ProcessingTime = result.ProcessingTime
				if result.FailedEntriesURL != "" {
					progress.FailedEntriesURL = result.FailedEntriesURL
				}
			}
		}
	}

	return &progress, nil
}

// UpdateLastFAQImportResultDisplayStatus updates the display status of FAQ import result
func (s *knowledgeService) UpdateLastFAQImportResultDisplayStatus(ctx context.Context, kbID string, displayStatus string) error {
	// Validate the displayStatus parameter.
	if displayStatus != "open" && displayStatus != "close" {
		return werrors.NewBadRequestError("invalid display status, must be 'open' or 'close'")
	}

	kb, ctx, err := s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	tenantID := kb.TenantID

	// Find the FAQ-type knowledge.
	knowledgeList, err := s.repo.ListKnowledgeByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return fmt.Errorf("failed to list knowledge: %w", err)
	}

	// Find the FAQ-type knowledge.
	var faqKnowledge *types.Knowledge
	for _, k := range knowledgeList {
		if k.Type == types.KnowledgeTypeFAQ {
			faqKnowledge = k
			break
		}
	}

	if faqKnowledge == nil {
		return werrors.NewNotFoundError("FAQ knowledge not found in this knowledge base")
	}

	// Parse the current import result.
	result, err := faqKnowledge.GetLastFAQImportResult()
	if err != nil {
		return fmt.Errorf("failed to parse FAQ import result: %w", err)
	}

	if result == nil {
		return werrors.NewNotFoundError("no FAQ import result found")
	}

	// Update the display status.
	result.DisplayStatus = displayStatus

	// Save the updated result.
	if err := faqKnowledge.SetLastFAQImportResult(result); err != nil {
		return fmt.Errorf("failed to set FAQ import result: %w", err)
	}

	// Update the database.
	if err := s.repo.UpdateKnowledge(ctx, faqKnowledge); err != nil {
		return fmt.Errorf("failed to update knowledge: %w", err)
	}

	return nil
}
