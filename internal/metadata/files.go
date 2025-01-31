package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileMetadata struct {
	Category     string    `json:"category"`
	LastModified time.Time `json:"last_modified"`
	LastChecked  time.Time `json:"last_checked"`
}

type MetadataStore struct {
	Files map[string]FileMetadata `json:"files"`
	path  string
}

func NewMetadataStore(basePath string) (*MetadataStore, error) {
	metadataPath := filepath.Join(basePath, "metadata.json")
	store := &MetadataStore{
		Files: make(map[string]FileMetadata),
		path:  metadataPath,
	}

	// Intentar cargar metadatos existentes
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading metadata: %w", err)
	}

	return store, nil
}

func (m *MetadataStore) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.Files)
}

func (m *MetadataStore) save() error {
	data, err := json.MarshalIndent(m.Files, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	return os.WriteFile(m.path, data, 0644)
}

func (m *MetadataStore) UpdateLastModified(category string, lastModified time.Time) error {
	m.Files[category] = FileMetadata{
		Category:     category,
		LastModified: lastModified,
		LastChecked:  time.Now(),
	}
	return m.save()
}

func (m *MetadataStore) GetLastModified(category string) (time.Time, bool) {
	if metadata, exists := m.Files[category]; exists {
		return metadata.LastModified, true
	}
	return time.Time{}, false
}
