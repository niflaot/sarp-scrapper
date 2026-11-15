package presence

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

type querierStub struct {
	body     string
	err      error
	provider string
	path     string
}

func (q *querierStub) QueryProvider(_ context.Context, provider, path string, _ url.Values) ([]byte, error) {
	q.provider, q.path = provider, path
	return []byte(q.body), q.err
}

func TestSourceOnline(t *testing.T) {
	querier := &querierStub{body: `{"message":"OK","payload":[{"character_id":231,"character_name":"Adam_Belfort","character_level":24,"character_jailed":0,"account_id":465,"account_name":"Pocho"}]}`}
	characters, err := NewSource(querier).Online(context.Background())
	if err != nil || len(characters) != 1 {
		t.Fatalf("Online() = %#v, %v", characters, err)
	}
	if got := characters[0]; got.ID != 231 || got.Name != "Adam_Belfort" || got.Level != 24 || got.AccountName != "Pocho" {
		t.Fatalf("unexpected character: %#v", got)
	}
	if querier.provider != "sarp" || querier.path != "/public/global-metrics/online" {
		t.Fatalf("unexpected query: %s %s", querier.provider, querier.path)
	}
}

func TestSourceRejectsBadResponses(t *testing.T) {
	for name, querier := range map[string]*querierStub{
		"no payload":  {body: `{"message":"OK"}`},
		"null":        {body: `{"payload":null}`},
		"not json":    {body: `<html>`},
		"query error": {err: errors.New("SARP API returned HTTP 403")},
	} {
		if _, err := NewSource(querier).Online(context.Background()); err == nil {
			t.Fatalf("%s: Online() expected an error", name)
		}
	}
	if characters, err := NewSource(&querierStub{body: `{"payload":[]}`}).Online(context.Background()); err != nil || len(characters) != 0 {
		t.Fatalf("an empty list is valid: %#v, %v", characters, err)
	}
}

func TestSourceOnlineAccounts(t *testing.T) {
	querier := &querierStub{body: `{"message":"OK","httpCode":200,"source":"home","payload":{"windowHours":4,"total":217,"users":[{"accountId":1171,"accountName":"varky","adminLevel":0},{"accountId":13297,"accountName":"x","adminLevel":2}]}}`}
	accounts, err := NewSource(querier).OnlineAccounts(context.Background())
	if err != nil || len(accounts) != 2 {
		t.Fatalf("OnlineAccounts() = %#v, %v", accounts, err)
	}
	if got := accounts[1]; got.ID != 13297 || got.Name != "x" || got.AdminLevel != 2 {
		t.Fatalf("unexpected account: %#v", got)
	}
	if querier.provider != "sarp" || querier.path != "/home/online-users" {
		t.Fatalf("unexpected query: %s %s", querier.provider, querier.path)
	}
	for name, body := range map[string]string{"no payload": `{}`, "array payload": `{"payload":[]}`, "no users": `{"payload":{"total":0}}`, "null users": `{"payload":{"users":null}}`} {
		if _, err := NewSource(&querierStub{body: body}).OnlineAccounts(context.Background()); err == nil {
			t.Fatalf("%s: OnlineAccounts() expected an error", name)
		}
	}
}
