package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/urmzd/graph-agent-dev-kit/kg/kgtypes"
)

// --- Mocks ---

type mockStore struct {
	entities          map[string]kgtypes.Entity
	relations         []kgtypes.Relation
	upsertEntityFunc  func(ctx context.Context, entity *kgtypes.ExtractedEntity, embedding []float32) (string, error)
	findByNameType    func(ctx context.Context, name, entityType string) ([]kgtypes.Entity, error)
	findByFuzzyName   func(ctx context.Context, name string, limit int) ([]kgtypes.Entity, error)
	createRelationFn  func(ctx context.Context, rel *kgtypes.RelationInput) (string, error)
	findRelsBetween   func(ctx context.Context, src, tgt string) ([]kgtypes.Relation, error)
	createEpisodeFn   func(ctx context.Context, input *kgtypes.EpisodeInput, uuids []string) (string, error)
	searchEmbeddingFn func(ctx context.Context, emb []float32, opts *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error)
	searchTextFn      func(ctx context.Context, query string, opts *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error)
	getGraphFn        func(ctx context.Context, limit int64) (*kgtypes.GraphData, error)
	getNodeFn         func(ctx context.Context, id string, depth int) (*kgtypes.NodeDetail, error)
	getProvenanceFn   func(ctx context.Context, factUUID string) ([]kgtypes.Episode, error)
	closed            bool
}

func newMockStore() *mockStore {
	return &mockStore{entities: make(map[string]kgtypes.Entity)}
}

func (m *mockStore) UpsertEntity(ctx context.Context, entity *kgtypes.ExtractedEntity, embedding []float32) (string, error) {
	if m.upsertEntityFunc != nil {
		return m.upsertEntityFunc(ctx, entity, embedding)
	}
	uuid := "entity-" + entity.Name
	m.entities[uuid] = kgtypes.Entity{UUID: uuid, Name: entity.Name, Type: entity.Type, Summary: entity.Summary}
	return uuid, nil
}

func (m *mockStore) GetEntity(_ context.Context, uuid string) (*kgtypes.Entity, error) {
	e, ok := m.entities[uuid]
	if !ok {
		return nil, kgtypes.ErrNodeNotFound
	}
	return &e, nil
}

func (m *mockStore) FindEntitiesByNameType(ctx context.Context, name, entityType string) ([]kgtypes.Entity, error) {
	if m.findByNameType != nil {
		return m.findByNameType(ctx, name, entityType)
	}
	return nil, nil
}

func (m *mockStore) FindEntitiesByFuzzyName(ctx context.Context, name string, limit int) ([]kgtypes.Entity, error) {
	if m.findByFuzzyName != nil {
		return m.findByFuzzyName(ctx, name, limit)
	}
	return nil, nil
}

func (m *mockStore) CreateRelation(ctx context.Context, rel *kgtypes.RelationInput) (string, error) {
	if m.createRelationFn != nil {
		return m.createRelationFn(ctx, rel)
	}
	uuid := "rel-" + rel.SourceUUID + "-" + rel.TargetUUID
	m.relations = append(m.relations, kgtypes.Relation{
		UUID: uuid, SourceUUID: rel.SourceUUID, TargetUUID: rel.TargetUUID,
		Type: rel.Type, Fact: rel.Fact, ValidAt: rel.ValidAt,
	})
	return uuid, nil
}

func (m *mockStore) InvalidateRelation(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (m *mockStore) FindRelationsBetweenEntities(ctx context.Context, src, tgt string) ([]kgtypes.Relation, error) {
	if m.findRelsBetween != nil {
		return m.findRelsBetween(ctx, src, tgt)
	}
	return nil, nil
}

func (m *mockStore) CreateEpisode(ctx context.Context, input *kgtypes.EpisodeInput, uuids []string) (string, error) {
	if m.createEpisodeFn != nil {
		return m.createEpisodeFn(ctx, input, uuids)
	}
	return "episode-1", nil
}

func (m *mockStore) SearchByEmbedding(ctx context.Context, emb []float32, opts *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error) {
	if m.searchEmbeddingFn != nil {
		return m.searchEmbeddingFn(ctx, emb, opts)
	}
	return nil, nil
}

func (m *mockStore) SearchByText(ctx context.Context, query string, opts *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error) {
	if m.searchTextFn != nil {
		return m.searchTextFn(ctx, query, opts)
	}
	return nil, nil
}

func (m *mockStore) GetGraph(ctx context.Context, limit int64) (*kgtypes.GraphData, error) {
	if m.getGraphFn != nil {
		return m.getGraphFn(ctx, limit)
	}
	return &kgtypes.GraphData{}, nil
}

func (m *mockStore) GetNode(ctx context.Context, id string, depth int) (*kgtypes.NodeDetail, error) {
	if m.getNodeFn != nil {
		return m.getNodeFn(ctx, id, depth)
	}
	return nil, kgtypes.ErrNodeNotFound
}

func (m *mockStore) GetFactProvenance(ctx context.Context, factUUID string) ([]kgtypes.Episode, error) {
	if m.getProvenanceFn != nil {
		return m.getProvenanceFn(ctx, factUUID)
	}
	return nil, nil
}

func (m *mockStore) Close(_ context.Context) error {
	m.closed = true
	return nil
}

type mockExtractor struct {
	entities  []kgtypes.ExtractedEntity
	relations []kgtypes.ExtractedRelation
	err       error
}

func (m *mockExtractor) Extract(_ context.Context, _ string) ([]kgtypes.ExtractedEntity, []kgtypes.ExtractedRelation, error) {
	return m.entities, m.relations, m.err
}

type mockEmbedder struct {
	embeddings [][]float32
	err        error
}

func (m *mockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

// --- Tests ---

func TestNew(t *testing.T) {
	store := newMockStore()
	eng := New(WithStore(store))

	if eng.store != store {
		t.Error("store not set")
	}
}

func TestApplyOntology(t *testing.T) {
	eng := New()
	ont := &kgtypes.Ontology{
		EntityTypes: []kgtypes.EntityTypeDef{{Name: "Person"}},
	}

	err := eng.ApplyOntology(context.Background(), ont)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eng.ontology != ont {
		t.Error("ontology not stored")
	}
}

func TestIngestEpisode_NoExtractor(t *testing.T) {
	eng := New(WithStore(newMockStore()))

	_, err := eng.IngestEpisode(context.Background(), &kgtypes.EpisodeInput{Body: "test"})
	if !errors.Is(err, kgtypes.ErrNoExtractor) {
		t.Errorf("error = %v, want ErrNoExtractor", err)
	}
}

func TestIngestEpisode_NoStore(t *testing.T) {
	eng := New(WithExtractor(&mockExtractor{}))

	_, err := eng.IngestEpisode(context.Background(), &kgtypes.EpisodeInput{Body: "test"})
	if !errors.Is(err, kgtypes.ErrStoreNotReady) {
		t.Errorf("error = %v, want ErrStoreNotReady", err)
	}
}

func TestIngestEpisode_Success(t *testing.T) {
	store := newMockStore()
	ext := &mockExtractor{
		entities: []kgtypes.ExtractedEntity{
			{Name: "Alice", Type: "Person", Summary: "A person"},
			{Name: "Acme", Type: "Organization", Summary: "A company"},
		},
		relations: []kgtypes.ExtractedRelation{
			{Source: "Alice", Target: "Acme", Type: "works_at", Fact: "Alice works at Acme"},
		},
	}
	emb := &mockEmbedder{}

	eng := New(WithStore(store), WithExtractor(ext), WithEmbedder(emb))

	result, err := eng.IngestEpisode(context.Background(), &kgtypes.EpisodeInput{
		Name: "test-episode", Body: "Alice works at Acme",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UUID != "episode-1" {
		t.Errorf("UUID = %q, want %q", result.UUID, "episode-1")
	}
	if len(result.EntityNodes) != 2 {
		t.Errorf("entities = %d, want 2", len(result.EntityNodes))
	}
	if len(result.EpisodicEdges) != 1 {
		t.Errorf("relations = %d, want 1", len(result.EpisodicEdges))
	}
}

func TestIngestEpisode_ExtractError(t *testing.T) {
	eng := New(
		WithStore(newMockStore()),
		WithExtractor(&mockExtractor{err: errors.New("LLM error")}),
	)

	_, err := eng.IngestEpisode(context.Background(), &kgtypes.EpisodeInput{Body: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIngestEpisode_RelationSkippedWhenEntityMissing(t *testing.T) {
	ext := &mockExtractor{
		entities: []kgtypes.ExtractedEntity{
			{Name: "Alice", Type: "Person", Summary: "A person"},
		},
		relations: []kgtypes.ExtractedRelation{
			// Bob was not extracted as an entity
			{Source: "Alice", Target: "Bob", Type: "knows", Fact: "Alice knows Bob"},
		},
	}

	eng := New(WithStore(newMockStore()), WithExtractor(ext))
	result, err := eng.IngestEpisode(context.Background(), &kgtypes.EpisodeInput{Body: "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.EpisodicEdges) != 0 {
		t.Errorf("relations = %d, want 0 (target entity missing)", len(result.EpisodicEdges))
	}
}

func TestIngestEpisode_DuplicateRelationSkipped(t *testing.T) {
	store := newMockStore()
	store.findRelsBetween = func(_ context.Context, _, _ string) ([]kgtypes.Relation, error) {
		return []kgtypes.Relation{
			{UUID: "existing", Fact: "Alice works at Acme", ValidAt: time.Now()},
		}, nil
	}

	ext := &mockExtractor{
		entities: []kgtypes.ExtractedEntity{
			{Name: "Alice", Type: "Person", Summary: "A person"},
			{Name: "Acme", Type: "Organization", Summary: "A company"},
		},
		relations: []kgtypes.ExtractedRelation{
			{Source: "Alice", Target: "Acme", Type: "works_at", Fact: "Alice works at Acme"},
		},
	}

	eng := New(WithStore(store), WithExtractor(ext))
	result, err := eng.IngestEpisode(context.Background(), &kgtypes.EpisodeInput{Body: "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.EpisodicEdges) != 0 {
		t.Errorf("relations = %d, want 0 (duplicate skipped)", len(result.EpisodicEdges))
	}
}

func TestSearchFacts_BM25Only(t *testing.T) {
	store := newMockStore()
	store.searchTextFn = func(_ context.Context, _ string, _ *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error) {
		return []kgtypes.ScoredFact{
			{Fact: kgtypes.Fact{UUID: "f1", FactText: "Alice works at Acme"}, Score: 1.0},
		}, nil
	}

	eng := New(WithStore(store))
	result, err := eng.SearchFacts(context.Background(), "Alice")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) != 1 {
		t.Errorf("facts = %d, want 1", len(result.Facts))
	}
}

func TestSearchFacts_HybridRRF(t *testing.T) {
	store := newMockStore()
	store.searchEmbeddingFn = func(_ context.Context, _ []float32, _ *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error) {
		return []kgtypes.ScoredFact{
			{Fact: kgtypes.Fact{UUID: "f1", FactText: "vector result"}, Score: 0.9},
			{Fact: kgtypes.Fact{UUID: "f2", FactText: "vector only"}, Score: 0.8},
		}, nil
	}
	store.searchTextFn = func(_ context.Context, _ string, _ *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error) {
		return []kgtypes.ScoredFact{
			{Fact: kgtypes.Fact{UUID: "f1", FactText: "vector result"}, Score: 1.0},
			{Fact: kgtypes.Fact{UUID: "f3", FactText: "text only"}, Score: 0.7},
		}, nil
	}

	eng := New(WithStore(store), WithEmbedder(&mockEmbedder{}))
	result, err := eng.SearchFacts(context.Background(), "test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) != 3 {
		t.Errorf("facts = %d, want 3", len(result.Facts))
	}
	// f1 should rank first (appears in both lists)
	if result.Facts[0].UUID != "f1" {
		t.Errorf("first fact = %q, want f1 (highest RRF score)", result.Facts[0].UUID)
	}
}

func TestSearchFacts_WithLimit(t *testing.T) {
	store := newMockStore()
	store.searchTextFn = func(_ context.Context, _ string, _ *kgtypes.SearchOptions) ([]kgtypes.ScoredFact, error) {
		return []kgtypes.ScoredFact{
			{Fact: kgtypes.Fact{UUID: "f1"}, Score: 1.0},
			{Fact: kgtypes.Fact{UUID: "f2"}, Score: 0.9},
			{Fact: kgtypes.Fact{UUID: "f3"}, Score: 0.8},
		}, nil
	}

	eng := New(WithStore(store))
	result, err := eng.SearchFacts(context.Background(), "test", kgtypes.WithLimit(2))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) != 2 {
		t.Errorf("facts = %d, want 2 (limited)", len(result.Facts))
	}
}

func TestGetEntity(t *testing.T) {
	store := newMockStore()
	store.entities["abc"] = kgtypes.Entity{UUID: "abc", Name: "Alice"}

	eng := New(WithStore(store))
	ent, err := eng.GetEntity(context.Background(), "abc")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ent.Name != "Alice" {
		t.Errorf("name = %q, want Alice", ent.Name)
	}
}

func TestGetEntity_NotFound(t *testing.T) {
	eng := New(WithStore(newMockStore()))
	_, err := eng.GetEntity(context.Background(), "nonexistent")

	if !errors.Is(err, kgtypes.ErrNodeNotFound) {
		t.Errorf("error = %v, want ErrNodeNotFound", err)
	}
}

func TestClose(t *testing.T) {
	store := newMockStore()
	eng := New(WithStore(store))

	err := eng.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.closed {
		t.Error("store was not closed")
	}
}

func TestClose_NilStore(t *testing.T) {
	eng := New()
	err := eng.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReciprocalRankFusion(t *testing.T) {
	listA := []kgtypes.ScoredFact{
		{Fact: kgtypes.Fact{UUID: "a", FactText: "fact a"}, Score: 1.0},
		{Fact: kgtypes.Fact{UUID: "b", FactText: "fact b"}, Score: 0.5},
	}
	listB := []kgtypes.ScoredFact{
		{Fact: kgtypes.Fact{UUID: "b", FactText: "fact b"}, Score: 1.0},
		{Fact: kgtypes.Fact{UUID: "c", FactText: "fact c"}, Score: 0.5},
	}

	facts := reciprocalRankFusion(listA, listB, 10)

	if len(facts) != 3 {
		t.Fatalf("facts = %d, want 3", len(facts))
	}
	// "b" appears in both lists at rank 1 and 0, so it should have highest RRF score
	if facts[0].UUID != "b" {
		t.Errorf("first = %q, want b (appears in both lists)", facts[0].UUID)
	}
}

func TestReciprocalRankFusion_Empty(t *testing.T) {
	facts := reciprocalRankFusion(nil, nil, 10)
	if len(facts) != 0 {
		t.Errorf("facts = %d, want 0", len(facts))
	}
}

func TestReciprocalRankFusion_LimitApplied(t *testing.T) {
	list := []kgtypes.ScoredFact{
		{Fact: kgtypes.Fact{UUID: "a"}, Score: 1.0},
		{Fact: kgtypes.Fact{UUID: "b"}, Score: 0.9},
		{Fact: kgtypes.Fact{UUID: "c"}, Score: 0.8},
	}

	facts := reciprocalRankFusion(list, nil, 2)
	if len(facts) != 2 {
		t.Errorf("facts = %d, want 2", len(facts))
	}
}
