package initializers

import (
	"fmt"
	"os"

	"github.com/johandrevandeventer/devices-api-server/internal/config"
	"github.com/johandrevandeventer/devices-api-server/internal/flags"
	coreutils "github.com/johandrevandeventer/devices-api-server/utils"
	"github.com/johandrevandeventer/textutils"
)

// InitConfig initializes the configuration file
func InitConfig() {
	if flags.FlagVerbose {
		coreutils.VerbosePrintln(textutils.BoldText("Initializing config..."))
	}

	newFiles, existingFiles, err := config.InitConfig()
	if err != nil {
		coreutils.VerbosePrintln(textutils.ColorText(textutils.Red, fmt.Sprintf("Failed to initialize configuration file: %s", err)))
		os.Exit(1)
	}

	if len(newFiles) > 0 {
		for _, file := range newFiles {
			coreutils.VerbosePrintln(textutils.ColorText(textutils.Green, fmt.Sprintf("-> Configuration file created: %s", file)))
		}

		coreutils.VerbosePrintln("")
	}

	if len(existingFiles) > 0 {
		for _, file := range existingFiles {
			coreutils.VerbosePrintln(textutils.ColorText(textutils.Yellow, fmt.Sprintf("-> Using configuration file: %s", file)))
		}

		coreutils.VerbosePrintln("")
	}

}
