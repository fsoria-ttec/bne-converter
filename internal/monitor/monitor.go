package monitor

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fsoria-ttec/bne-converter/internal/config"
	"github.com/sirupsen/logrus" // logging
)

type Monitor struct {
	client        *http.Client
	config        *config.MonitorConfig
	baseURL       string
	logger        *logrus.Logger
	lastCheckHash map[string]string
}

type FileChange struct {
	URL          string
	IsNew        bool
	LastModified time.Time
	ETag         string
}

func New(cfg *config.Config, logger *logrus.Logger) *Monitor {
	client := &http.Client{
		Timeout: cfg.Monitor.Timeout,
	}

	return &Monitor{
		client:        client,
		config:        &cfg.Monitor,
		baseURL:       cfg.Crawler.BaseURL,
		logger:        logger,
		lastCheckHash: make(map[string]string),
	}
}

func (m *Monitor) Start(ctx context.Context) (<-chan FileChange, <-chan error) {
	changes := make(chan FileChange)
	errs := make(chan error)

	go func() {
		defer close(changes)
		defer close(errs)

		ticker := time.NewTicker(m.config.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.checkForChanges(ctx, changes); err != nil {
					errs <- err
				}
			}
		}
	}()

	return changes, errs
}

func (m *Monitor) checkForChanges(ctx context.Context, changes chan<- FileChange) error {
	req, err := http.NewRequestWithContext(ctx, "GET", m.baseURL, nil)
	if err != nil {
		return fmt.Errorf("Error al crear petición: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("Error al realizar petición: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Código de estado inesperado: %d", resp.StatusCode)
	}

	// Leer y calcular el hash del contenido
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error al leer cuerpo de respuesta: %w", err)
	}

	hash := m.calculateHash(content)
	lastHash, exists := m.lastCheckHash[m.baseURL]

	if !exists || hash != lastHash {
		m.lastCheckHash[m.baseURL] = hash
		changes <- FileChange{
			URL:          m.baseURL,
			IsNew:        !exists,
			LastModified: m.parseLastModified(resp.Header.Get("Last-Modified")),
			ETag:         resp.Header.Get("ETag"),
		}
	}

	return nil
}

func (m *Monitor) calculateHash(content []byte) string {
	hasher := md5.New()
	hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (m *Monitor) parseLastModified(lastModified string) time.Time {
	if lastModified == "" {
		return time.Now()
	}

	t, err := time.Parse(time.RFC1123, lastModified)
	if err != nil {
		m.logger.Warnf("Error al parsear cabecera Last-Modified: %v", err)
		return time.Now()
	}

	return t
}
