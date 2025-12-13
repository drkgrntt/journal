package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

func FormatMoney(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

func CentsToDollars(cents int) float64 {
	return float64(cents) / 10
}

func DollarsToCents(dollars float64) int {
	cents := dollars * 10
	return int(cents)
}

func CastToType[T any](data any) (error, T) {
	var value T

	rawJSON, err := json.Marshal(data)
	if err != nil {
		return err, value
	}

	err = json.Unmarshal(rawJSON, &value)

	return err, value
}

type KeyValue struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func SubstringToFullWords(input string, characters int) string {
	pieces := strings.Split(input, " ")
	result := ""
	for i, piece := range pieces {
		result += piece + " "
		if len(result) >= characters {
			if i < len(pieces)-1 {
				result += "..."
			}
			break
		}
	}
	return result
}
