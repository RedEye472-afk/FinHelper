package domain

import "testing"

func TestAccount_Validate(t *testing.T) {
	p := int64(0)
	valid := Account{UserID: 1, Name: "Наличные", Type: AccountCash, Currency: "RUB"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid account rejected: %v", err)
	}

	cases := []struct {
		name string
		mut  func(Account) Account
	}{
		{"missing user_id", func(a Account) Account { a.UserID = 0; return a }},
		{"missing name", func(a Account) Account { a.Name = ""; return a }},
		{"invalid type", func(a Account) Account { a.Type = "bogus"; return a }},
		{"missing currency", func(a Account) Account { a.Currency = ""; return a }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.mut(valid).Validate(); err == nil {
				t.Fatalf("expected validation error, got nil")
			}
		})
	}
	_ = p
}

func TestCategory_Validate(t *testing.T) {
	pid := int64(2)
	valid := Category{UserID: 1, Name: "Продукты", ParentID: &pid}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid category rejected: %v", err)
	}
	if err := (Category{UserID: 1, Name: "Транспорт", ParentID: nil}).Validate(); err != nil {
		t.Fatalf("category with nil parent rejected: %v", err)
	}

	zero := int64(0)
	cases := []struct {
		name string
		mut  func(Category) Category
	}{
		{"missing user_id", func(c Category) Category { c.UserID = 0; return c }},
		{"missing name", func(c Category) Category { c.Name = ""; return c }},
		{"zero parent_id", func(c Category) Category { c.ParentID = &zero; return c }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.mut(valid).Validate(); err == nil {
				t.Fatalf("expected validation error, got nil")
			}
		})
	}
}

func TestMoney_NewHelpers(t *testing.T) {
	// FromDecimal rounds to scale 2 (extra precision is dropped, rounded).
	got := FromDecimal(MustParseMoney("1.009").v)
	want := MustParseMoney("1.01")
	if !got.Equal(want) {
		t.Errorf("FromDecimal rounding = %s, want %s", got, want)
	}
	// Already at scale: no change.
	got = FromDecimal(MustParseMoney("47.50").v)
	if !got.Equal(MustParseMoney("47.50")) {
		t.Errorf("FromDecimal scale-2 = %s, want 47.50", got)
	}

	// Abs / Neg.
	neg := MustParseMoney("-47.50")
	if !neg.Abs().Equal(MustParseMoney("47.50")) {
		t.Errorf("Abs(-47.50) = %s, want 47.50", neg.Abs())
	}
	if !MustParseMoney("47.50").Neg().Equal(MustParseMoney("-47.50")) {
		t.Errorf("Neg(47.50) wrong")
	}

	// AddAll.
	got = AddAll([]Money{MustParseMoney("0.10"), MustParseMoney("0.20")})
	if !got.Equal(MustParseMoney("0.30")) {
		t.Errorf("AddAll = %s, want 0.30 (decimal precision, not float)", got)
	}
	if !AddAll(nil).Equal(Zero) {
		t.Errorf("AddAll(nil) should be Zero")
	}
}
