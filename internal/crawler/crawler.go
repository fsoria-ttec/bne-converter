package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsoria-ttec/bne-converter/internal/config"
	"github.com/fsoria-ttec/bne-converter/internal/constants"
	"github.com/sirupsen/logrus" // logging
)

type Crawler struct {
	client    *http.Client
	config    *config.CrawlerConfig
	logger    *logrus.Logger
	semaphore chan struct{}
}

type DownloadResult struct {
	Category  string
	URL       string
	FilePath  string
	Error     error
	Timestamp time.Time
}

func New(cfg *config.Config, logger *logrus.Logger) *Crawler {
	return &Crawler{
		client: &http.Client{
			Timeout: time.Minute * 10, // timeout largo para archivos grandes
		},
		config:    &cfg.Crawler,
		logger:    logger,
		semaphore: make(chan struct{}, cfg.Crawler.MaxConcurrentDownloads),
	}
}

func (c *Crawler) DownloadAll(ctx context.Context) []DownloadResult {
	var wg sync.WaitGroup
	results := make([]DownloadResult, 0)
	resultsChan := make(chan DownloadResult, len(constants.BNECategories))

	// Iniciar descargas concurrentes
	for _, category := range constants.BNECategories {
		// Comprobar lista de categorias seleccionadas
		if len(c.config.ManualMode.SelectedCategories) > 0 {
			found := false
			for _, selectedCat := range c.config.ManualMode.SelectedCategories {
				if category.Id == selectedCat {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		wg.Add(1)
		go func(cat constants.Category) {
			defer wg.Done()

			// Adquirir semáforo
			c.semaphore <- struct{}{}
			defer func() { <-c.semaphore }()

			url := fmt.Sprintf("%s%s%s", c.config.BaseURL, cat.Id, constants.MRCFileSuffix)
			c.logger.Debugf("URL de descarga: %s", url)
			result := c.Download(ctx, cat.Id, url)

			select {
			case resultsChan <- result:
			case <-ctx.Done():
				return
			}
		}(category)
	}

	// Recoger resultados
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

func (c *Crawler) Download(ctx context.Context, category, url string) DownloadResult {
	result := DownloadResult{
		Category:  category,
		URL:       url,
		Timestamp: time.Now(),
	}

	var err error
	for attempt := 1; attempt <= c.config.RetryAttempts; attempt++ {
		result.FilePath, err = c.downloadFile(ctx, category, url)
		if err == nil {
			return result
		}

		if attempt < c.config.RetryAttempts {
			c.logger.Warnf("Intento nº %d fallido para %s: %v. Reintentando...", attempt, url, err)
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result
			case <-time.After(c.config.RetryDelay):
			}
		}
	}

	result.Error = fmt.Errorf("todos los intentos de descarga han fallado: %w", err)
	return result
}

func (c *Crawler) downloadFile(ctx context.Context, category, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("Error al crear petición: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error al realizar petición: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Código de respuesta inesperado: %d", resp.StatusCode)
	}

	// Crear directorio específico para cada categoría
	categoryDir := filepath.Join(c.config.DownloadPath, category)
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		return "", fmt.Errorf("Error creando directorio de descarga: %w", err)
	}

	// Generar nombre de archivo basado en la URL y timestamp
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(url))
	filePath := filepath.Join(categoryDir, fileName)

	// Crear el archivo
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("Error creando archivo: %w", err)
	}
	defer file.Close()

	// Copiar el contenido
	if _, err := io.Copy(file, resp.Body); err != nil {
		os.Remove(filePath) // limpiar archivo parcial en caso de error
		return "", fmt.Errorf("Error copiando contenido: %w", err)
	}

	return filePath, nil
}

func (c *Crawler) ValidateXML(filePath string) error {
	// TODO: Implementar validación de XML
	// - Verificar que el archivo es XML válido
	// - Validar contra schema si está disponible
	// - Verificar estructura esperada
	return nil
}
