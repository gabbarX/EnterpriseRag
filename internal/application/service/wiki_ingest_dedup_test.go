package service

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types/interfaces"
)

func TestDedupMergeRejectReason(t *testing.T) {
	cases := []struct {
		name        string
		src, dst    string
		candidates  map[string]bool
		wantAllowed bool
	}{
		{
			name:        "allowed: dst is a candidate for this item, same type prefix",
			src:         "entity/acme-corp",
			dst:         "entity/acme-corporation",
			candidates:  map[string]bool{"entity/acme-corporation": true},
			wantAllowed: true,
		},
		{
			name:        "rejected: dst similar to a DIFFERENT item (union hallucination)",
			src:         "entity/acme-open",
			dst:         "entity/hiring-agent",
			candidates:  map[string]bool{"entity/acme-ur": true}, // hiring-agent not here
			wantAllowed: false,
		},
		{
			name:        "rejected: dst is not a candidate at all (pure hallucination)",
			src:         "entity/hy3-preview",
			dst:         "entity/llm-cli-tool",
			candidates:  nil,
			wantAllowed: false,
		},
		{
			name:        "rejected: type mismatch even when dst is a candidate",
			src:         "entity/foo",
			dst:         "concept/foo",
			candidates:  map[string]bool{"concept/foo": true},
			wantAllowed: false,
		},
		{
			name:        "rejected: missing type prefix",
			src:         "foo",
			dst:         "entity/foo",
			candidates:  map[string]bool{"entity/foo": true},
			wantAllowed: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason := dedupMergeRejectReason(tc.src, tc.dst, tc.candidates)
			gotAllowed := reason == ""
			if gotAllowed != tc.wantAllowed {
				t.Fatalf("dedupMergeRejectReason(%q, %q) allowed=%v (reason=%q), want allowed=%v",
					tc.src, tc.dst, gotAllowed, reason, tc.wantAllowed)
			}
		})
	}
}

// helper: build a minimal entity/concept WikiPage.
func makePage(slug, title, typ string, aliases ...string) *types.WikiPage {
	return &types.WikiPage{
		Slug:     slug,
		Title:    title,
		PageType: typ,
		Aliases:  types.StringArray(aliases),
	}
}

func pageSlugs(pages []*types.WikiPage) []string {
	out := make([]string, 0, len(pages))
	for _, p := range pages {
		out = append(out, p.Slug)
	}
	return out
}

func containsSlug(pages []*types.WikiPage, slug string) bool {
	for _, p := range pages {
		if p.Slug == slug {
			return true
		}
	}
	return false
}

// Regression for the observed hallucination in llm_debug/20260422_171316.675.log:
// deepseek-v3.2 merged concept/varshik-arjit-avkash into
// concept/madhur-lokgeet-parampara despite zero character overlap.
// With the prefilter in place, the folk-music page should not even
// be offered to the LLM as a candidate for that new item.
func TestSelectDedupCandidatePages_FiltersUnrelatedHallucinationTarget(t *testing.T) {
	newItems := []extractedItem{
		{Slug: "concept/varshik-arjit-avkash", Name: "वार्षिक अर्जित अवकाश", Aliases: []string{"अर्जित अवकाश"}},
		{Slug: "entity/bhartiya-state-bank", Name: "भारतीय स्टेट बैंक"},
	}

	// Build a corpus that mirrors the real log's shape: ~90 unrelated pages
	// plus a few that share tokens with the new items (so the prefilter has
	// plausible near-matches to keep).
	pages := []*types.WikiPage{
		makePage("concept/madhur-lokgeet-parampara", "मधुर लोकगीत परंपरा", "concept"),
		makePage("concept/kaushal-vikas-prashikshan", "कौशल विकास प्रशिक्षण", "concept"),
		makePage("concept/prashikshan-mulyankan", "प्रशिक्षण मूल्यांकन", "concept"),
		makePage("concept/kritrim-buddhimatta-suraksha", "कृत्रिम बुद्धिमत्ता सुरक्षा", "concept", "AI सुरक्षा"),
		makePage("entity/maharashtra-shiksha-vibhag", "महाराष्ट्र शिक्षा विभाग", "entity", "शिक्षा विभाग"),
		makePage("entity/pune-vyavsayik-vidyalaya", "पुणे व्यावसायिक विद्यालय", "entity"),
	}
	// Pad with filler so we exceed dedupSmallCorpusBypass.
	for i := 0; i < 40; i++ {
		pages = append(pages, makePage(
			"concept/filler-"+strings.Repeat("x", i+1),
			"Filler Concept "+strings.Repeat("Alpha", i+1),
			"concept",
		))
	}

	got := selectDedupCandidatePages(newItems, pages)

	if containsSlug(got, "concept/madhur-lokgeet-parampara") {
		t.Fatalf("expected unrelated page to be filtered out, but got it in candidates: %v",
			pageSlugs(got))
	}
	if len(got) >= len(pages) {
		t.Fatalf("expected prefilter to shrink the corpus (%d pages), but kept %d",
			len(pages), len(got))
	}
}

// A related page (shares tokens / characters with a new item) must survive
// the prefilter so the LLM can still evaluate the merge.
func TestSelectDedupCandidatePages_KeepsRelatedPages(t *testing.T) {
	newItems := []extractedItem{
		{Slug: "concept/varshik-arjit-avkash", Name: "वार्षिक अर्जित अवकाश", Aliases: []string{"अर्जित अवकाश"}},
	}
	// Build > dedupSmallCorpusBypass pages so the filter actually runs.
	pages := []*types.WikiPage{
		// Directly related: existing page whose title overlaps with the new
		// item on "अर्जित अवकाश". Prefilter MUST keep this.
		makePage("concept/arjit-avkash", "अर्जित अवकाश", "concept", "वार्षिक अर्जित अवकाश"),
		// Unrelated.
		makePage("concept/madhur-lokgeet-parampara", "मधुर लोकगीत परंपरा", "concept"),
	}
	for i := 0; i < 30; i++ {
		pages = append(pages, makePage(
			"entity/filler-"+strings.Repeat("x", i+1),
			"Filler Entity "+strings.Repeat("Alpha", i+1),
			"entity",
		))
	}

	got := selectDedupCandidatePages(newItems, pages)

	if !containsSlug(got, "concept/arjit-avkash") {
		t.Fatalf("expected strongly-related page to be kept, got: %v", pageSlugs(got))
	}
}

// On small corpora the filter should be a no-op (minus page-type filtering):
// passing every page through is cheap and avoids cutting legitimate matches
// when the prompt is already small.
func TestSelectDedupCandidatePages_SmallCorpusBypass(t *testing.T) {
	newItems := []extractedItem{
		{Slug: "concept/a", Name: "A"},
	}
	pages := []*types.WikiPage{
		makePage("concept/wholly-unrelated-1", "असंबंधित एक", "concept"),
		makePage("concept/wholly-unrelated-2", "असंबंधित दो", "concept"),
		makePage("concept/wholly-unrelated-3", "असंबंधित तीन", "concept"),
	}
	got := selectDedupCandidatePages(newItems, pages)
	if len(got) != len(pages) {
		t.Fatalf("expected bypass on small corpus: got %d, want %d", len(got), len(pages))
	}
}

// Non-entity/concept pages (summaries, comparisons, …) must be stripped regardless
// of corpus size — they are never valid merge targets.
func TestSelectDedupCandidatePages_DropsNonEntityConcept(t *testing.T) {
	newItems := []extractedItem{
		{Slug: "concept/foo", Name: "Foo"},
	}
	pages := []*types.WikiPage{
		makePage("summary/some-doc", "Some Doc Summary", types.WikiPageTypeSummary),
		makePage("comparison/foo-vs-bar", "Foo vs Bar", types.WikiPageTypeComparison),
		makePage("concept/foo-related", "Foo Related", types.WikiPageTypeConcept),
	}
	got := selectDedupCandidatePages(newItems, pages)
	for _, p := range got {
		if p.PageType != types.WikiPageTypeEntity && p.PageType != types.WikiPageTypeConcept {
			t.Fatalf("non-entity/concept page should have been filtered: %s (%s)",
				p.Slug, p.PageType)
		}
	}
}

// surfaceGrams must yield empty intersection for the real hallucinated pair,
// confirming the underlying similarity signal is doing its job.
func TestSurfaceGrams_UnrelatedDevanagariPair(t *testing.T) {
	a := surfaceGrams("वार्षिक अर्जित अवकाश")
	b := surfaceGrams("मधुर लोकगीत परंपरा")
	for k := range a {
		if _, ok := b[k]; ok {
			t.Fatalf("expected zero bigram overlap, but shared %q", k)
		}
	}
}

// Latin abbreviation ↔ full-name pair must score highly (> floor) so the
// filter keeps legitimate merge candidates like "Acme Corp" ↔ "Acme Corporation".
func TestDedupPairScore_AcmeCorpVariant(t *testing.T) {
	a := dedupSurface{
		slugTokens:   slugBaseTokens("entity/acme-corp"),
		nameGramSets: gramsPerSurface([]string{"Acme Corp"}),
	}
	b := dedupSurface{
		slugTokens:   slugBaseTokens("entity/acme-corporation"),
		nameGramSets: gramsPerSurface([]string{"Acme Corporation"}),
	}
	score := dedupPairScore(a, b)
	if score < dedupCandidateScoreFloor {
		t.Fatalf("expected Acme Corp ↔ Corporation score above floor %v, got %v",
			dedupCandidateScoreFloor, score)
	}
}

// Unrelated non-Latin pair (the exact case observed in production) must score 0.
func TestDedupPairScore_UnrelatedDevanagariPair(t *testing.T) {
	a := dedupSurface{
		slugTokens:   slugBaseTokens("concept/varshik-arjit-avkash"),
		nameGramSets: gramsPerSurface([]string{"वार्षिक अर्जित अवकाश", "अर्जित अवकाश"}),
	}
	b := dedupSurface{
		slugTokens:   slugBaseTokens("concept/madhur-lokgeet-parampara"),
		nameGramSets: gramsPerSurface([]string{"मधुर लोकगीत परंपरा"}),
	}
	if s := dedupPairScore(a, b); s >= dedupCandidateScoreFloor {
		t.Fatalf("expected unrelated pair to score below floor %v, got %v",
			dedupCandidateScoreFloor, s)
	}
}

func TestNormalizeWikiIdentityTitlePreservesSemanticPunctuation(t *testing.T) {
	if got := normalizeWikiIdentityTitle("  Acme  Corp "); got != "acmecorp" {
		t.Fatalf("normalizeWikiIdentityTitle whitespace/case = %q, want acmecorp", got)
	}
	if normalizeWikiIdentityTitle("कहानी") == normalizeWikiIdentityTitle("«कहानी»") {
		t.Fatal("concept title and work/chapter title must keep distinct identities")
	}
}

func TestExactIdentityTargetSameTypeOnly(t *testing.T) {
	item := extractedItem{Name: "प्रेमचंद", Slug: "entity/premchand"}
	pages := map[string]*types.WikiPageLite{
		"entity/prem-chand": {
			Slug:     "entity/prem-chand",
			Title:    "प्रेम चंद",
			PageType: types.WikiPageTypeEntity,
		},
		"concept/prem-chand": {
			Slug:     "concept/prem-chand",
			Title:    "प्रेमचंद",
			PageType: types.WikiPageTypeConcept,
		},
	}
	candidates := map[string]bool{
		"entity/prem-chand":  true,
		"concept/prem-chand": true,
	}
	if got := exactIdentityTarget(item, types.WikiPageTypeEntity, candidates, pages); got != "entity/prem-chand" {
		t.Fatalf("exactIdentityTarget = %q, want entity/prem-chand", got)
	}
}

func TestWikiIdentityClaimConvergesDifferentSlugs(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	svc := &wikiIngestService{redisClient: rdb}
	ctx := context.Background()

	proposals := []string{
		"entity/premchand",
		"entity/prem-chand",
		"entity/premchand-hindi",
		"entity/dhanpat-rai",
	}
	results := make([]string, len(proposals))
	var wg sync.WaitGroup
	for i, proposal := range proposals {
		i, proposal := i, proposal
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = svc.claimWikiIdentitySlug(
				ctx, "kb-1", types.WikiPageTypeEntity, "प्रेम चंद", proposal, false, &WikiBatchContext{},
			)
		}()
	}
	wg.Wait()
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatalf("concurrent identity claims did not converge: %#v", results)
		}
	}

	// A verified existing page is authoritative and replaces a provisional
	// reservation left by an earlier map worker.
	authoritative := svc.claimWikiIdentitySlug(
		ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/prem-chand", true, &WikiBatchContext{},
	)
	third := svc.claimWikiIdentitySlug(
		ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/premchand-hindi", false, &WikiBatchContext{},
	)
	if authoritative != "entity/prem-chand" || third != authoritative {
		t.Fatalf("authoritative identity did not replace claim: authoritative=%q third=%q", authoritative, third)
	}
}

func TestWikiIdentityClaimKeepsDistinctTypesAndPunctuation(t *testing.T) {
	batch := &WikiBatchContext{}
	svc := &wikiIngestService{}
	ctx := context.Background()

	concept := svc.claimWikiIdentitySlug(
		ctx, "kb-1", types.WikiPageTypeConcept, "कहानी", "concept/kahani-concept", false, batch,
	)
	chapter := svc.claimWikiIdentitySlug(
		ctx, "kb-1", types.WikiPageTypeConcept, "«कहानी»", "concept/kahani-chapter", false, batch,
	)
	entity := svc.claimWikiIdentitySlug(
		ctx, "kb-1", types.WikiPageTypeEntity, "कहानी", "entity/kahani-book", false, batch,
	)
	if concept != "concept/kahani-concept" || chapter != "concept/kahani-chapter" || entity != "entity/kahani-book" {
		t.Fatalf("distinct identities were collapsed: concept=%q chapter=%q entity=%q", concept, chapter, entity)
	}
}

func TestStabilizeExtractedIdentitiesCoalescesEvidence(t *testing.T) {
	svc := &wikiIngestService{}
	batch := &WikiBatchContext{}
	items := []extractedItem{
		{
			Name: "प्रेम चंद", Slug: "entity/premchand", Aliases: []string{"धनपत राय"},
			Description: "लेखक", Details: "संक्षिप्त", SourceChunks: []string{"chunk-1"},
		},
		{
			Name: "प्रेमचंद", Slug: "entity/prem-chand", Aliases: []string{"Premchand"},
			Description: "हिंदी और उर्दू के प्रसिद्ध उपन्यासकार", Details: "अधिक विस्तृत विवरण", SourceChunks: []string{"chunk-2"},
		},
	}
	got := svc.stabilizeExtractedIdentities(
		context.Background(), "kb-1", types.WikiPageTypeEntity, items, nil, nil, batch,
	)
	if len(got) != 1 {
		t.Fatalf("stabilized item count = %d, want 1", len(got))
	}
	if got[0].Slug != "entity/premchand" {
		t.Fatalf("stabilized slug = %q, want first claimed slug", got[0].Slug)
	}
	if got[0].Name != "प्रेमचंद" {
		t.Fatalf("compact display name not preferred: %#v", got[0])
	}
	if got[0].Description != "हिंदी और उर्दू के प्रसिद्ध उपन्यासकार" || got[0].Details != "अधिक विस्तृत विवरण" {
		t.Fatalf("richer fallback text not preserved: %#v", got[0])
	}
	if len(got[0].SourceChunks) != 2 {
		t.Fatalf("source chunks were not unioned: %#v", got[0].SourceChunks)
	}
}

func TestWikiIdentityClaimRedisOverridesStaleLocal(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	svc := &wikiIngestService{redisClient: rdb}
	ctx := context.Background()
	batch := &WikiBatchContext{}
	identity := normalizeWikiIdentityTitle("प्रेमचंद")
	batch.identityClaims.Store(types.WikiPageTypeEntity+"\x00"+identity, "entity/premchand")
	if err := rdb.Set(ctx, wikiIdentityClaimPrefix+"kb-1:"+types.WikiPageTypeEntity+":"+identity, "entity/prem-chand", wikiIdentityClaimTTL).Err(); err != nil {
		t.Fatalf("seed redis claim: %v", err)
	}

	got := svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/munshi-premchand", false, batch)
	if got != "entity/prem-chand" {
		t.Fatalf("redis claim should beat stale local map: got %q", got)
	}
	if stored, _ := batch.identityClaims.Load(types.WikiPageTypeEntity + "\x00" + identity); stored != "entity/prem-chand" {
		t.Fatalf("local cache not refreshed from redis: %#v", stored)
	}
}

func TestStabilizeLLMMergeDoesNotOverrideRedisClaim(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	svc := &wikiIngestService{redisClient: rdb}
	ctx := context.Background()
	batch := &WikiBatchContext{}

	first := svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/premchand", false, batch)
	if first != "entity/premchand" {
		t.Fatalf("provisional claim = %q, want entity/premchand", first)
	}

	got := svc.stabilizeExtractedIdentities(ctx, "kb-1", types.WikiPageTypeEntity, []extractedItem{
		{Name: "प्रेमचंद", Slug: "entity/prem-chand"},
	}, map[string]string{"entity/prem-chand": "entity/premchand-institute"}, nil, batch)
	if len(got) != 1 || got[0].Slug != "entity/premchand" {
		t.Fatalf("LLM merge overwrote identity claim: %#v", got)
	}
}

func TestStabilizeExactTargetIsAuthoritative(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	svc := &wikiIngestService{redisClient: rdb}
	ctx := context.Background()
	batch := &WikiBatchContext{}

	svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/premchand", false, batch)
	got := svc.stabilizeExtractedIdentities(ctx, "kb-1", types.WikiPageTypeEntity, []extractedItem{
		{Name: "प्रेमचंद", Slug: "entity/premchand"},
	}, nil, map[string]string{"entity/premchand": "entity/prem-chand"}, batch)
	if len(got) != 1 || got[0].Slug != "entity/prem-chand" {
		t.Fatalf("exact existing page should replace provisional claim: %#v", got)
	}

	follow := svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/munshi-premchand", false, &WikiBatchContext{})
	if follow != "entity/prem-chand" {
		t.Fatalf("authoritative exact hit did not stick in redis: %q", follow)
	}
}

func TestWikiIdentityClaimLiteConcurrentSameBatch(t *testing.T) {
	svc := &wikiIngestService{}
	batch := &WikiBatchContext{}
	ctx := context.Background()
	proposals := []string{"entity/premchand", "entity/prem-chand", "entity/premchand-hindi"}
	results := make([]string, len(proposals))
	var wg sync.WaitGroup
	for i, proposal := range proposals {
		i, proposal := i, proposal
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", proposal, false, batch)
		}()
	}
	wg.Wait()
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatalf("lite same-batch claims did not converge: %#v", results)
		}
	}
}

func TestRemapSlugUpdatesByIdentityConverges(t *testing.T) {
	svc := &wikiIngestService{}
	batch := &WikiBatchContext{}
	ctx := context.Background()
	svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/premchand", false, batch)

	got := svc.remapSlugUpdatesByIdentity(ctx, "kb-1", map[string][]SlugUpdate{
		"entity/premchand": {{
			Slug: "entity/premchand", Type: types.WikiPageTypeEntity,
			Item: extractedItem{Name: "प्रेमचंद", Slug: "entity/premchand"},
		}},
		"entity/prem-chand": {{
			Slug: "entity/prem-chand", Type: types.WikiPageTypeEntity,
			Item: extractedItem{Name: "प्रेम चंद", Slug: "entity/prem-chand"},
		}},
		"summary/doc": {{Slug: "summary/doc", Type: types.WikiPageTypeSummary}},
	}, batch)
	if len(got["entity/premchand"]) != 2 {
		t.Fatalf("entity updates should coalesce onto claimed slug: %#v", got)
	}
	for _, u := range got["entity/premchand"] {
		if u.Item.Slug != "entity/premchand" {
			t.Fatalf("remap left Item.Slug stale: %#v", u)
		}
	}
	if len(got["summary/doc"]) != 1 {
		t.Fatalf("summary slug should stay untouched: %#v", got)
	}
	if _, ok := got["entity/prem-chand"]; ok {
		t.Fatalf("unclaimed romanization should have been remapped away: %#v", got)
	}
}

func TestReclaimExtractedIdentitiesCoalescesCitationSlugs(t *testing.T) {
	svc := &wikiIngestService{}
	batch := &WikiBatchContext{}
	svc.claimWikiIdentitySlug(context.Background(), "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/premchand", false, batch)

	entities, concepts := svc.reclaimExtractedIdentities(context.Background(), "kb-1", []extractedItem{
		{Name: "प्रेमचंद", Slug: "entity/premchand", SourceChunks: []string{"c1"}},
		{Name: "प्रेम चंद", Slug: "entity/prem-chand", SourceChunks: []string{"c2"}},
	}, nil, batch)
	if len(entities) != 1 || entities[0].Slug != "entity/premchand" {
		t.Fatalf("citation slugs did not reclaim onto identity: entities=%#v", entities)
	}
	if len(entities[0].SourceChunks) != 2 {
		t.Fatalf("reclaim dropped citation evidence: %#v", entities[0])
	}
	if len(concepts) != 0 {
		t.Fatalf("unexpected concepts: %#v", concepts)
	}
}

func TestWikiIdentityClaimReplacesInvalidRedisValue(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	svc := &wikiIngestService{redisClient: rdb}
	ctx := context.Background()
	identity := normalizeWikiIdentityTitle("प्रेमचंद")
	if err := rdb.Set(ctx, wikiIdentityClaimPrefix+"kb-1:"+types.WikiPageTypeEntity+":"+identity, "garbage", wikiIdentityClaimTTL).Err(); err != nil {
		t.Fatalf("seed invalid redis claim: %v", err)
	}

	first := svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/premchand", false, &WikiBatchContext{})
	if first != "entity/premchand" {
		t.Fatalf("invalid redis value should be replaced: got %q", first)
	}
	second := svc.claimWikiIdentitySlug(ctx, "kb-1", types.WikiPageTypeEntity, "प्रेमचंद", "entity/prem-chand", false, &WikiBatchContext{})
	if second != "entity/premchand" {
		t.Fatalf("callers did not converge after replacing invalid value: %q", second)
	}
}

type stubNormalizedTitleWiki struct {
	interfaces.WikiPageService
	mu    sync.Mutex
	calls int
	last  []string
	pages []*types.WikiPageLite
}

func (s *stubNormalizedTitleWiki) FindPagesByNormalizedTitles(
	_ context.Context, _, _ string, identities []string,
) ([]*types.WikiPageLite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.last = append([]string(nil), identities...)
	want := make(map[string]bool, len(identities))
	for _, id := range identities {
		want[id] = true
	}
	out := make([]*types.WikiPageLite, 0, len(s.pages))
	for _, p := range s.pages {
		if p != nil && want[normalizeWikiIdentityTitle(p.Title)] {
			out = append(out, p)
		}
	}
	return out, nil
}

func TestAttachExactIdentityPagesBatchesAndCaches(t *testing.T) {
	stub := &stubNormalizedTitleWiki{
		pages: []*types.WikiPageLite{
			{Slug: "entity/prem-chand", Title: "प्रेम चंद", PageType: types.WikiPageTypeEntity},
		},
	}
	svc := &wikiIngestService{wikiService: stub}
	batch := &WikiBatchContext{}
	ctx := context.Background()
	items := []extractedItem{
		{Name: "प्रेमचंद", Slug: "entity/premchand"},
		{Name: "प्रेम चंद", Slug: "entity/munshi-premchand"},
		{Name: "निराला", Slug: "entity/nirala"},
	}

	candidatePages := make(map[string]*types.WikiPageLite)
	itemCandidates := make(map[string]map[string]bool)
	svc.attachExactIdentityPages(ctx, "kb-1", types.WikiPageTypeEntity, items, candidatePages, itemCandidates, batch)
	if stub.calls != 1 {
		t.Fatalf("expected 1 batched lookup, got %d identities=%v", stub.calls, stub.last)
	}
	if !itemCandidates["entity/premchand"]["entity/prem-chand"] || !itemCandidates["entity/munshi-premchand"]["entity/prem-chand"] {
		t.Fatalf("exact page not bound to both romanizations: %#v", itemCandidates)
	}
	if itemCandidates["entity/nirala"]["entity/prem-chand"] {
		t.Fatalf("premchand page leaked onto निराला: %#v", itemCandidates)
	}

	svc.attachExactIdentityPages(ctx, "kb-1", types.WikiPageTypeEntity, items, candidatePages, itemCandidates, batch)
	if stub.calls != 1 {
		t.Fatalf("batch cache should skip the second lookup, got %d", stub.calls)
	}

	svc.attachExactIdentityPages(ctx, "kb-1", types.WikiPageTypeEntity, append(items,
		extractedItem{Name: "महादेवी", Slug: "entity/mahadevi"},
	), candidatePages, itemCandidates, batch)
	if stub.calls != 2 {
		t.Fatalf("cache miss should query only the new identity, got %d last=%v", stub.calls, stub.last)
	}
	if len(stub.last) != 1 || stub.last[0] != normalizeWikiIdentityTitle("महादेवी") {
		t.Fatalf("second lookup should only ask for महादेवी: %v", stub.last)
	}
}
