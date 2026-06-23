package pii

import "testing"

func TestMask(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain text", "Оплата в ресторане", "Оплата в ресторане"},
		{
			"person cyrillic",
			"Перевод от ИВАН ИВАНОВИЧ И.",
			"Перевод от [PERSON]",
		},
		{
			"person latin",
			"Transfer from JOHN A. SMITH",
			"Transfer from [PERSON]",
		},
		{
			"phone russian",
			"Звонок +7 (999) 123-45-67",
			"Звонок [PHONE]",
		},
		{
			"phone short",
			"Платёж 89991234567",
			"Платёж [PHONE]",
		},
		{
			"email",
			"Контакт user@example.com тут",
			"Контакт [EMAIL] тут",
		},
		{
			"card",
			"Списание 4111 1111 1111 1111",
			"Списание [CARD]",
		},
		{
			"passport",
			"Проверка 45 10 123456",
			"Проверка [PASSPORT]",
		},
		{
			"medical",
			"Чек АПТЕКА-РУСЬ",
			"Чек [MEDICAL]-РУСЬ",
		},
		{
			"legal",
			"Услуги Нотариус Иванов",
			"Услуги [LEGAL] Иванов",
		},
		{
			"keeps amounts",
			"Покупка на сумму 47073.46 руб",
			"Покупка на сумму 47073.46 руб",
		},
		{
			"keeps dates",
			"Операция от 2024-01-31",
			"Операция от 2024-01-31",
		},
		{
			"multiple types at once",
			"Перевод от ПЕТР ПЕТРОВ П. тел +79001112233 mail a@b.com",
			"Перевод от [PERSON] тел [PHONE] mail [EMAIL]",
		},
		{
			"idempotent",
			"[PERSON]",
			"[PERSON]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Mask(tt.in)
			if got != tt.want {
				t.Errorf("Mask(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
