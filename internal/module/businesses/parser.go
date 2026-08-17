package businesses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func ParseAvailableBusinesses(body []byte) ([]Business, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode businesses JSON: %w", err)
	}
	containers := make([][]any, 0)
	if array, ok := root.([]any); ok {
		containers = append(containers, array)
	}
	var visit func(any)
	visit = func(value any) {
		record, ok := value.(map[string]any)
		if !ok {
			return
		}
		for key, child := range record {
			normalized := normalize(key)
			if normalized == "businesses" || normalized == "negocios" {
				if array, ok := child.([]any); ok {
					containers = append(containers, array)
				}
			}
			visit(child)
		}
	}
	visit(root)
	if len(containers) == 0 {
		return nil, fmt.Errorf("businesses JSON does not contain a businesses array")
	}
	byID := make(map[string]Business)
	for _, container := range containers {
		for _, item := range container {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			business, ok := parseBusiness(record)
			if ok {
				byID[business.ID] = business
			}
		}
	}
	result := make([]Business, 0, len(byID))
	for _, business := range byID {
		result = append(result, business)
	}
	sort.Slice(result, func(i, j int) bool { return numericLess(result[i].ID, result[j].ID) })
	return result, nil
}

func parseBusiness(record map[string]any) (Business, bool) {
	id, ok := stringValue(value(record, "id", "businessid", "business_id", "key", "negocioid"))
	if !ok {
		return Business{}, false
	}
	name, _ := stringValue(value(record, "name", "businessname", "business_name", "title", "label", "nombre"))
	if name == "" {
		name = "Negocio #" + id
	}
	price, _ := integerValue(value(record, "price", "value", "saleprice", "sale_price", "purchaseprice", "purchase_price", "precio", "valor"))
	icon, _ := stringValue(value(record, "icon", "iconurl", "icon_url", "image", "imageurl", "image_url", "logo"))
	return Business{ID: id, Name: name, Value: price, Icon: icon}, true
}

func value(record map[string]any, aliases ...string) any {
	for key, candidate := range record {
		for _, alias := range aliases {
			if normalize(key) == normalize(alias) {
				return candidate
			}
		}
	}
	return nil
}

func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}

func stringValue(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		return trimmed, trimmed != ""
	case json.Number:
		return typed.String(), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	}
	return "", false
}

func integerValue(value any) (int64, bool) {
	if number, ok := value.(json.Number); ok {
		parsed, err := number.Int64()
		return parsed, err == nil
	}
	text, ok := stringValue(value)
	if !ok {
		return 0, false
	}
	clean := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '-' {
			return r
		}
		return -1
	}, text)
	parsed, err := strconv.ParseInt(clean, 10, 64)
	return parsed, err == nil
}

func numericLess(left, right string) bool {
	leftNumber, leftErr := strconv.ParseInt(left, 10, 64)
	rightNumber, rightErr := strconv.ParseInt(right, 10, 64)
	if leftErr == nil && rightErr == nil {
		return leftNumber < rightNumber
	}
	return left < right
}
