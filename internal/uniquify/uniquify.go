package uniquify

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Params struct {
	Brightness	float64
	Contrast	float64
	Saturation	float64
	Gamma		float64
	HueShift	float64
	NoiseLevel	int
	Speed		float64
}

func Randomize() Params {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return Params{
		Brightness:	randRange(r, -0.04, 0.04),
		Contrast:	randRange(r, 0.95, 1.05),
		Saturation:	randRange(r, 0.92, 1.08),
		Gamma:		randRange(r, 0.94, 1.06),
		HueShift:	randRange(r, -3.0, 3.0),
		NoiseLevel:	2 + r.Intn(5),
		Speed:		randRange(r, 0.99, 1.01),
	}
}

func randRange(r *rand.Rand, min, max float64) float64 {
	return min + r.Float64()*(max-min)
}

func (p Params) String() string {
	return fmt.Sprintf("brightness=%.3f contrast=%.2f saturation=%.2f gamma=%.2f hue=%.1f° noise=%d speed=%.3f",
		p.Brightness, p.Contrast, p.Saturation, p.Gamma, p.HueShift, p.NoiseLevel, p.Speed)
}

func FindFFmpeg() (string, error) {

	candidates := []string{
		"ffmpeg",
		"ffmpeg.exe",
		`C:\ffmpeg\bin\ffmpeg.exe`,
		`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
	}
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("ffmpeg not found in PATH. Install it: winget install Gyan.FFmpeg")
}

func Process(inputPath string) (outputPath string, params Params, err error) {
	ffmpeg, err := FindFFmpeg()
	if err != nil {
		return "", Params{}, err
	}

	params = Randomize()

	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outputPath = filepath.Join(dir, fmt.Sprintf("%s_unique_%d%s", base, time.Now().UnixMilli(), ext))

	videoFilter := fmt.Sprintf(
		"scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2:black,eq=brightness=%.4f:contrast=%.3f:saturation=%.3f:gamma=%.3f,hue=h=%.2f,noise=alls=%d:allf=t",
		params.Brightness,
		params.Contrast,
		params.Saturation,
		params.Gamma,
		params.HueShift,
		params.NoiseLevel,
	)

	args := []string{
		"-i", inputPath,
		"-vf", videoFilter,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "18",
		"-c:a", "copy",
		"-y",
		outputPath,
	}

	cmd := exec.Command(ffmpeg, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return "", params, fmt.Errorf("ffmpeg failed: %w", err)
	}

	return outputPath, params, nil
}

func ProcessInPlace(inputPath string) (Params, error) {
	outputPath, params, err := Process(inputPath)
	if err != nil {
		return params, err
	}

	if err := os.Remove(inputPath); err != nil {
		os.Remove(outputPath)
		return params, fmt.Errorf("remove original: %w", err)
	}
	if err := os.Rename(outputPath, inputPath); err != nil {
		return params, fmt.Errorf("rename output: %w", err)
	}
	return params, nil
}

func ProcessForAccount(inputPath string, accountID int64) (string, Params, error) {
	ffmpeg, err := FindFFmpeg()
	if err != nil {
		return "", Params{}, err
	}

	params := Randomize()

	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outputPath := filepath.Join(dir, fmt.Sprintf("%s_acc%d_%d%s", base, accountID, time.Now().UnixMilli(), ext))

	videoFilter := fmt.Sprintf(
		"scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2:black,eq=brightness=%.4f:contrast=%.3f:saturation=%.3f:gamma=%.3f,hue=h=%.2f,noise=alls=%d:allf=t",
		params.Brightness,
		params.Contrast,
		params.Saturation,
		params.Gamma,
		params.HueShift,
		params.NoiseLevel,
	)

	args := []string{
		"-i", inputPath,
		"-vf", videoFilter,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "18",
		"-c:a", "copy",
		"-y",
		outputPath,
	}

	cmd := exec.Command(ffmpeg, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return "", params, fmt.Errorf("ffmpeg failed: %w", err)
	}

	return outputPath, params, nil
}
