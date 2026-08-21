package warehouses

import "testing"

func TestParseAvailableWarehouses(t *testing.T) {
	body := []byte(`{"message":"OK","payload":{"houses":[],"businesses":[],"warehouses":[{"id":1345,"address":"Camino del Sol, Richman","value":"$742,203","ExitX":123.5,"ExitY":456.25}]}}`)
	warehouses, err := ParseAvailableWarehouses(body)
	if err != nil || len(warehouses) != 1 {
		t.Fatalf("ParseAvailableWarehouses() = %#v, %v", warehouses, err)
	}
	warehouse := warehouses[0]
	if warehouse.ID != "1345" || warehouse.Address != "Camino del Sol, Richman" || warehouse.Value != 742203 || warehouse.ExitX == nil || *warehouse.ExitX != 123.5 || warehouse.ExitY == nil || *warehouse.ExitY != 456.25 || warehouse.Icon != defaultIconURL {
		t.Fatalf("unexpected warehouse: %#v", warehouse)
	}
}

func TestParseAvailableWarehousesDoesNotConfuseOtherMapCollections(t *testing.T) {
	warehouses, err := ParseAvailableWarehouses([]byte(`{"payload":{"houses":[{"id":1,"value":900000}],"businesses":[{"id":2,"value":200000}],"warehouses":[]}}`))
	if err != nil || len(warehouses) != 0 {
		t.Fatalf("ParseAvailableWarehouses() = %#v, %v", warehouses, err)
	}
}

func TestParseAvailableWarehousesRequiresCollection(t *testing.T) {
	if _, err := ParseAvailableWarehouses([]byte(`{"payload":{"houses":[]}}`)); err == nil {
		t.Fatal("ParseAvailableWarehouses() expected a missing collection error")
	}
}
