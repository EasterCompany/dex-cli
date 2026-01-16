package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/cache"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// getWhisperModelPath returns the path where whisper models are stored
func getWhisperModelPath() (string, error) {
	dexterPath, err := config.GetDexterPath()
	if err != nil {
		return "", fmt.Errorf("failed to get dexter path: %w", err)
	}
	return filepath.Join(dexterPath, "models", "whisper"), nil
}

// InitWhisper installs faster-whisper and downloads the model
func InitWhisper() error {
	ui.PrintHeader("Initializing Whisper (Faster-Whisper)")

	// Get python paths
	_, pythonExecutable, pipExecutable, _, err := getPythonPaths("3.14")
	if err != nil {
		return err
	}

	// Check if python executable exists
	if _, err := os.Stat(pythonExecutable); err != nil {
		return fmt.Errorf("python environment not found. Run 'dex python --version' first to initialize")
	}

	// Install required packages
	ui.PrintInfo("Installing required Python packages...")
	packages := []string{
		"av>=11.0.0",
		"faster-whisper>=1.0.0",
		"numpy",
		"nvidia-cublas-cu12",
		"nvidia-cudnn-cu12",
	}

	// Install all packages in one command to ensure dependency resolution works correctly
	args := append([]string{"install", "-U"}, packages...)
	installCmd := exec.Command(pipExecutable, args...)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	ui.PrintInfo(fmt.Sprintf("Running: %s %s", pipExecutable, fmt.Sprint(args)))
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install python packages: %w", err)
	}
	ui.PrintSuccess("All Python packages installed successfully.")

	// Create models directory
	modelPath, err := getWhisperModelPath()
	if err != nil {
		return err
	}
	// We will store the model in .../models/whisper/large-v3-turbo
	// faster-whisper downloads specific files, so we pass this dir to it.
	finalModelDir := filepath.Join(modelPath, "large-v3-turbo")

	if err := os.MkdirAll(modelPath, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	ui.PrintInfo("Downloading Faster-Whisper large-v3-turbo model...")

	// Python script to download the model
	downloadScript := fmt.Sprintf(`
import logging
from faster_whisper import download_model

logging.basicConfig(level=logging.INFO)
print("Downloading model to %s...")
# Using a reliable conversion of large-v3-turbo
model_id = "deepdml/faster-whisper-large-v3-turbo-ct2"
download_model(model_id, output_dir="%s")
print("Download complete.")
`, finalModelDir, finalModelDir)

	cmd := exec.Command(pythonExecutable, "-c", downloadScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download whisper model: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Model downloaded successfully to %s", finalModelDir))
	ui.PrintSuccess("Whisper initialization complete!")
	ui.PrintInfo("You can now use 'dex whisper transcribe' to transcribe audio files.")
	return nil
}

// getLibraryPathAdditions queries the python environment for nvidia library paths
func getLibraryPathAdditions(pythonExecutable string) (string, error) {
	script := `
import os
import sys

paths = []
try:
    import nvidia.cudnn
    paths.append(os.path.join(list(nvidia.cudnn.__path__)[0], 'lib'))
except Exception:
    pass

try:
    import nvidia.cublas
    paths.append(os.path.join(list(nvidia.cublas.__path__)[0], 'lib'))
except Exception:
    pass

print(os.pathsep.join(paths))
`
	cmd := exec.Command(pythonExecutable, "-c", script)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// TranscribeFile transcribes an audio file using whisper
func TranscribeFile(filePath string) error {
	fmt.Fprintf(os.Stderr, "\n=== Transcribing: %s ===\n", filePath)

	// Get python paths (Use 3.10 for stability)
	_, pythonExecutable, _, _, err := getPythonPaths("3.10")
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("audio file not found: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get model path
	modelPath, err := getWhisperModelPath()
	if err != nil {
		return err
	}
	modelDir := filepath.Join(modelPath, "large-v3-turbo")

	// Check if model exists
	if _, err := os.Stat(modelDir); err != nil {
		return fmt.Errorf("whisper model not found at %s. Run 'dex whisper init' first", modelDir)
	}

	// Create transcription script using faster-whisper
	transcribeScript := fmt.Sprintf(`
import sys
import os
import json

# Inject LD_LIBRARY_PATH inside python to ensure ctranslate2 finds the libs
try:
    import nvidia.cudnn
    import nvidia.cublas
    
    cudnn_lib = os.path.join(list(nvidia.cudnn.__path__)[0], 'lib')
    cublas_lib = os.path.join(list(nvidia.cublas.__path__)[0], 'lib')
    
    current_ld = os.environ.get("LD_LIBRARY_PATH", "")
    new_ld = f"{cudnn_lib}:{cublas_lib}:{current_ld}"
    os.environ["LD_LIBRARY_PATH"] = new_ld
except Exception as e:
    # Quietly fail if nvidia libs aren't found (might be CPU only)
    pass

from faster_whisper import WhisperModel

try:
    print(f"Loading Faster-Whisper model from {r'%s'}...", file=sys.stderr)
    
    # Try to initialize with CUDA first
    device = "cuda"
    compute_type = "float16"
    
    try:
        model = WhisperModel(
            r"%s", 
            device=device, 
            compute_type=compute_type
        )
        print("Initialized Whisper on GPU (CUDA)", file=sys.stderr)
    except Exception as e:
        print(f"CUDA initialization failed ({e}), falling back to CPU...", file=sys.stderr)
        device = "cpu"
        compute_type = "int8"
        model = WhisperModel(
            r"%s", 
            device=device, 
            compute_type=compute_type
        )

    audio_path = r"%s"
    print("Transcribing audio...", file=sys.stderr)

    # 1. Transcribe (Force English)
    segments, info = model.transcribe(
        audio_path, 
        task="transcribe", 
        language="en", 
        beam_size=5,
        vad_filter=True,
        vad_parameters=dict(min_silence_duration_ms=500),
        condition_on_previous_text=False,
        compression_ratio_threshold=2.4,
        log_prob_threshold=-1.0
    )
    
    original_text_parts = []
    # Common hallucinations from Whisper on silent audio
    hallucinations = ["thank you", "thanks", "you", "bye", "thank you.", "thanks.", "you.", "bye."]
    
    for segment in segments:
        text_clean = segment.text.strip().lower()
        # Check against hallucinations list (exact match or close)
        is_hallucination = False
        if text_clean in hallucinations:
            is_hallucination = True
        # Also check for repeated "Thank you. Thank you."
        if "thank you" in text_clean and len(text_clean) < 25:
             is_hallucination = True

        if is_hallucination:
            # Only keep if extremely confident
            if segment.avg_logprob < -0.25:
                continue
        
        original_text_parts.append(segment.text)
    original_text = "".join(original_text_parts).strip()
    
    # Output structured JSON
    output = {
        "original_transcription": original_text
    }
    print(json.dumps(output))

except Exception as e:
    error_out = {"error": str(e)}
    print(json.dumps(error_out))
    import traceback
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
`, modelDir, modelDir, modelDir, absPath)

	fmt.Fprintf(os.Stderr, "Loading model and transcribing...\n")

	// Inject nvidia library paths into LD_LIBRARY_PATH
	libPaths, err := getLibraryPathAdditions(pythonExecutable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get nvidia library paths: %v\n", err)
	}

	cmd := exec.Command(pythonExecutable, "-c", transcribeScript)

	// Set environment variables
	env := os.Environ()
	if libPaths != "" {
		currentLD := os.Getenv("LD_LIBRARY_PATH")
		newLD := libPaths
		if currentLD != "" {
			newLD = libPaths + string(os.PathListSeparator) + currentLD
		}
		env = append(env, "LD_LIBRARY_PATH="+newLD)
	}
	cmd.Env = env

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Transcription complete!\n")
	return nil
}

// TranscribeRedisKey transcribes audio stored in Redis
func TranscribeRedisKey(key string) error {
	fmt.Fprintf(os.Stderr, "\n=== Transcribing from Redis Key: %s ===\n", key)

	ctx := context.Background()
	rdb, err := cache.GetLocalClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer func() { _ = rdb.Close() }()

	// Fetch audio data
	audioData, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("failed to fetch audio from Redis: %w", err)
	}

	// Create temporary file for the audio data
	tmpFile, err := os.CreateTemp("", "whisper-*.wav")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	// Write to temporary file
	if _, err := tmpFile.Write(audioData); err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Reuse TranscribeFile logic
	return TranscribeFile(tmpFile.Name())
}
