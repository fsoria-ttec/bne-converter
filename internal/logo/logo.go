package logo

import (
	"fmt"
	"strings"

	"github.com/fsoria-ttec/bne-converter/internal/config"
	"github.com/sirupsen/logrus"
)

const asciiArt = `
██████╗ ███╗   ██╗███████╗
██╔══██╗████╗  ██║██╔════╝
██████╔╝██╔██╗ ██║█████╗  
██╔══██╗██║╚██╗██║██╔══╝  
██████╔╝██║ ╚████║███████╗
╚═════╝ ╚═╝  ╚═══╝╚══════╝
| ►  c o n v e r t e r ► |
`

func Print(log *logrus.Logger, cfg *config.Config) {
	lines := strings.Split(strings.TrimSpace(asciiArt), "\n")

	fmt.Println()
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()

	fmt.Printf("------ Versión %s ------", cfg.Version) // información de versión
	fmt.Println()
}
