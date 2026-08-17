package houses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	housePopupPattern = regexp.MustCompile(`(?is)llave\s*[:#]?\s*([0-9]+)\s*(.*?)\s*(?:precio\s*[:]?\s*)?\$\s*([0-9.,]+)`)
	htmlTagPattern    = regexp.MustCompile(`<[^>]*>`)
)

func ParseAvailableHouses(body []byte) ([]House, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode map JSON: %w", err)
	}

	byKey := make(map[string]House)
	houseShaped := 0
	recognizedContainer := false
	var visit func(any)
	visit = func(value any) {
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				visit(item)
			}
		case map[string]any:
			for key, child := range typed {
				if normalized := normalize(key); normalized == "houses" || normalized == "properties" || normalized == "casas" {
					if _, ok := child.([]any); ok {
						recognizedContainer = true
					}
				}
			}
			if house, shaped, available := parseHouse(typed); shaped {
				houseShaped++
				if available {
					house.Available = true
					if existing, found := byKey[house.Key]; found {
						house = mergeHouse(existing, house)
					}
					byKey[house.Key] = house
				}
			}
			for _, child := range typed {
				visit(child)
			}
		}
	}
	visit(root)
	if houseShaped == 0 && !recognizedContainer {
		return nil, fmt.Errorf("map JSON does not contain recognizable house records")
	}

	result := make([]House, 0, len(byKey))
	for _, house := range byKey {
		result = append(result, house)
	}
	sort.Slice(result, func(i, j int) bool { return numericLess(result[i].Key, result[j].Key) })
	return result, nil
}

func mergeHouse(existing, incoming House) House {
	if incoming.Address == "" {
		incoming.Address = existing.Address
	}
	if incoming.Price == 0 {
		incoming.Price = existing.Price
	}
	if incoming.Icon == "" {
		incoming.Icon = existing.Icon
	}
	if incoming.Latitude == nil {
		incoming.Latitude = existing.Latitude
	}
	if incoming.Longitude == nil {
		incoming.Longitude = existing.Longitude
	}
	return incoming
}

func parseHouse(record map[string]any) (House, bool, bool) {
	icon, _ := stringValue(value(record, "icon", "iconurl", "iconpath", "marker", "image", "src"))
	kind, _ := stringValue(value(record, "type", "kind", "category"))
	status, hasStatus := stringValue(value(record, "status", "state", "saleStatus"))
	available, hasAvailable := boolValue(value(record, "available", "forsale", "isforsale", "sale", "enventa"))
	houseSignal := containsAny(strings.ToLower(icon), "radar_propertyg", "house", "casa") || containsAny(strings.ToLower(kind), "house", "casa", "property", "propiedad")

	key, keyOK := stringValue(value(record, "key", "housekey", "propertykey", "llave", "id"))
	address, addressOK := stringValue(value(record, "address", "direccion", "location", "name"))
	price, priceOK := integerValue(value(record, "price", "saleprice", "value", "precio"))
	if popupHouse, ok := houseFromPopup(record); ok {
		if !keyOK {
			key, keyOK = popupHouse.Key, true
		}
		if !addressOK {
			address, addressOK = popupHouse.Address, popupHouse.Address != ""
		}
		if !priceOK {
			price, priceOK = popupHouse.Price, true
		}
	}
	if !keyOK || (!addressOK && !priceOK && !houseSignal) {
		return House{}, false, false
	}

	house := House{Key: key, Address: address, Price: price, Icon: icon}
	if latitude, ok := floatValue(value(record, "latitude", "lat", "y")); ok {
		house.Latitude = &latitude
	}
	if longitude, ok := floatValue(value(record, "longitude", "lng", "lon", "x")); ok {
		house.Longitude = &longitude
	}

	if !houseSignal && !hasAvailable && !hasStatus {
		return House{}, false, false
	}
	if containsAny(strings.ToLower(icon), "radar_propertyg") {
		return house, true, true
	}
	if hasAvailable {
		return house, true, available
	}
	if hasStatus {
		normalized := normalize(status)
		availableStatuses := map[string]bool{
			"available": true, "forsale": true, "sale": true,
			"disponible": true, "enventa": true,
		}
		return house, true, availableStatuses[normalized]
	}
	// A house-only endpoint commonly omits an availability flag. In that case,
	// the configured endpoint itself defines the set as available inventory.
	return house, true, houseSignal
}

func houseFromPopup(record map[string]any) (House, bool) {
	for _, alias := range []string{"popup", "popuphtml", "description", "content", "html", "tooltip"} {
		text, ok := stringValue(value(record, alias))
		if !ok {
			continue
		}
		plain := html.UnescapeString(htmlTagPattern.ReplaceAllString(text, " "))
		plain = strings.Join(strings.Fields(plain), " ")
		match := housePopupPattern.FindStringSubmatch(plain)
		if len(match) != 4 {
			continue
		}
		price, ok := integerValue(match[3])
		if !ok {
			continue
		}
		address := strings.TrimSpace(match[2])
		address = strings.Trim(address, "-–—|: ")
		return House{Key: match[1], Address: address, Price: price}, true
	}
	return House{}, false
}

func value(record map[string]any, aliases ...string) any {
	for key, candidate := range record {
		normalized := normalize(key)
		for _, alias := range aliases {
			if normalized == normalize(alias) {
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

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
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
	if number, ok := value.(float64); ok {
		return int64(number), true
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

func boolValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed, err == nil
	case json.Number:
		parsed, err := typed.Int64()
		return parsed != 0, err == nil
	}
	return false, false
}

func numericLess(left, right string) bool {
	l, lErr := strconv.ParseInt(left, 10, 64)
	r, rErr := strconv.ParseInt(right, 10, 64)
	if lErr == nil && rErr == nil {
		return l < r
	}
	return left < right
}
