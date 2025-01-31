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
	"github.com/fsoria-ttec/bne-converter/internal/metadata"
	"github.com/sirupsen/logrus" // logging
)

type Crawler struct {
	client    *http.Client
	config    *config.CrawlerConfig
	logger    *logrus.Logger
	semaphore chan struct{}
	metadata  *metadata.MetadataStore
}

type DownloadResult struct {
	Category     string
	URL          string
	FilePath     string
	Error        error
	Timestamp    time.Time
	LastModified time.Time
}

func New(cfg *config.Config, logger *logrus.Logger) (*Crawler, error) {
	metadataStore, err := metadata.NewMetadataStore(cfg.Crawler.DownloadPath)
	if err != nil {
		return nil, fmt.Errorf("error al inicializar el almacén de metadatos (%w)", err)
	}

	return &Crawler{
		client: &http.Client{
			Timeout: time.Minute * 10, // timeout largo para archivos grandes
		},
		config:    &cfg.Crawler,
		logger:    logger,
		semaphore: make(chan struct{}, cfg.Crawler.MaxConcurrentDownloads),
		metadata:  metadataStore,
	}, nil
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
			c.logger.Debugf("URL: %s", url)
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

	// Obtener información de archivo remoto y comparar
	needsUpdate, remoteLastModified, err := c.checkIfNeedsUpdate(ctx, category, url)
	if err != nil {
		result.Error = fmt.Errorf("error comprobando actualizaciones (%w)", err)
		return result
	}

	if !needsUpdate {
		c.logger.Infof("%s ya es la versión más reciente, omitiendo descarga", category)
		result.FilePath = filepath.Join(c.config.DownloadPath, category, fmt.Sprintf("%s%s", category, constants.MRCFileSuffix))
		result.LastModified, _ = c.metadata.GetLastModified(category)
		return result
	}

	var downloadErr error
	for attempt := 1; attempt <= c.config.RetryAttempts; attempt++ {
		result.FilePath, downloadErr = c.downloadFile(ctx, category, url, remoteLastModified)
		if downloadErr == nil {
			c.logger.Infof("Descarga completada: %s", category)
			result.LastModified = remoteLastModified
			return result
		}

		if attempt < c.config.RetryAttempts {
			c.logger.Warnf("Intento nº%d fallido para %s: %v. Reintentando...", attempt, url, downloadErr)
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

func (c *Crawler) checkIfNeedsUpdate(ctx context.Context, category, url string) (bool, time.Time, error) {
	// Crear petición HEAD
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, time.Time{}, fmt.Errorf("error creando petición HEAD (%w)", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, time.Time{}, fmt.Errorf("error realizando petición HEAD (%w)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, time.Time{}, fmt.Errorf("código de estado inesperado en HEAD (%d)", resp.StatusCode)
	}

	// Obtener Last-Modified del servidor
	remoteLastModified, err := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	if err != nil {
		remoteLastModified = time.Now() // si no se puede parsear fecha, actualizar
	}

	// Comprobar Last-Modified guardado
	localLastModified, exists := c.metadata.GetLastModified(category)
	if !exists {
		return true, remoteLastModified, nil // si no hay registro, descargar
	}

	// Comparar timestamps
	needsUpdate := remoteLastModified.After(localLastModified)
	c.logger.Debugf("%s: Remoto ->%v, Local ->%v)", category, remoteLastModified, localLastModified)

	return needsUpdate, remoteLastModified, nil
}

func (c *Crawler) downloadFile(ctx context.Context, category, url string, lastModified time.Time) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error al crear petición (%w)", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al realizar petición (%w)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("código de respuesta inesperado (%d)", resp.StatusCode)
	}

	// Crear directorios específicos para cada categoría
	categoryDir := filepath.Join(c.config.DownloadPath, category)
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		return "", fmt.Errorf("error creando directorio de descarga (%w)", err)
	}

	// Generar nombre de archivo: ID de categoria + URL
	filePath := filepath.Join(categoryDir, filepath.Base(url))

	// Comprobar si el archivo ya existe
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			return "", fmt.Errorf("error eliminando archivo existente (%w)", err)
		}
		c.logger.Debugf("Archivo preexistente eliminado: %s", filePath)
	}

	// Crear archivo
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error creando archivo (%w)", err)
	}
	defer file.Close()

	// Copiar contenido
	if _, err := io.Copy(file, resp.Body); err != nil {
		os.Remove(filePath) // limpiar archivo parcial en caso de error
		return "", fmt.Errorf("error copiando contenido (%w)", err)
	}

	// Actualizar metadatos
	if err := c.metadata.UpdateLastModified(category, lastModified); err != nil {
		c.logger.Warnf("Error al actualizar metadatos de %s: %v", category, err)
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
