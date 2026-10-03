package balance

import "testing"

func TestPresetsValid(t *testing.T) {
	for _, p := range Presets() {
		if errs := p.Validate(); len(errs) > 0 {
			t.Errorf("%s: %v", p.Name, errs)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	d := Default()
	c, err := Unmarshal(d.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if c.S("oreshnik") == nil || c.W("kab") == nil || c.U("sniper") == nil {
		t.Fatal("lookup failed after round trip")
	}
}
