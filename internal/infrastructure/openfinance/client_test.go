package openfinance

import (
	"encoding/json"
	"testing"
)

func TestGetAmount(t *testing.T) {
	cases := []struct {
		raw  string
		want float64
	}{
		{`-100`, -100},
		{`34.99`, 34.99},
		{`"25.50"`, 25.50},
		{`null`, 0},
		{`""`, 0},
		{`0`, 0},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			tx := &Transaction{AmountRaw: json.RawMessage(c.raw)}
			got, err := tx.GetAmount()
			if err != nil || got != c.want {
				t.Errorf("GetAmount(%s) = %v, %v; want %v, nil", c.raw, got, err, c.want)
			}
		})
	}
}

func TestGetAmount_NilReceiver(t *testing.T) {
	var tx *Transaction
	got, err := tx.GetAmount()
	if err != nil || got != 0 {
		t.Errorf("GetAmount(nil) = %v, %v; want 0, nil", got, err)
	}
}

func TestParseRawInt(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{`10`, 10},
		{`"3"`, 3},
		{`null`, 0},
		{`""`, 0},
		{`0`, 0},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			got, err := parseRawInt(json.RawMessage(c.raw), "field")
			if err != nil || got != c.want {
				t.Errorf("parseRawInt(%s) = %v, %v; want %v, nil", c.raw, got, err, c.want)
			}
		})
	}
}

func TestGetInstallmentNumber_NilReceiver(t *testing.T) {
	var cc *TransactionCreditData
	got, err := cc.GetInstallmentNumber()
	if err != nil || got != 0 {
		t.Errorf("GetInstallmentNumber(nil) = %v, %v; want 0, nil", got, err)
	}
}
