package initializers

import (
	"fmt"
	"os"

	"github.com/johandrevandeventer/devices-api-server/internal/flags"
	devicesdb "github.com/johandrevandeventer/devices-api-server/pkg/db"
	"github.com/johandrevandeventer/devices-api-server/pkg/db/models"
	coreutils "github.com/johandrevandeventer/devices-api-server/utils"
	"github.com/johandrevandeventer/textutils"
)

func InitDB() {
	_, err := devicesdb.NewDB()
	if err != nil {
		fmt.Println(textutils.BoldText("Initializing db..."))
		fmt.Println(textutils.ColorText(textutils.Red, fmt.Sprintf("Failed to initialize db: %s", err)))
		os.Exit(1)
	}

	if flags.FlagVerbose {
		coreutils.VerbosePrintln(textutils.BoldText("Initializing db..."))
	}

	initTables(devicesdb.BMS_DB_Instance)

	// defer db.Close()
	// db.Migrate("auth_tokens", models.AuthToken{})
	// db.Migrate("customers", models.Customer{})
	// db.Migrate("sites", models.Site{})
	// db.Migrate("devices", models.Device{})
	// db.Migrate("device_statuses", models.DeviceStatus{})
}

func initTables(db *devicesdb.BMS_DB) {
	tablesList := []string{
		"auth_tokens",
		"customers",
		"sites",
		"devices",
		"device_statuses",
	}

	existingTablesList := []string{}
	newTablesList := []string{}

	for _, table := range tablesList {
		if !db.TableExists(table) {
			newTablesList = append(newTablesList, table)
		} else {
			existingTablesList = append(existingTablesList, table)
		}
	}

	if len(existingTablesList) > 0 {
		for _, table := range existingTablesList {
			fmt.Println(textutils.ColorText(textutils.Yellow, fmt.Sprintf("-> Table exists: %s", table)))
		}
	}

	if len(newTablesList) > 0 {
		for _, table := range newTablesList {
			switch table {
			case "auth_tokens":
				db.Migrate("auth_tokens", models.AuthToken{})
			case "customers":
				db.Migrate("customers", models.Customer{})
			case "sites":
				db.Migrate("sites", models.Site{})
			case "devices":
				db.Migrate("devices", models.Device{})
			case "device_statuses":
				db.Migrate("device_statuses", models.DeviceStatus{})
			}

			fmt.Println(textutils.ColorText(textutils.Green, fmt.Sprintf("-> Table created: %s", table)))
		}
	}
}
