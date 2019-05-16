package restic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func convertSizeToBytes(dataSize string) (float64, error) {
	parts := strings.Split(dataSize, " ")
	if len(parts) != 2 {
		return 0, errors.New("invalid data size format")
	}

	switch parts[1] {
	case "B":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size, nil
	case "KiB", "KB":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size * 1024, nil
	case "MiB", "MB":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size * 1024 * 1024, nil
	case "GiB", "GB":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size * 1024 * 1024 * 1024, nil
	}
	return 0, errors.New("unknown unit for data size")
}

func convertTimeToSeconds(processingTime string) (int, error) {
	var minutes, seconds int
	_, err := fmt.Sscanf(processingTime, "%dm%ds", &minutes, &seconds)
	if err != nil {
		return 0, err
	}

	return minutes*60 + seconds, nil
}

func formatBytes(c uint64) string {
	b := float64(c)

	switch {
	case c > 1<<40:
		return fmt.Sprintf("%.3f TiB", b/(1<<40))
	case c > 1<<30:
		return fmt.Sprintf("%.3f GiB", b/(1<<30))
	case c > 1<<20:
		return fmt.Sprintf("%.3f MiB", b/(1<<20))
	case c > 1<<10:
		return fmt.Sprintf("%.3f KiB", b/(1<<10))
	default:
		return fmt.Sprintf("%d B", c)
	}
}

func formatSeconds(sec uint64) string {
	hours := sec / 3600
	sec -= hours * 3600
	min := sec / 60
	sec -= min * 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, min, sec)
	}

	return fmt.Sprintf("%d:%02d", min, sec)
}
