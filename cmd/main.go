package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsoria-ttec/bne-converter/internal/config"
	"github.com/fsoria-ttec/bne-converter/internal/crawler"
	"github.com/fsoria-ttec/bne-converter/internal/logger"
	"github.com/fsoria-ttec/bne-converter/internal/monitor"
	"github.com/fsoria-ttec/bne-converter/internal/spinner"
	"github.com/sirupsen/logrus"
)

type RunMode struct {
	Manual      bool
	Monitor     bool
	ForceUpdate bool
	Debug       bool
}

func main() {

	// Configurar flags
	mode := parseFlags()

	// Detectar si la terminal soporta colores
	useColors := true
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		useColors = false
	}

	// Cargar configuración inicial y configurar logger
	cfg, err := config.Load()
	log := logrus.New()
	log.SetOutput(os.Stdout)

	if err != nil {
		log.Fatalf("Error al cargar configuración inicial: %v", err)
	}

	log.SetFormatter(logger.NewCustomFormatter(cfg.Logging, useColors))

	if mode.Debug {
		log.SetLevel(logrus.DebugLevel)
		log.Debugf("Debug activado")
	} else {
		log.SetLevel(cfg.Logging.GetLogLevel()) // establecer nivel de config.yaml
	}

	// Configurar contexto con cancelación
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inicializar crawler
	crw := crawler.New(cfg, log)

	// Manejar modo manual
	if mode.Manual {
		log.Info("Ejecutando en modo manual")
		if err := runManualMode(ctx, cfg, crw, log); err != nil {
			log.Fatalf("Error al ejecutar el modo manual: %v", err)
		}
		return
	}

	// Inicializar monitor (modo por defecto)
	mon := monitor.New(cfg, log)
	changes, errs := mon.Start(ctx)

	// Manejo de señales
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Manejar modo de forzar actualización
	if mode.ForceUpdate {
		log.Info("Actualización forzada solicitada")
		go func() {
			results := crw.DownloadAll(ctx)
			for _, result := range results {
				if result.Error != nil {
					log.Errorf("Error al descargar %s: %v", result.Category, result.Error)
					continue
				}
				log.Infof("Descarga completada para %s en %s", result.Category, result.FilePath)

				if err := processDownloadedFile(ctx, result.FilePath, log); err != nil {
					log.Errorf("Error al procesar %s: %v", result.FilePath, err)
				}
			}
		}()
	}

	// Procesar cambios y errores (modo monitor)
	log.Info("Ejecutando en modo monitor")

	// Iniciar spinner
	spin := spinner.New("Monitorizando cambios...")
	spin.Start(ctx)
	defer spin.Stop()

	for {
		select {
		case change, ok := <-changes:
			if !ok {
				log.Info("Canal de cambios cerrado")
				return
			}
			log.Infof("Cambio detectado en %s", change.URL)

			go func() {
				results := crw.DownloadAll(ctx)
				for _, result := range results {
					if result.Error != nil {
						log.Errorf("Error al descargar %s: %v", result.Category, result.Error)
						continue
					}
					log.Infof("Descarga completada para %s en %s", result.Category, result.FilePath)

					if err := processDownloadedFile(ctx, result.FilePath, log); err != nil {
						log.Errorf("Error al procesar %s: %v", result.FilePath, err)
					}
				}
			}()

		case err, ok := <-errs:
			if !ok {
				log.Info("Canal de errores cerrado")
				return
			}
			log.Errorf("Error al monitorizar: %v", err)

		case <-sigChan:
			spin.Stop()
			log.Info("Finalizando ejecución...")
			cancel()
			return
		}
	}

}

func parseFlags() RunMode {
	mode := RunMode{}

	flag.BoolVar(&mode.Manual, "manual", false, "Ejecutar en modo manual")
	flag.BoolVar(&mode.Monitor, "monitor", false, "Ejecutar en modo monitor")
	flag.BoolVar(&mode.ForceUpdate, "forzar", false, "Forzar actualización y monitorizar")
	flag.BoolVar(&mode.Debug, "debug", false, "Activar logs de debug")

	flag.Parse()
	return mode
}

func runManualMode(ctx context.Context, cfg *config.Config, crw *crawler.Crawler,
	log *logrus.Logger) error {
	log.Info("Ejecutando descarga...")

	results := crw.DownloadAll(ctx)

	var hasErrors bool
	for _, result := range results {
		if result.Error != nil {
			log.Errorf("Error al descargar %s: %v", result.Category, result.Error)
			hasErrors = true
			continue
		}
		log.Infof("Descarga completada para %s en %s", result.Category, result.FilePath)

		if err := processDownloadedFile(ctx, result.FilePath, log); err != nil {
			log.Errorf("Error al procesar %s: %v", result.FilePath, err)
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("algunas descargas o procesamientos fallaron, revisa los logs para más detalles")
	}

	log.Info("Ejecución manual completada con éxito")
	return nil
}

func processDownloadedFile(ctx context.Context, filePath string, logger *logrus.Logger) error {
	// TODO: Implementar procesamiento del archivo
	// - Validar XML
	// - Parsear contenido
	// - Transformar datos
	// - Almacenar en base de datos
	return nil
}
