package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/utils"
)

// Whisper handles whisper-related commands for speech-to-text transcription
func Whisper(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("whisper command requires a subcommand (init, transcribe)")
	}

	subcommand := args[0]

	switch subcommand {
	case "init":
		return utils.InitWhisper()

	case "transcribe":
		if len(args) < 2 {
			return fmt.Errorf("transcribe requires a flag: -f <file_path> or -b <base64_audio_data>")
		}

		flag := args[1]

		switch flag {
		case "-f", "--file":
			if len(args) < 3 {
				return fmt.Errorf("-f flag requires a file path argument")
			}
			filePath := args[2]
			return utils.TranscribeFile(filePath)

		case "-b", "--bytes":
			if len(args) < 3 {
				return fmt.Errorf("-b flag requires base64 encoded audio data")
			}
			encodedData := args[2]
			return utils.TranscribeBytes(encodedData)

		default:
			return fmt.Errorf("unknown flag: %s. Use -f <file_path> or -b <base64_data>", flag)
		}

	default:
		fmt.Println("Available commands:")
		fmt.Println("  dex whisper init                    # Install whisper and download models")
		fmt.Println("  dex whisper transcribe -f <path>    # Transcribe an audio file")
		fmt.Println("  dex whisper transcribe -b <data>    # Transcribe base64 encoded audio data")
		return fmt.Errorf("unknown whisper subcommand: %s", subcommand)
	}
}
