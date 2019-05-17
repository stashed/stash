package restic

import (
	"fmt"
	"strings"
)

func convertSizeToBytes(dataSize string) (float64, error) {
	var size float64

	switch {
	case strings.HasSuffix(dataSize, "TiB"):
		_, err := fmt.Sscanf(dataSize, "%f TiB", &size)
		if err != nil {
			return 0, nil
		}
		return size * (1 << 40), nil
	case strings.HasSuffix(dataSize, "GiB"):
		_, err := fmt.Sscanf(dataSize, "%f GiB", &size)
		if err != nil {
			return 0, nil
		}
		return size * (1 << 30), nil
	case strings.HasSuffix(dataSize, "MiB"):
		_, err := fmt.Sscanf(dataSize, "%f MiB", &size)
		if err != nil {
			return 0, nil
		}
		return size * (1 << 20), nil
	case strings.HasSuffix(dataSize, "KiB"):
		_, err := fmt.Sscanf(dataSize, "%f KiB", &size)
		if err != nil {
			return 0, nil
		}
		return size * (1 << 10), nil
	default:
		_, err := fmt.Sscanf(dataSize, "%f B", &size)
		if err != nil {
			return 0, nil
		}
		return size, nil

	}
}

func convertTimeToSeconds(processingTime string) (uint64, error) {
	var h, m, s uint64
	parts := strings.Split(processingTime, ":")
	if len(parts) == 3 {
		_, err := fmt.Sscanf(processingTime, "%d:%d:%d", &h, &m, &s)
		if err != nil {
			return 0, err
		}
	} else if len(parts) == 2 {
		_, err := fmt.Sscanf(processingTime, "%d:%d", &m, &s)
		if err != nil {
			return 0, err
		}
	} else {
		_, err := fmt.Sscanf(processingTime, "%d", &s)
		if err != nil {
			return 0, err
		}
	}

	return h*3600 + m*60 + s, nil
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
