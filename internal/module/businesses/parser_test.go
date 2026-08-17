package businesses

import "testing"

func TestParseAvailableBusinesses(t *testing.T) {
	body := []byte(`{"message":"OK","payload":{"businesses":[{"id":12,"name":"24/7 Market","salePrice":"$250,000","iconUrl":"https://example.com/shop.png"}],"total":1}}`)
	businesses, err := ParseAvailableBusinesses(body)
	if err != nil || len(businesses) != 1 {
		t.Fatalf("ParseAvailableBusinesses() = %#v, %v", businesses, err)
	}
	if businesses[0].ID != "12" || businesses[0].Name != "24/7 Market" || businesses[0].Value != 250000 {
		t.Fatalf("unexpected business: %#v", businesses[0])
	}
}

func TestParseAvailableBusinessesEmpty(t *testing.T) {
	businesses, err := ParseAvailableBusinesses([]byte(`{"businesses":[],"total":0}`))
	if err != nil || len(businesses) != 0 {
		t.Fatalf("ParseAvailableBusinesses() = %#v, %v", businesses, err)
	}
}
