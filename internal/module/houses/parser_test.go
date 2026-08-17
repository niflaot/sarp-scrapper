package houses

import (
	"strings"
	"testing"
)

func TestParseAvailableHouses(t *testing.T) {
	body := []byte(`{
      "data": {
        "markers": [
          {"key": 749, "address": "Olympia Avenue", "price": "$1,338,876", "icon": "radar_propertyG.png", "lat": -12.5, "lng": 44.25},
          {"house_key": "20", "direccion": "Rodeo", "precio": 900000, "type": "casa", "for_sale": true},
          {"key": "21", "address": "Occupied", "price": 100, "type": "house", "status": "not_for_sale"},
          {"key": 749, "address": "Olympia Avenue", "price": 1338876, "available": true}
        ]
      }
    }`)

	houses, err := ParseAvailableHouses(body)
	if err != nil {
		t.Fatalf("ParseAvailableHouses() error = %v", err)
	}
	if len(houses) != 2 || houses[0].Key != "20" || houses[1].Key != "749" {
		t.Fatalf("unexpected houses: %#v", houses)
	}
	if houses[1].Price != 1338876 || houses[1].Latitude == nil || *houses[1].Latitude != -12.5 {
		t.Fatalf("unexpected parsed house: %#v", houses[1])
	}
	if !houses[0].Available || !houses[1].Available {
		t.Fatalf("houses must be marked available: %#v", houses)
	}
}

func TestParseAvailableHousesErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "invalid JSON", body: `{`, want: "decode map JSON"},
		{name: "unknown schema", body: `{"data":[{"foo":"bar"}]}`, want: "recognizable house records"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseAvailableHouses([]byte(test.body))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseAvailableHouses() error = %v", err)
			}
		})
	}
}

func TestParseAvailableHousesAllowsEmptyAvailableSet(t *testing.T) {
	houses, err := ParseAvailableHouses([]byte(`[{"key":1,"address":"Owned","price":50,"type":"house","available":false}]`))
	if err != nil || len(houses) != 0 {
		t.Fatalf("ParseAvailableHouses() = %#v, %v", houses, err)
	}
}

func TestParseAvailableHousesFromMapIconPopup(t *testing.T) {
	body := []byte("{\"message\":\"OK\",\"payload\":{\"icons\":[{\"id\":749,\"src\":\"/assets/images/map_icons/radar_propertyG.png\",\"popup\":\"<strong>Llave: 749</strong><br>Olympia Avenue, Rodeo<br>Precio: $1,338,876\",\"x\":44.25,\"y\":-12.5}]}}")

	houses, err := ParseAvailableHouses(body)
	if err != nil || len(houses) != 1 {
		t.Fatalf("ParseAvailableHouses() = %#v, %v", houses, err)
	}
	house := houses[0]
	if house.Key != "749" || house.Address != "Olympia Avenue, Rodeo" || house.Price != 1338876 || !house.Available {
		t.Fatalf("unexpected house: %#v", house)
	}
}
