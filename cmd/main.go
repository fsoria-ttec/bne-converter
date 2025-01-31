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
	"github.com/fsoria-ttec/bne-converter/internal/logo"
	"github.com/fsoria-ttec/bne-converter/internal/monitor"
	"github.com/fsoria-ttec/bne-converter/internal/spinner"
	"github.com/sirupsen/logrus" // logging
)

type RunMode struct {
	Manual      bool
	Monitor     bool
	ForceUpdate bool
	Debug       bool
	Version     bool
}

func main() {

	// Flags
	mode := parseFlags()

	// Detectar si terminal soporta colores
	useColors := true
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		useColors = false
	}

	// Configuración inicial
	cfg, err := config.Load()

	// Logger
	log := logrus.New()
	log.SetOutput(os.Stdout)

	if err != nil {
		log.Fatalf("Error al cargar configuración inicial: %v", err)
	}

	log.SetFormatter(logger.NewCustomFormatter(cfg.Logging, useColors))

	if mode.Version {
		logo.Print(log, cfg)
		os.Exit(0)
	}

	// Manejar modo -debug
	if mode.Debug {
		log.SetLevel(logrus.DebugLevel)
		log.Debugf("Modo Debug activo")
	} else {
		log.SetLevel(cfg.Logging.GetLogLevel()) // obtener nivel de config.yaml
	}

	// Contexto con cancelación
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inicializar crawler
	crw, err := crawler.New(cfg, log)
	if err != nil {
		log.Fatalf("Error al inicializar crawler: %v", err)
	}

	// Manejar modo -manual
	if mode.Manual {
		log.Info("Modo Manual activo")
		if err := runManualMode(ctx, cfg, crw, log); err != nil {
			log.Fatalf("Error al ejecutar el modo manual: %v", err)
		}
		return
	}

	// Manejar señales
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Manejar modo -forzar
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

	// Manejar modo monitor (opción por defecto)
	mon := monitor.New(cfg, log)
	changes, errs := mon.Start(ctx)
	log.Info("Modo Monitor activo")

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

	flag.BoolVar(&mode.Manual, "manual", false, "Ejecutar en Modo Manual: comprobar descarga de los archivos MARC para actualizar BBDD y CKAN")
	flag.BoolVar(&mode.Monitor, "monitor", false, "Ejecutar en Modo Monitor: monitorizar web del BNE en tiempo real para comprobar actualizaciones de ficheros MARC")
	flag.BoolVar(&mode.ForceUpdate, "forzar", false, "Forzar actualización y monitorizar")
	flag.BoolVar(&mode.Debug, "debug", false, "Activar logs de debug")
	flag.BoolVar(&mode.Version, "version", false, "Información de versión")

	flag.Parse()
	return mode
}

func runManualMode(ctx context.Context, cfg *config.Config, crw *crawler.Crawler,
	log *logrus.Logger) error {
	log.Infof("Ejecutando descarga en %s...", cfg.Crawler.DownloadPath)

	results := crw.DownloadAll(ctx)

	var hasErrors bool
	for _, result := range results {
		if result.Error != nil {
			hasErrors = true
			continue
		}

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
