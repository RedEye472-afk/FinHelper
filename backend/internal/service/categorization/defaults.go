// Package categorization implements BUSINESS_LOGIC.md ф.2 — rules-based
// auto-categorization of operations.
//
// Scope-lock (Plane from GLM.md): NO ML, NO bank integration. Two mechanisms:
//
//  1. Keyword rules  (Level 2, ~70%) — substring match against the PII-masked
//     counterparty + description. System defaults are seeded at registration;
//     the user adds/edits their own.
//  2. Counterparty overrides with a confirmation counter (Level 3 simplified).
//     Each manual correction increments a counter; at domain.LearnThreshold
//     (3) the override becomes authoritative. A pure counter stands in for the
//     ML model — deterministic, auditable, no training data.
//
// MCC codes (Level 1, ~90%) are intentionally absent: MVP has no bank
// integration, so there are no MCC values to match on.
//
// Match precedence (first hit wins):
//  1. learned override  (confirmations >= LearnThreshold)  — authoritative
//  2. user keyword rule (priority >= 100)
//  3. system keyword rule
//  4. tentative override (confirmations < LearnThreshold) — suggested, ask UI
//  5. no match — return zero, UI must prompt the user
//
// The categorizer is a pure decision over inputs; it never talks to the DB
// directly. Storage is abstracted behind Repo so unit tests run without one.
package categorization

import "github.com/RedEye472-afk/FinHelper/internal/domain"

// Confidence constants. These are probabilities, not monetary amounts, so a
// decimal literal is overkill; we keep them as strings to match the storage
// shape (NUMERIC(4,3)) and avoid float64 at the API boundary.
const (
	// ConfidenceKeyword — 0.75. Above the 0.70 confirm floor, so a keyword
	// match is applied directly with a "verify if wrong" affordance rather
	// than a blocking prompt.
	ConfidenceKeyword = "0.75"
	// ConfidenceOverrideLearned — 0.95. An override with confirmations >=
	// LearnThreshold is authoritative; high enough to apply without prompting.
	ConfidenceOverrideLearned = "0.95"
	// ConfidenceOverrideTentative — 0.50. Below the 0.70 floor, so the UI
	// asks the user to confirm a tentative (under-trained) override.
	ConfidenceOverrideTentative = "0.50"
)

// SystemCategories is the default category set seeded for every new user at
// registration. Order matters only for display; the seed is idempotent on
// (user_id, name). Names are reused by SystemKeywordRules below.
var SystemCategories = []string{
	"Продукты",
	"Кафе и рестораны",
	"Транспорт",
	"Такси",
	"Связь и интернет",
	"Жильё",
	"Коммунальные услуги",
	"Здоровье",
	"Аптека",
	"Развлечения",
	"Одежда и вещи",
	"Подарки",
	"Подписки",
	"Спорт",
	"Образование",
	"Налоги и сборы",
	"Прочее",
}

// SystemKeywordRules maps keywords to a SystemCategories name. Keywords are
// lowercased; matching is substring against the PII-masked counterparty +
// description. Curated for the Russian retail/payment landscape — these are
// high-signal merchant tokens, not generic words (avoid "кофе" which matches
// too much; use merchant names instead).
var SystemKeywordRules = []KeywordDefault{
	// Продукты
	{Keyword: "магнит", Category: "Продукты"},
	{Keyword: "пятерочк", Category: "Продукты"},
	{Keyword: "перекрест", Category: "Продукты"},
	{Keyword: "ашан", Category: "Продукты"},
	{Keyword: "лента", Category: "Продукты"},
	{Keyword: "вкусвилл", Category: "Продукты"},
	{Keyword: "samokat", Category: "Продукты"},
	{Keyword: "самокат", Category: "Продукты"},
	{Keyword: "сельпо", Category: "Продукты"},
	{Keyword: "дикси", Category: "Продукты"},

	// Кафе и рестораны
	{Keyword: "kfc", Category: "Кафе и рестораны"},
	{Keyword: "burger king", Category: "Кафе и рестораны"},
	{Keyword: "вкусно и точка", Category: "Кафе и рестораны"},
	{Keyword: "delivery club", Category: "Кафе и рестораны"},
	{Keyword: "яндекс еда", Category: "Кафе и рестораны"},
	{Keyword: "яндекс.еда", Category: "Кафе и рестораны"},
	{Keyword: "додо пицца", Category: "Кафе и рестораны"},
	{Keyword: "dominos", Category: "Кафе и рестораны"},
	{Keyword: "папа джонс", Category: "Кафе и рестораны"},
	{Keyword: "ресторан", Category: "Кафе и рестораны"},
	{Keyword: "кафе", Category: "Кафе и рестораны"},

	// Транспорт
	{Keyword: "мосгортранс", Category: "Транспорт"},
	{Keyword: "метрополитен", Category: "Транспорт"},
	{Keyword: "тройка", Category: "Транспорт"},
	{Keyword: "ржд", Category: "Транспорт"},
	{Keyword: "аэроэкспресс", Category: "Транспорт"},
	{Keyword: "авиакомпан", Category: "Транспорт"},

	// Такси
	{Keyword: "яндекс го", Category: "Такси"},
	{Keyword: "яндекс.го", Category: "Такси"},
	{Keyword: "yandex go", Category: "Такси"},
	{Keyword: "такси", Category: "Такси"},
	{Keyword: "ситимобил", Category: "Такси"},
	{Keyword: "vezy", Category: "Такси"},
	{Keyword: "везёт", Category: "Такси"},

	// Связь и интернет
	{Keyword: "мтс", Category: "Связь и интернет"},
	{Keyword: "билайн", Category: "Связь и интернет"},
	{Keyword: "мегафон", Category: "Связь и интернет"},
	{Keyword: "tele2", Category: "Связь и интернет"},
	{Keyword: "теле2", Category: "Связь и интернет"},
	{Keyword: "ростелеком", Category: "Связь и интернет"},
	{Keyword: "дом.ru", Category: "Связь и интернет"},

	// Жильё
	{Keyword: "аренд", Category: "Жильё"},
	{Keyword: "квартплат", Category: "Жильё"},

	// Коммунальные услуги
	{Keyword: "жкх", Category: "Коммунальные услуги"},
	{Keyword: "энергосбыт", Category: "Коммунальные услуги"},
	{Keyword: "электроэн", Category: "Коммунальные услуги"},
	{Keyword: "водоканал", Category: "Коммунальные услуги"},
	{Keyword: "газпром газораспредел", Category: "Коммунальные услуги"},
	{Keyword: "муниципал", Category: "Коммунальные услуги"},

	// Здоровье
	{Keyword: "клиник", Category: "Здоровье"},
	{Keyword: "больниц", Category: "Здоровье"},
	{Keyword: "стоматолог", Category: "Здоровье"},
	{Keyword: "медцентр", Category: "Здоровье"},
	{Keyword: "инвитро", Category: "Здоровье"},
	{Keyword: "gemotest", Category: "Здоровье"},
	{Keyword: "гемотест", Category: "Здоровье"},

	// Аптека
	{Keyword: "аптек", Category: "Аптека"},
	{Keyword: "ригла", Category: "Аптека"},
	{Keyword: "eapteka", Category: "Аптека"},
	{Keyword: "36.6", Category: "Аптека"},

	// Развлечения
	{Keyword: "каро", Category: "Развлечения"},
	{Keyword: "кинотеатр", Category: "Развлечения"},
	{Keyword: "steam", Category: "Развлечения"},
	{Keyword: "стим", Category: "Развлечения"},

	// Одежда и вещи
	{Keyword: "wildberries", Category: "Одежда и вещи"},
	{Keyword: "вайлдберриз", Category: "Одежда и вещи"},
	{Keyword: "ozon", Category: "Одежда и вещи"},
	{Keyword: "озон", Category: "Одежда и вещи"},
	{Keyword: "lamoda", Category: "Одежда и вещи"},

	// Подарки
	{Keyword: "цветов", Category: "Подарки"},
	{Keyword: "подарок", Category: "Подарки"},

	// Подписки
	{Keyword: "netflix", Category: "Подписки"},
	{Keyword: "нетфликс", Category: "Подписки"},
	{Keyword: "spotify", Category: "Подписки"},
	{Keyword: "яндекс плюс", Category: "Подписки"},
	{Keyword: "яндекс.плюс", Category: "Подписки"},
	{Keyword: "yandex plus", Category: "Подписки"},
	{Keyword: "icloud", Category: "Подписки"},
	{Keyword: "google one", Category: "Подписки"},
	{Keyword: "youtube premium", Category: "Подписки"},
	{Keyword: "telegram", Category: "Подписки"},

	// Спорт
	{Keyword: "world class", Category: "Спорт"},
	{Keyword: "фитнес", Category: "Спорт"},
	{Keyword: "спортклуб", Category: "Спорт"},

	// Образование
	{Keyword: "skillbox", Category: "Образование"},
	{Keyword: "яндекс практикум", Category: "Образование"},
	{Keyword: "нетолог", Category: "Образование"},
	{Keyword: "geekbrains", Category: "Образование"},
	{Keyword: "coursera", Category: "Образование"},
	{Keyword: "udemy", Category: "Образование"},

	// Налоги и сборы
	{Keyword: "фнс", Category: "Налоги и сборы"},
	{Keyword: "налог", Category: "Налоги и сборы"},
	{Keyword: "госуслуг", Category: "Налоги и сборы"},
}

// KeywordDefault is a (substring → category name) seed entry. The name is
// resolved to a category id at seeding time against the user's system
// categories; if the name is absent the rule is skipped (seed never fails).
type KeywordDefault struct {
	Keyword  string
	Category string
}

// ConfidenceForTier maps a match tier (see package doc) to its confidence
// string. Centralized so the service and tests agree on the values.
func ConfidenceForTier(tier MatchTier) string {
	switch tier {
	case TierOverrideLearned:
		return ConfidenceOverrideLearned
	case TierOverrideTentative:
		return ConfidenceOverrideTentative
	default: // TierKeyword
		return ConfidenceKeyword
	}
}

// MatchTier labels which precedence layer produced a categorization. The UI
// uses it to decide between "apply" and "ask to confirm".
type MatchTier int

const (
	// TierNone — no match; the UI must prompt the user.
	TierNone MatchTier = iota
	// TierKeyword — keyword rule (system or user).
	TierKeyword
	// TierOverrideTentative — override below the LearnThreshold; suggest, ask.
	TierOverrideTentative
	// TierOverrideLearned — override at/above the LearnThreshold; authoritative.
	TierOverrideLearned
)

// _ keeps the domain import meaningful: LearnThreshold lives there so both the
// service and any future ML adapter share one source of truth.
var _ = domain.LearnThreshold
