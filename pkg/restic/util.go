package restic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func convertToMinutesSeconds(time string) (int, int, error) {
	parts := strings.Split(time, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("failed to convert minutes")
	}
	minutes, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	fraction, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	seconds := int((fraction * 60) / 100)
	if seconds >= 60 {
		m := int(seconds / 60)
		minutes = minutes + m
		seconds = seconds - m*60
	}

	return minutes, seconds, nil
}

func separators(r rune) bool {
	return r == ' ' || r == '\t' || r == ','
}

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
