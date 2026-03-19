package kg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urmzd/graph-agent-dev-kit/kg/internal/engine"
	"github.com/urmzd/graph-agent-dev-kit/kg/kgtypes"
	kgsurrealdb "github.com/urmzd/graph-agent-dev-kit/kg/surrealdb"
)

// Config holds configuration for creating a Graph.
type Config struct {
	SurrealDBURL string
	Namespace    string
	Database     string
	Username     string
	Password     string
	Extractor    kgtypes.Extractor
	Embedder     kgtypes.Embedder
	Logger       *slog.Logger
	Store        kgtypes.Store
}

// Option configures kg.
type Option func(*Config)

// WithSurrealDB configures SurrealDB connection.
func WithSurrealDB(url, namespace, database, username, password string) Option {
	return func(c *Config) {
		c.SurrealDBURL = url
		c.Namespace = namespace
		c.Database = database
		c.Username = username
		c.Password = password
	}
}

// WithExtractor sets the entity/relation extractor.
func WithExtractor(ext kgtypes.Extractor) Option {
	return func(c *Config) {
		c.Extractor = ext
	}
}

// WithEmbedder sets the vector embedder.
func WithEmbedder(emb kgtypes.Embedder) Option {
	return func(c *Config) {
		c.Embedder = emb
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithStore sets a pre-created store, skipping automatic store creation.
// Use this when you need direct access to the store (e.g. for DB connection sharing).
func WithStore(s kgtypes.Store) Option {
	return func(c *Config) {
		c.Store = s
	}
}

// NewGraph creates a new Graph using the provided options.
// This wires up the GraphEngine with the configured Store, Extractor, and Embedder.
func NewGraph(ctx context.Context, opts ...Option) (kgtypes.Graph, error) {
	cfg := &Config{}
	for _, o := range opts {
		o(cfg)
	}

	var store kgtypes.Store
	if cfg.Store != nil {
		store = cfg.Store
	} else if cfg.SurrealDBURL != "" {
		var err error
		store, err = kgsurrealdb.NewStore(ctx, kgsurrealdb.StoreConfig{
			URL:       cfg.SurrealDBURL,
			Namespace: cfg.Namespace,
			Database:  cfg.Database,
			Username:  cfg.Username,
			Password:  cfg.Password,
			Logger:    cfg.Logger,
		})
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no backend configured: use WithSurrealDB or WithStore")
	}

	engineOpts := []engine.Option{
		engine.WithStore(store),
	}
	if cfg.Extractor != nil {
		engineOpts = append(engineOpts, engine.WithExtractor(cfg.Extractor))
	}
	if cfg.Embedder != nil {
		engineOpts = append(engineOpts, engine.WithEmbedder(cfg.Embedder))
	}
	if cfg.Logger != nil {
		engineOpts = append(engineOpts, engine.WithLogger(cfg.Logger))
	}

	return engine.New(engineOpts...), nil
}
