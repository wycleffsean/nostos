package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// LogMessage is the JSON structure for LSP logging
type LogMessage struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Type    int    `json:"type"`
		Message string `json:"message"`
	} `json:"params"`
}

func sendLogMessage(message string) {
	logMsg := LogMessage{
		Jsonrpc: "2.0",
		Method:  "window/logMessage",
	}
	logMsg.Params.Type = 3 // Info level
	logMsg.Params.Message = message

	jsonData, _ := json.Marshal(logMsg)
	fmt.Printf("Content-Length: %d\r\n\r\n%s", len(jsonData), jsonData) // LSP message format
}

// lspCmd represents the language server command.
var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Launch the language server for Nostos.",
	Long:  `The lsp command starts the built-in language server, enabling features like autocompletion, diagnostics, and more in supported editors.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetOutput(os.Stderr) // Ensure logs appear in *debug*
		// log.SetFlags(0)          // No timestamps
		log.Println("Nostos LSP initializing...")
		for {
			sendLogMessage("Hello from Nostos LSP!")
			log.Println("Debug log to Kakoune stderr")

			time.Sleep(5 * time.Second)
		}
	},
}

func init() {
	RootCmd.AddCommand(lspCmd)
}
