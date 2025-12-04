package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

// InitWhisper installs whisper dependencies and clones the large-v3-turbo model from Hugging Face
func InitWhisper() error {
	ui.PrintHeader("Initializing Whisper")

	// Get python paths
	_, pythonExecutable, pipExecutable, _, err := getPythonPaths()
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
		"transformers",
		"torch",
		"torchaudio",
		"accelerate",
		"huggingface-hub",
	}

	for _, pkg := range packages {
		ui.PrintInfo(fmt.Sprintf("Installing %s...", pkg))
		cmd := exec.Command(pipExecutable, "install", "-U", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg, err)
		}
	}
	ui.PrintSuccess("All Python packages installed successfully.")

	// Check if git-lfs is installed
	ui.PrintInfo("Checking for git-lfs...")
	if _, err := exec.LookPath("git-lfs"); err != nil {
		ui.PrintWarning("git-lfs not found. Installing...")
		cmd := exec.Command("yay", "-S", "--noconfirm", "git-lfs")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install git-lfs: %w", err)
		}
		ui.PrintSuccess("git-lfs installed successfully.")
	} else {
		ui.PrintSuccess("git-lfs is already installed.")
	}

	// Initialize git-lfs
	cmd := exec.Command("git", "lfs", "install")
	if err := cmd.Run(); err != nil {
		ui.PrintWarning("git-lfs install returned error (may already be initialized)")
	}

	// Create models directory
	modelPath, err := getWhisperModelPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(modelPath, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	// Check if model already exists
	modelDir := filepath.Join(modelPath, "large-v3-turbo")
	if _, err := os.Stat(modelDir); err == nil {
		ui.PrintWarning("Model directory already exists. Skipping clone.")
		ui.PrintInfo(fmt.Sprintf("Model location: %s", modelDir))
		ui.PrintSuccess("Whisper initialization complete!")
		return nil
	}

	ui.PrintInfo("Cloning Whisper large-v3-turbo model from Hugging Face...")
	ui.PrintInfo("This may take several minutes depending on your internet connection...")

	// Clone the model from Hugging Face
	cmd = exec.Command(
		"git", "clone",
		"https://huggingface.co/openai/whisper-large-v3-turbo",
		modelDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = modelPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone whisper model: %w", err)
	}
	ui.PrintSuccess(fmt.Sprintf("Model cloned successfully to %s", modelDir))

	ui.PrintSuccess("Whisper initialization complete!")
	ui.PrintInfo("You can now use 'dex whisper transcribe' to transcribe audio files.")
	return nil
}

// TranscribeFile transcribes an audio file using whisper
func TranscribeFile(filePath string) error {
	fmt.Fprintf(os.Stderr, "\n=== Transcribing: %s ===\n", filePath)

	// Get python paths
	_, pythonExecutable, _, _, err := getPythonPaths()
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

	// Create transcription script using transformers
	transcribeScript := fmt.Sprintf(`
import sys
import json
import torch
import torchaudio
import torchaudio.transforms as T
from transformers import AutoModelForSpeechSeq2Seq, AutoProcessor, pipeline

try:
    # Set device
    device = "cuda:0" if torch.cuda.is_available() else "cpu"
    torch_dtype = torch.float16 if torch.cuda.is_available() else torch.float32

    print("Loading Whisper model from local path...", file=sys.stderr)
    model_path = "%s"
    audio_path = "%s"

    # Load model and processor from local directory
    model = AutoModelForSpeechSeq2Seq.from_pretrained(
        model_path,
        torch_dtype=torch_dtype,
        low_cpu_mem_usage=True,
        use_safetensors=True,
        local_files_only=True
    )
    model.to(device)

    processor = AutoProcessor.from_pretrained(model_path, local_files_only=True)

    # Detect Language
    print("Detecting language...", file=sys.stderr)
    waveform, sample_rate = torchaudio.load(audio_path)
    if sample_rate != 16000:
        resampler = T.Resample(sample_rate, 16000)
        waveform = resampler(waveform)
    
    # Mix to mono if needed
    if waveform.shape[0] > 1:
        waveform = torch.mean(waveform, dim=0, keepdim=True)

    # Get features for first 30s
    input_features = processor(waveform.squeeze().numpy(), sampling_rate=16000, return_tensors="pt").input_features
    input_features = input_features.to(device).to(torch_dtype)

    # Generate to detect language (force task to transcribe so it detects language)
    # We rely on the model's first token output for language
    model_output = model.generate(input_features, max_new_tokens=5)
    
    # The second token usually contains the language code (index 1)
    # Token 0: <|startoftranscript|>
    # Token 1: <|lang|>
    if len(model_output[0]) > 1:
        lang_id = model_output[0][1].item()
        detected_lang = processor.tokenizer.decode([lang_id]).replace("<|", "").replace("|>", "")
    else:
        detected_lang = "en" # Fallback

    print(f"Detected language: {detected_lang}", file=sys.stderr)

    # Create pipeline
    pipe = pipeline(
        "automatic-speech-recognition",
        model=model,
        tokenizer=processor.tokenizer,
        feature_extractor=processor.feature_extractor,
        max_new_tokens=128,
        torch_dtype=torch_dtype,
        device=device,
    )

    print("Transcribing audio...", file=sys.stderr)
    result = pipe(audio_path, generate_kwargs={"task": "transcribe"})
    original_text = result["text"]

    english_translation = ""
    if detected_lang != "en":
        print("Translating to English...", file=sys.stderr)
        trans_result = pipe(audio_path, generate_kwargs={"task": "translate", "language": "en"})
        english_translation = trans_result["text"]

    # Output structured JSON
    output = {
        "original_transcription": original_text.strip(),
        "detected_language": detected_lang,
        "english_translation": english_translation.strip()
    }
    print(json.dumps(output))

except Exception as e:
    error_out = {"error": str(e)}
    print(json.dumps(error_out))
    import traceback
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
`, modelDir, absPath)

	fmt.Fprintf(os.Stderr, "Loading model and transcribing...\n")
	cmd := exec.Command(pythonExecutable, "-c", transcribeScript)
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
