package warehouses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func ParseAvailableWarehouses(body []byte) ([]Warehouse, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode map JSON: %w", err)
	}

	containers := warehouseContainers(root)
	if len(containers) == 0 {
		return nil, fmt.Errorf("map JSON does not contain a warehouses array")
	}
	byID := make(map[string]Warehouse)
	for _, container := range containers {
		for _, item := range container {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			warehouse, ok := parseWarehouse(record)
			if ok {
				byID[warehouse.ID] = warehouse
			}
		}
	}

	result := make([]Warehouse, 0, len(byID))
	for _, warehouse := range byID {
		result = append(result, warehouse)
	}
	sort.Slice(result, func(i, j int) bool { return numericLess(result[i].ID, result[j].ID) })
	return result, nil
}

func warehouseContainers(root any) [][]any {
	containers := make([][]any, 0)
	var visit func(any)
	visit = func(value any) {
		switch typed := value.(type) {
		case []any:
			for _, child := range typed {
				visit(child)
			}
		case map[string]any:
			for key, child := range typed {
				if normalized := normalize(key); normalized == "warehouses" || normalized == "almacenes" {
					if array, ok := child.([]any); ok {
						containers = append(containers, array)
						continue
					}
				}
				visit(child)
			}
		}
	}
	visit(root)
	return containers
}

func parseWarehouse(record map[string]any) (Warehouse, bool) {
	id, ok := stringValue(value(record, "id", "warehouseid", "warehouse_id", "key", "llave"))
	if !ok {
		return Warehouse{}, false
	}
	address, _ := stringValue(value(record, "address", "direccion", "location", "name"))
	if address == "" {
		address = "Almacén #" + id
	}
	price, _ := integerValue(value(record, "value", "price", "saleprice", "sale_price", "precio", "valor"))
	icon, _ := stringValue(value(record, "icon", "iconurl", "icon_url", "image", "imageurl"))
	if icon == "" {
		icon = defaultIconURL
	}
	warehouse := Warehouse{ID: id, Address: address, Value: price, Icon: icon}
	if exitX, ok := floatValue(value(record, "exitx", "exit_x", "x")); ok {
		warehouse.ExitX = &exitX
	}
	if exitY, ok := floatValue(value(record, "exity", "exit_y", "y")); ok {
		warehouse.ExitY = &exitY
	}
	return warehouse, true
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

func floatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	}
	return 0, false
}

func numericLess(left, right string) bool {
	leftNumber, leftErr := strconv.ParseInt(left, 10, 64)
	rightNumber, rightErr := strconv.ParseInt(right, 10, 64)
	if leftErr == nil && rightErr == nil {
		return leftNumber < rightNumber
	}
	return left < right
}
