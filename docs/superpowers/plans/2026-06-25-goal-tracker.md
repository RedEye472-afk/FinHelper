# Goal Tracker (Фича 5) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Реализовать трекер финансовых целей — CRUD целей, журнал внеплановых пополнений (идемпотентный), проекция статуса и what-if симуляция — на формулах фонда возмещения (Копнова Гл. 3.3.3).

**Architecture:** Полная симметрия с эталонным пакетом `budget`. Новый mathcore-подпакет `mathcore/goals/` (strictly decimal, без float64-bridges). Слои: storage (sqlmock-тесты) → service (Repo interface + fakeRepo) → transport/http (httptest). Гибридная модель `effective_current = goals.current_amount + Σ contributions`. Минимальные правки в `dashboard.go`, `router.go`, `main.go`.

**Tech Stack:** Go 1.26, chi v5, shopspring/decimal, pgx/v5 + database/sql, sqlmock, `C:\Program Files\Go\bin\go.exe` (Windows, go не в PATH).

**Спецификация:** `docs/superpowers/specs/2026-06-25-goal-tracker-design.md`

**Команды проверки (Windows):**
```bash
cd C:/Users/user/ZCodeProject/FinHelper/backend
"C:/Program Files/Go/bin/go.exe" build ./...
"C:/Program Files/Go/bin/go.exe" vet ./...
"C:/Program Files/Go/bin/go.exe" test ./...
```

---

## File Structure

| Файл | Ответственность | Создать/Modify |
|---|---|---|
| `internal/mathcore/goals/doc.go` | package doc + sentinel errors | Create |
| `internal/mathcore/goals/sinkingfund.go` | SolveTerm, SolveContribution, SolveFutureValue, InflateTarget | Create |
| `internal/mathcore/goals/sinkingfund_test.go` | golden + edge-cases | Create |
| `internal/domain/goal.go` | Goal, GoalContribution, GoalStatus, Validate | Create |
| `migrations/0003_goals_contributions.sql` | таблица goal_contributions | Create |
| `internal/storage/goals.go` | CRUD goals + contributions + Sum | Create |
| `internal/storage/goals_test.go` | sqlmock-тесты | Create |
| `internal/storage/dashboard.go` | +Σ contributions в GoalProgresses | Modify |
| `internal/service/goals/goals.go` | Service + Repo interface + Projection/Simulate | Create |
| `internal/service/goals/goals_test.go` | fakeRepo + сценарии | Create |
| `internal/transport/http/goals.go` | handler + Register | Create |
| `internal/transport/http/goals_test.go` | httptest + fakeRepo | Create |
| `internal/transport/http/router.go` | Deps.Goals + монтаж | Modify |
| `cmd/server/main.go` | goals.NewService(pool) | Modify |
| `PROGRESS.md` | запись о фиче 5 | Modify |

---

## Task 1: mathcore/goals — sentinel errors и doc.go

**Files:**
- Create: `internal/mathcore/goals/doc.go`

- [ ] **Step 1: Создать doc.go с sentinel errors**

```go
// Package goals implements sinking-fund formulas for savings-goal planning
// (BUSINESS_LOGIC.md ф.5). Given a present capital P, a target amount S, a
// periodic contribution A, a per-period rate i, and a number of periods n,
// the package solves for any one unknown analytically.
//
// Design principle (CLAUDE.md §1 "Детерминизм"): all calculations use
// decimal.Decimal — never float64. No numerical solver is required: each
// unknown has a closed form (Копнова Гл. 3.3.3), so this package adds NO
// new float64 bridge. The project's documented float64 bridges remain at 2
// (credit/BrentQ for PSK + XIRR).
//
// Conventions:
//   - i is the per-period rate (annual_yield / 12 for monthly contributions)
//   - n is the number of periods (months)
//   - amounts are passed as decimal.Decimal; money-scale rounding is the
//     caller's responsibility (this package is pure math)
//
// Source: Копнова Г.П. "Финансовая математика" Гл. 3.3.3 "Фонд возмещения";
// MATH_FORMULAS.md §2.1 (annuity family).
package goals

import "errors"

// Sentinel errors. Callers branch on errors.Is.
var (
	// ErrNonPositiveTarget — target amount S must be > 0.
	ErrNonPositiveTarget = errors.New("goals: target amount must be > 0")
	// ErrNonPositiveContribution — periodic contribution A must be > 0 where required.
	ErrNonPositiveContribution = errors.New("goals: contribution must be > 0")
	// ErrInvalidPeriods — number of periods n must be > 0 where required.
	ErrInvalidPeriods = errors.New("goals: periods must be > 0")
	// ErrUnreachable — the contribution is too small to ever reach the target
	// (it does not cover the growth of interest on the present capital, so the
	// gap widens instead of closing). Returned by SolveTerm.
	ErrUnreachable = errors.New("goals: contribution too small to reach target")
	// ErrInvalidRate — periodic rate i must be > 0 for SolveTerm (Ln requires it).
	ErrInvalidRate = errors.New("goals: periodic rate must be > 0 for SolveTerm")
	// ErrDeflation100Percent — inflation = -1 would divide by zero in InflateTarget.
	ErrDeflation100Percent = errors.New("goals: inflation = -1 would divide by zero")
)
```

- [ ] **Step 2: Проверить сборку**

Run: `"C:/Program Files/Go/bin/go.exe" build ./internal/mathcore/goals/`
Expected: BUILD_OK (no output, exit 0)

- [ ] **Step 3: Commit**

```bash
git add internal/mathcore/goals/doc.go
git commit -m "feat(goals): mathcore package doc + sentinels"
```

---

## Task 2: SolveFutureValue + тесты

Формула: `S = P·(1+i)^n + A·((1+i)^n − 1)/i`. Если `i=0` → `S = P + A·n`.

**Files:**
- Create: `internal/mathcore/goals/sinkingfund.go`
- Create: `internal/mathcore/goals/sinkingfund_test.go`

- [ ] **Step 1: Написать падающий тест на SolveFutureValue**

`internal/mathcore/goals/sinkingfund_test.go`:
```go
package goals

import (
	"testing"

	"github.com/shopspring/decimal"
)

// decimalsEqual проверяет равенство с допуском 1e-6 (как в tvm-тестах).
func decimalsEqual(a, b decimal.Decimal) bool {
	diff := a.Sub(b).Abs()
	return diff.LessThan(decimal.NewFromFloat(1e-6))
}

func TestSolveFutureValue_CompoundGrowth(t *testing.T) {
	// P=100000, A=10000, i=0.01 (годовая 0.12 / 12), n=12 мес.
	P := decimal.NewFromInt(100000)
	A := decimal.NewFromInt(10000)
	i := decimal.NewFromFloat(0.01)
	n := 12
	// (1.01)^12 = 1.12682503... ; P*factor = 112682.50 ; A*(factor-1)/i = 1268250.30
	// Итого ≈ 1381507.80 (перепроверить калькулятором при реализации)
	got, err := SolveFutureValue(P, A, i, n)
	if err != nil {
		t.Fatalf("SolveFutureValue: %v", err)
	}
	// Эталон: P*(1.01)^12 = 112682.50; A*((1.01)^12-1)/0.01 = 1268250.30
	// Сумма = 1380932.80. Считаем точно decimal.Pow и сверяем до копейки.
	want := decimal.NewFromString("1380932.80") // placeholder — пересчитать в Step 3
	if !decimalsEqual(got, want) {
		t.Errorf("SolveFutureValue = %s, want near %s", got, want)
	}
}

func TestSolveFutureValue_ZeroRate_Fallback(t *testing.T) {
	// i=0 → S = P + A*n = 100000 + 10000*12 = 220000
	got, err := SolveFutureValue(decimal.NewFromInt(100000), decimal.NewFromInt(10000), decimal.Zero, 12)
	if err != nil {
		t.Fatalf("SolveFutureValue: %v", err)
	}
	if !got.Equal(decimal.NewFromInt(220000)) {
		t.Errorf("zero-rate fallback = %s, want 220000", got)
	}
}

func TestSolveFutureValue_ZeroPeriods_ReturnsPrincipal(t *testing.T) {
	// n=0 → S = P (нечего наращивать)
	got, err := SolveFutureValue(decimal.NewFromInt(100000), decimal.NewFromInt(10000), decimal.NewFromFloat(0.01), 0)
	if err != nil {
		t.Fatalf("SolveFutureValue: %v", err)
	}
	if !got.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("n=0 = %s, want 100000 (principal only)", got)
	}
}

func TestSolveFutureValue_NegativePeriods_Error(t *testing.T) {
	_, err := SolveFutureValue(decimal.NewFromInt(100000), decimal.NewFromInt(10000), decimal.NewFromFloat(0.01), -1)
	if err == nil {
		t.Errorf("expected ErrInvalidPeriods for n<0")
	}
}
```

- [ ] **Step 2: Запустить тест, убедиться что он падает (функция не определена)**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: FAIL / compile error "undefined: SolveFutureValue"

- [ ] **Step 3: Реализовать SolveFutureValue в sinkingfund.go**

`internal/mathcore/goals/sinkingfund.go`:
```go
package goals

import "github.com/shopspring/decimal"

// SolveFutureValue computes the accumulated amount after n periods:
//
//	S = P·(1+i)^n + A·((1+i)^n − 1)/i
//
// Variables: P present capital, A periodic contribution, i per-period rate,
// n number of periods.
// Source: Копнова Г.П. Гл. 3.3.3; MATH_FORMULAS.md §2.1.
//
// Edge cases:
//   - i = 0  → linear fallback S = P + A·n
//   - n = 0  → S = P
//   - n < 0  → ErrInvalidPeriods
func SolveFutureValue(P, A, i decimal.Decimal, n int) (decimal.Decimal, error) {
	if n < 0 {
		return decimal.Zero, ErrInvalidPeriods
	}
	if n == 0 {
		return P, nil
	}
	if i.IsZero() {
		// Без процентов: просто сумма взносов плюс исходный капитал.
		return P.Add(A.Mul(decimal.NewFromInt(int64(n)))), nil
	}
	one := decimal.NewFromInt(1)
	factor := one.Add(i).Pow(decimal.NewFromInt(int64(n))) // (1+i)^n
	principalGrown := P.Mul(factor)
	annuityPart := A.Mul(factor.Sub(one)).Div(i)
	return principalGrown.Add(annuityPart), nil
}
```

**Важно:** Пересчитать эталон в `TestSolveFutureValue_CompoundGrowth`. Реальная формула даёт:
- `(1.01)^12` = 1.12682503...
- `P·factor` = 100000·1.12682503 = **112682.503**
- `A·(factor−1)/i` = 10000·0.12682503/0.01 = 10000·12.682503 = **126825.03**
- **Итого = 239507.53** (исправить `want` на это значение, не placeholder)

- [ ] **Step 4: Запустить тест, убедиться что он проходит**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: PASS (4 теста)

- [ ] **Step 5: Commit**

```bash
git add internal/mathcore/goals/sinkingfund.go internal/mathcore/goals/sinkingfund_test.go
git commit -m "feat(goals): SolveFutureValue + tests"
```

---

## Task 3: SolveContribution + тесты

Формула: `A = (S − P·(1+i)^n) · i / ((1+i)^n − 1)`. Если `i=0` → `A = (S−P)/n`.

**Files:**
- Modify: `internal/mathcore/goals/sinkingfund.go` (append function)
- Modify: `internal/mathcore/goals/sinkingfund_test.go` (append tests)

- [ ] **Step 1: Дописать падающий тест на SolveContribution**

Добавить в `sinkingfund_test.go`:
```go
func TestSolveContribution_Compound(t *testing.T) {
	// Симметричный к SolveFutureValue: P=100000, S=239507.53, i=0.01, n=12 → A≈10000
	P := decimal.NewFromInt(100000)
	S := decimal.NewFromString("239507.53")
	i := decimal.NewFromFloat(0.01)
	got, err := SolveContribution(P, S, i, 12)
	if err != nil {
		t.Fatalf("SolveContribution: %v", err)
	}
	// Возврат должно дать ~10000 (с точностью до копеек округления эталона)
	if !decimalsEqual(got, decimal.NewFromInt(10000)) {
		t.Errorf("SolveContribution = %s, want ~10000", got)
	}
}

func TestSolveContribution_ZeroRate_Fallback(t *testing.T) {
	// i=0 → A = (S−P)/n = (200000−100000)/12 = 8333.33
	got, err := SolveContribution(decimal.NewFromInt(100000), decimal.NewFromInt(200000), decimal.Zero, 12)
	if err != nil {
		t.Fatalf("SolveContribution: %v", err)
	}
	want := decimal.NewFromString("8333.333333333333333333333333")
	if !decimalsEqual(got, want) {
		t.Errorf("zero-rate fallback = %s, want ~8333.33", got)
	}
}

func TestSolveContribution_NegativePeriods_Error(t *testing.T) {
	_, err := SolveContribution(decimal.NewFromInt(100000), decimal.NewFromInt(200000), decimal.NewFromFloat(0.01), -1)
	if err == nil {
		t.Errorf("expected ErrInvalidPeriods for n<0")
	}
}

func TestSolveContribution_AlreadyReached_Zero(t *testing.T) {
	// S <= P·(1+i)^n → взнос не нужен (уже достигли): (S - grown) <= 0 → A = 0
	// P=200000, S=100000, n=12 → grown > S, A должно быть 0 (не отрицательным)
	got, err := SolveContribution(decimal.NewFromInt(200000), decimal.NewFromInt(100000), decimal.NewFromFloat(0.01), 12)
	if err != nil {
		t.Fatalf("SolveContribution: %v", err)
	}
	if got.IsNegative() {
		t.Errorf("contribution must not be negative when target reached, got %s", got)
	}
}
```

- [ ] **Step 2: Запустить тест, убедиться что падает**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: FAIL "undefined: SolveContribution"

- [ ] **Step 3: Реализовать SolveContribution (добавить в sinkingfund.go)**

```go
// SolveContribution computes the periodic contribution needed to reach target
// S after n periods, given present capital P and per-period rate i:
//
//	A = (S − P·(1+i)^n) · i / ((1+i)^n − 1)
//
// Source: Копнова Г.П. Гл. 3.3.3 (inverse of SolveFutureValue).
//
// Edge cases:
//   - i = 0  → linear fallback A = (S − P)/n
//   - n <= 0 → ErrInvalidPeriods
//   - S <= P·(1+i)^n → returns 0 (target already reached by capital growth)
func SolveContribution(P, S, i decimal.Decimal, n int) (decimal.Decimal, error) {
	if n <= 0 {
		return decimal.Zero, ErrInvalidPeriods
	}
	if i.IsZero() {
		return S.Sub(P).Div(decimal.NewFromInt(int64(n))), nil
	}
	one := decimal.NewFromInt(1)
	factor := one.Add(i).Pow(decimal.NewFromInt(int64(n)))
	grown := P.Mul(factor)
	if !S.GreaterThan(grown) {
		// Цель уже достигнута ростом капитала — взнос не требуется.
		return decimal.Zero, nil
	}
	need := S.Sub(grown)
	return need.Mul(i).Div(factor.Sub(one)), nil
}
```

- [ ] **Step 4: Запустить тест, убедиться что проходит**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: PASS (все тесты)

- [ ] **Step 5: Commit**

```bash
git add internal/mathcore/goals/sinkingfund.go internal/mathcore/goals/sinkingfund_test.go
git commit -m "feat(goals): SolveContribution + tests"
```

---

## Task 4: SolveTerm + тесты

Формула: `n = ln((S·i + A)/(A + P·i)) / ln(1+i)`.

**Files:**
- Modify: `internal/mathcore/goals/sinkingfund.go`
- Modify: `internal/mathcore/goals/sinkingfund_test.go`

- [ ] **Step 1: Дописать падающий тест на SolveTerm**

Добавить в `sinkingfund_test.go`:
```go
func TestSolveTerm_BasicCase(t *testing.T) {
	// P=0, S=1000000, A=50000, i=0.01 → n = ln((10000+50000)/(50000+0))/ln(1.01)
	//   = ln(60000/50000)/ln(1.01) = ln(1.2)/ln(1.01) = 0.18232/0.00995 = 18.32 мес
	got, err := SolveTerm(decimal.Zero, decimal.NewFromInt(1000000), decimal.NewFromInt(50000), decimal.NewFromFloat(0.01))
	if err != nil {
		t.Fatalf("SolveTerm: %v", err)
	}
	// Должно быть ~18 месяцев (округляем до целых вверху при применении)
	if got.LessThan(decimal.NewFromInt(18)) || got.GreaterThan(decimal.NewFromInt(19)) {
		t.Errorf("SolveTerm = %s, want between 18 and 19", got)
	}
}

func TestSolveTerm_Unreachable_ContributionTooSmall(t *testing.T) {
	// P=0, S=1000000, A=1 (копейки), i=0.50 (50% мес — абсурд): A < S·i
	// → числитель (S·i + A)/(A + P·i) = 500001/1 > 1, но знаменатель ln(1+i)OK
	// На самом деле unreachable: A ≤ S·i не бывает unreachable, надо P·i ≥ A·...:
	// Возьмём P=1000000, i=0.50, A=1, S=100: A < P·i (1 < 500000) → растёт быстрее
	_, err := SolveTerm(decimal.NewFromInt(1000000), decimal.NewFromInt(2000000), decimal.NewFromInt(1), decimal.NewFromFloat(0.50))
	if err != ErrUnreachable {
		t.Errorf("expected ErrUnreachable, got %v", err)
	}
}

func TestSolveTerm_ZeroRate_Error(t *testing.T) {
	// i=0 → Ln(1) = 0, деление на 0. ErrInvalidRate.
	_, err := SolveTerm(decimal.Zero, decimal.NewFromInt(1000000), decimal.NewFromInt(50000), decimal.Zero)
	if err != ErrInvalidRate {
		t.Errorf("expected ErrInvalidRate for i=0, got %v", err)
	}
}

func TestSolveTerm_AlreadyReached_Zero(t *testing.T) {
	// P >= S → уже накопили, n=0
	got, err := SolveTerm(decimal.NewFromInt(1000000), decimal.NewFromInt(500000), decimal.NewFromInt(100), decimal.NewFromFloat(0.01))
	if err != nil {
		t.Fatalf("SolveTerm: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("expected n=0 when P>=S, got %s", got)
	}
}
```

- [ ] **Step 2: Запустить тест, убедиться что падает**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: FAIL "undefined: SolveTerm"

- [ ] **Step 3: Реализовать SolveTerm (добавить в sinkingfund.go)**

```go
// SolveTerm computes the number of periods needed to reach target S with
// present capital P, periodic contribution A, and per-period rate i:
//
//	n = ln((S·i + A) / (A + P·i)) / ln(1+i)
//
// Returns a decimal (caller rounds up to whole periods).
// Source: Копнова Г.П. Гл. 3.3.3 (inverse of SolveFutureValue for n).
//
// Edge cases:
//   - i <= 0  → ErrInvalidRate (Ln requires 1+i > 0 AND != 1)
//   - P >= S  → returns 0 (already reached)
//   - A <= P·i → ErrUnreachable (contribution doesn't outpace interest growth
//     of present capital; the target recedes instead of approaching)
func SolveTerm(P, S, A, i decimal.Decimal) (decimal.Decimal, error) {
	if !i.IsPositive() {
		return decimal.Zero, ErrInvalidRate
	}
	if P.GreaterThanOrEqual(S) {
		return decimal.Zero, nil
	}
	// Unreachable: вклады A не превышают рост P·i → S убегает.
	if A.LessThanOrEqual(P.Mul(i)) {
		return decimal.Zero, ErrUnreachable
	}
	one := decimal.NewFromInt(1)
	num := S.Mul(i).Add(A)         // S·i + A
	denom := A.Add(P.Mul(i))       // A + P·i
	ratio := num.Div(denom)        // должно быть > 1 (проверено выше)
	// ratio = (1+i)^n → n = ln(ratio)/ln(1+i)
	return ratio.Ln().Div(one.Add(i).Ln()), nil
}
```

- [ ] **Step 4: Запустить тест, убедиться что проходит**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mathcore/goals/sinkingfund.go internal/mathcore/goals/sinkingfund_test.go
git commit -m "feat(goals): SolveTerm + tests"
```

---

## Task 5: InflateTarget + тесты

Формула: `S_инфл = S·(1+π)^(n/12)` (n в месяцах → годовая степень).

**Files:**
- Modify: `internal/mathcore/goals/sinkingfund.go`
- Modify: `internal/mathcore/goals/sinkingfund_test.go`

- [ ] **Step 1: Дописать падающий тест**

Добавить в `sinkingfund_test.go`:
```go
func TestInflateTarget_PositiveInflation(t *testing.T) {
	// S=1000000, π=0.06, n=24 мес (2 года) → 1000000·(1.06)^2 = 1123600
	got, err := InflateTarget(decimal.NewFromInt(1000000), decimal.NewFromFloat(0.06), 24)
	if err != nil {
		t.Fatalf("InflateTarget: %v", err)
	}
	want := decimal.NewFromString("1123600")
	if !decimalsEqual(got, want) {
		t.Errorf("InflateTarget = %s, want ~1123600", got)
	}
}

func TestInflateTarget_ZeroInflation_Unchanged(t *testing.T) {
	got, err := InflateTarget(decimal.NewFromInt(1000000), decimal.Zero, 24)
	if err != nil {
		t.Fatalf("InflateTarget: %v", err)
	}
	if !got.Equal(decimal.NewFromInt(1000000)) {
		t.Errorf("zero inflation should return S unchanged, got %s", got)
	}
}

func TestInflateTarget_Deflation100_Error(t *testing.T) {
	_, err := InflateTarget(decimal.NewFromInt(1000000), decimal.NewFromInt(-1), 24)
	if err != ErrDeflation100Percent {
		t.Errorf("expected ErrDeflation100Percent, got %v", err)
	}
}
```

- [ ] **Step 2: Запустить тест, убедиться что падает**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: FAIL "undefined: InflateTarget"

- [ ] **Step 3: Реализовать InflateTarget (добавить в sinkingfund.go)**

```go
// InflateTarget adjusts a nominal target for inflation over n months:
//
//	S_инфл = S · (1+π)^(n/12)
//
// Variables: S nominal target, π annual inflation (fraction), n months.
// Source: MATH_FORMULAS.md §1.4 (Fisher); BUSINESS_LOGIC ф.5 "корректировка
// цели на инфляцию".
//
// Edge cases:
//   - π = 0  → returns S unchanged
//   - π = -1 → ErrDeflation100Percent (division-by-zero equivalent)
func InflateTarget(S, inflation decimal.Decimal, months int) (decimal.Decimal, error) {
	if inflation.IsZero() {
		return S, nil
	}
	one := decimal.NewFromInt(1)
	denom := one.Add(inflation)
	if denom.IsZero() {
		return decimal.Zero, ErrDeflation100Percent
	}
	yearsExp := decimal.NewFromInt(int64(months)).Div(decimal.NewFromInt(12))
	factor := denom.Pow(yearsExp)
	return S.Mul(factor), nil
}
```

- [ ] **Step 4: Полная проверка mathcore**

Run: `"C:/Program Files/Go/bin/go.exe" build ./... && "C:/Program Files/Go/bin/go.exe" vet ./... && "C:/Program Files/Go/bin/go.exe" test ./internal/mathcore/goals/`
Expected: BUILD_OK / VET_OK / PASS (все тесты)

- [ ] **Step 5: Commit**

```bash
git add internal/mathcore/goals/sinkingfund.go internal/mathcore/goals/sinkingfund_test.go
git commit -m "feat(goals): InflateTarget + complete mathcore goals package"
```

---

## Task 6: domain/goal.go — типы и валидация

**Files:**
- Create: `internal/domain/goal.go`

- [ ] **Step 1: Создать domain/goal.go**

```go
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Goal — финансовая цель пользователя (BUSINESS_LOGIC ф.5). Зеркалирует таблицу
// goals из миграции 0001. Money-поля — всегда domain.Money (decimal, scale=2).
type Goal struct {
	ID                  int64
	UserID              int64
	Name                string
	TargetAmount        Money
	CurrentAmount       Money           // baseline «уже накоплено до старта трекинга»
	MonthlyContribution *Money          // nil = регулярного взноса нет
	TargetDate          *time.Time      // nil = без дедлайна
	ExpectedYield       decimal.Decimal // годовая ставка, напр. 0.08
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// GoalContribution — запись о внеплановом пополнении цели (BUSINESS_LOGIC ф.5).
// Идемпотентность по (user_id, goal_id, contribution_id) — аналогично calc_id
// в operations (ф.1).
type GoalContribution struct {
	ID              int64
	UserID          int64
	GoalID          int64
	ContributionID  string // client-generated
	Amount          Money
	ContributionDate time.Time
	Comment         string
	CreatedAt       time.Time
}

// GoalStatus — ярлык здоровья цели для UI и Projection.
type GoalStatus string

const (
	// StatusOnTrack — взнос/срок достаточны.
	StatusGoalOnTrack GoalStatus = "on_track"
	// StatusAtRisk — на грани (взноса едва хватает; вакансия < 10% запаса).
	StatusGoalAtRisk GoalStatus = "at_risk"
	// StatusBehind — не успеем накопить к дедлайну (взнос < требуемого).
	StatusGoalBehind GoalStatus = "behind"
	// StatusAchieved — effective current >= target.
	StatusGoalAchieved GoalStatus = "achieved"
	// StatusNoDeadline — target_date не задан и нет monthly_contribution.
	StatusGoalNoDeadline GoalStatus = "no_deadline"
)

// ValidateGoal проверяет инварианты цели перед сохранением.
// Возвращает ошибки для: пустого name, target<=0, expected_yield<0,
// target_date в прошлом, monthly_contribution<0.
func ValidateGoal(name string, target Money, yield decimal.Decimal, targetDate *time.Time, monthly *Money, now time.Time) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("goal: name required")
	}
	if !target.IsPositive() {
		return errors.New("goal: target_amount must be positive")
	}
	if yield.IsNegative() {
		return errors.New("goal: expected_yield must be >= 0")
	}
	if targetDate != nil && targetDate.Before(now) {
		return errors.New("goal: target_date must be in the future")
	}
	if monthly != nil && monthly.IsNegative() {
		return errors.New("goal: monthly_contribution must be >= 0")
	}
	return nil
}
```

- [ ] **Step 2: Проверить сборку**

Run: `"C:/Program Files/Go/bin/go.exe" build ./...`
Expected: BUILD_OK

- [ ] **Step 3: Commit**

```bash
git add internal/domain/goal.go
git commit -m "feat(goals): domain types Goal, GoalContribution, GoalStatus, ValidateGoal"
```

---

## Task 7: Миграция 0003_goals_contributions.sql

**Files:**
- Create: `migrations/0003_goals_contributions.sql`

- [ ] **Step 1: Создать миграцию**

```sql
-- FinHelper — migration 0003: goal contributions journal (ф.5)
-- Журнал внеплановых пополнений целей. Идемпотентность по
-- (user_id, goal_id, contribution_id) — аналогично calc_id в operations (ф.1).

CREATE TABLE goal_contributions (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id           BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    goal_id           BIGINT NOT NULL REFERENCES goals (id) ON DELETE CASCADE,
    contribution_id   TEXT NOT NULL,
    amount            NUMERIC(28, 2) NOT NULL CHECK (amount > 0 AND amount = ROUND(amount, 2)),
    contribution_date DATE NOT NULL,
    comment           TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, goal_id, contribution_id)
);

CREATE INDEX goal_contributions_goal_idx ON goal_contributions (goal_id);

CREATE TRIGGER goal_contributions_touch BEFORE UPDATE ON goal_contributions
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
```

- [ ] **Step 2: Commit**

```bash
git add migrations/0003_goals_contributions.sql
git commit -m "feat(goals): migration 0003 — goal_contributions table"
```

---

## Task 8: storage/goals.go — CRUD целей

**Files:**
- Create: `internal/storage/goals.go`
- Create: `internal/storage/goals_test.go`

- [ ] **Step 1: Написать storage/goals.go с типами + CRUD целей**

`internal/storage/goals.go`:
```go
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/shopspring/decimal"
)

// Storage-level mirror of domain.Goal. Kept separate so storage doesn't import
// domain's pointer-bearing types; conversion happens at the boundary.
// (Следуем паттерну из budgets.go — domain.Money используется напрямую.)

// Sentinel errors для goals + contributions.
var (
	ErrGoalNotFound          = errors.New("storage: goal not found")
	ErrContributionExists    = errors.New("storage: contribution already exists")
	ErrContributionNotFound  = errors.New("storage: contribution not found")
)

// CreateGoal inserts a goal. Возвращает записанную цель с id/timestamps.
func (p *Pool) CreateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error) {
	if g.UserID == 0 {
		return domain.Goal{}, errors.New("storage: goal requires user_id")
	}
	const q = `
		INSERT INTO goals (user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	var (
		monthlyNull sql.NullString
		dateNull    sql.NullTime
	)
	if g.MonthlyContribution != nil {
		monthlyNull = sql.NullString{String: g.MonthlyContribution.String(), Valid: true}
	}
	if g.TargetDate != nil {
		dateNull = sql.NullTime{Time: *g.TargetDate, Valid: true}
	}
	err := p.DB.QueryRowContext(ctx, q,
		g.UserID, g.Name, g.TargetAmount.Decimal(), g.CurrentAmount.Decimal(),
		monthlyNull, dateNull, g.ExpectedYield,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return domain.Goal{}, fmt.Errorf("storage: create goal: %w", err)
	}
	return g, nil
}

// GetGoal возвращает цель по (userID, id). Не найдена → ErrGoalNotFound.
func (p *Pool) GetGoal(ctx context.Context, userID, id int64) (domain.Goal, error) {
	const q = `
		SELECT id, user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield, created_at, updated_at
		FROM goals
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	var (
		g             domain.Goal
		targetRaw     = new(decimalScanner)
		currentRaw    = new(decimalScanner)
		yieldRaw      = new(decimalScanner)
		monthlyNull   sql.NullString
		dateNull      sql.NullTime
	)
	err := p.DB.QueryRowContext(ctx, q, id, userID).Scan(
		&g.ID, &g.UserID, &g.Name, targetRaw, currentRaw, &monthlyNull, &dateNull, yieldRaw, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Goal{}, ErrGoalNotFound
		}
		return domain.Goal{}, fmt.Errorf("storage: get goal: %w", err)
	}
	g.TargetAmount = domain.FromDecimal(targetRaw.d)
	g.CurrentAmount = domain.FromDecimal(currentRaw.d)
	g.ExpectedYield = yieldRaw.d
	if monthlyNull.Valid {
		m, _ := domain.ParseMoney(monthlyNull.String)
		g.MonthlyContribution = &m
	}
	if dateNull.Valid {
		t := dateNull.Time
		g.TargetDate = &t
	}
	return g, nil
}

// ListGoals возвращает все неудалённые цели пользователя.
func (p *Pool) ListGoals(ctx context.Context, userID int64) ([]domain.Goal, error) {
	const q = `
		SELECT id, user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield, created_at, updated_at
		FROM goals
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY id
	`
	rows, err := p.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: list goals: %w", err)
	}
	defer rows.Close()

	var out []domain.Goal
	for rows.Next() {
		var (
			g           domain.Goal
			targetRaw   = new(decimalScanner)
			currentRaw  = new(decimalScanner)
			yieldRaw    = new(decimalScanner)
			monthlyNull sql.NullString
			dateNull    sql.NullTime
		)
		if err := rows.Scan(&g.ID, &g.UserID, &g.Name, targetRaw, currentRaw, &monthlyNull, &dateNull, yieldRaw, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan goal: %w", err)
		}
		g.TargetAmount = domain.FromDecimal(targetRaw.d)
		g.CurrentAmount = domain.FromDecimal(currentRaw.d)
		g.ExpectedYield = yieldRaw.d
		if monthlyNull.Valid {
			m, _ := domain.ParseMoney(monthlyNull.String)
			g.MonthlyContribution = &m
		}
		if dateNull.Valid {
			t := dateNull.Time
			g.TargetDate = &t
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// UpdateGoal мутирует редактируемые поля. id/user_id — identity.
func (p *Pool) UpdateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error) {
	const q = `
		UPDATE goals
		SET name = $1, target_amount = $2, current_amount = $3, monthly_contribution = $4, target_date = $5, expected_yield = $6
		WHERE id = $7 AND user_id = $8 AND deleted_at IS NULL
		RETURNING id, user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield, created_at, updated_at
	`
	var (
		monthlyNull sql.NullString
		dateNull    sql.NullTime
	)
	if g.MonthlyContribution != nil {
		monthlyNull = sql.NullString{String: g.MonthlyContribution.String(), Valid: true}
	}
	if g.TargetDate != nil {
		dateNull = sql.NullTime{Time: *g.TargetDate, Valid: true}
	}
	var (
		out          domain.Goal
		targetRaw    = new(decimalScanner)
		currentRaw   = new(decimalScanner)
		yieldRaw     = new(decimalScanner)
		monthlyBack  sql.NullString
		dateBack     sql.NullTime
	)
	err := p.DB.QueryRowContext(ctx, q,
		g.Name, g.TargetAmount.Decimal(), g.CurrentAmount.Decimal(),
		monthlyNull, dateNull, g.ExpectedYield, g.ID, g.UserID,
	).Scan(&out.ID, &out.UserID, &out.Name, targetRaw, currentRaw, &monthlyBack, &dateBack, yieldRaw, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Goal{}, ErrGoalNotFound
		}
		return domain.Goal{}, fmt.Errorf("storage: update goal: %w", err)
	}
	out.TargetAmount = domain.FromDecimal(targetRaw.d)
	out.CurrentAmount = domain.FromDecimal(currentRaw.d)
	out.ExpectedYield = yieldRaw.d
	if monthlyBack.Valid {
		m, _ := domain.ParseMoney(monthlyBack.String)
		out.MonthlyContribution = &m
	}
	if dateBack.Valid {
		t := dateBack.Time
		out.TargetDate = &t
	}
	return out, nil
}

// DeleteGoal мягко удаляет цель.
func (p *Pool) DeleteGoal(ctx context.Context, userID, id int64) error {
	const q = `UPDATE goals SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	res, err := p.DB.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("storage: delete goal: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete goal rows: %w", err)
	}
	if n == 0 {
		return ErrGoalNotFound
	}
	return nil
}

// SumContributions возвращает Σ amount по contributions цели. Используется в
// Projection для гибридной модели effective = current + Σ contributions.
// Возвращает Zero при отсутствии пополнений (COALESCE).
func (p *Pool) SumContributions(ctx context.Context, userID, goalID int64) (domain.Money, error) {
	const q = `SELECT COALESCE(SUM(amount), 0) FROM goal_contributions WHERE user_id = $1 AND goal_id = $2`
	tot := new(decimalScanner)
	if err := p.DB.QueryRowContext(ctx, q, userID, goalID).Scan(tot); err != nil {
		return domain.Zero, fmt.Errorf("storage: sum contributions: %w", err)
	}
	return domain.FromDecimal(tot.d), nil
}

// проверка, что decimal импортирован (для будущего использования)
var _ = decimal.Zero
var _ time.Duration
```

- [ ] **Step 2: Проверить сборку**

Run: `"C:/Program Files/Go/bin/go.exe" build ./...`
Expected: BUILD_OK

- [ ] **Step 3: Commit**

```bash
git add internal/storage/goals.go
git commit -m "feat(goals): storage CRUD for goals + SumContributions"
```

---

## Task 9: storage/goals.go — CRUD contributions + идемпотентность

**Files:**
- Modify: `internal/storage/goals.go` (append)
- Create: `internal/storage/goals_test.go`

- [ ] **Step 1: Дописать методы contributions в storage/goals.go**

Добавить в конец файла:
```go
// CreateContribution вставляет запись о пополнении. Идемпотентность:
// (user_id, goal_id, contribution_id) UNIQUE → 23505 транслируется в
// ErrContributionExists. Стиль операций (ф.1): INSERT…RETURNING + sentinel.
func (p *Pool) CreateContribution(ctx context.Context, c domain.GoalContribution) (domain.GoalContribution, error) {
	if c.UserID == 0 || c.GoalID == 0 {
		return domain.GoalContribution{}, errors.New("storage: contribution requires user_id and goal_id")
	}
	if c.ContributionID == "" {
		return domain.GoalContribution{}, errors.New("storage: contribution_id required")
	}
	const q = `
		INSERT INTO goal_contributions (user_id, goal_id, contribution_id, amount, contribution_date, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := p.DB.QueryRowContext(ctx, q,
		c.UserID, c.GoalID, c.ContributionID, c.Amount.Decimal(), c.ContributionDate, c.Comment,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return domain.GoalContribution{}, translatePgError(err, "goal_contributions_user_id_goal_id_contribution_id_key", ErrContributionExists)
	}
	return c, nil
}

// GetContributionByClientID возвращает запись по (userID, goalID, contributionID).
// Используется для идемпотентного ответа при ErrContributionExists (как
// operations.GetOperationByCalcID в ф.1).
func (p *Pool) GetContributionByClientID(ctx context.Context, userID, goalID int64, contributionID string) (domain.GoalContribution, error) {
	const q = `
		SELECT id, user_id, goal_id, contribution_id, amount, contribution_date, comment, created_at
		FROM goal_contributions
		WHERE user_id = $1 AND goal_id = $2 AND contribution_id = $3
	`
	var (
		c             domain.GoalContribution
		amountRaw     = new(decimalScanner)
		dateRaw       time.Time
	)
	err := p.DB.QueryRowContext(ctx, q, userID, goalID, contributionID).Scan(
		&c.ID, &c.UserID, &c.GoalID, &c.ContributionID, amountRaw, &dateRaw, &c.Comment, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.GoalContribution{}, ErrContributionNotFound
		}
		return domain.GoalContribution{}, fmt.Errorf("storage: get contribution: %w", err)
	}
	c.Amount = domain.FromDecimal(amountRaw.d)
	c.ContributionDate = dateRaw
	return c, nil
}

// ListContributions возвращает журнал пополнений цели (хронологически).
func (p *Pool) ListContributions(ctx context.Context, userID, goalID int64) ([]domain.GoalContribution, error) {
	const q = `
		SELECT id, user_id, goal_id, contribution_id, amount, contribution_date, comment, created_at
		FROM goal_contributions
		WHERE user_id = $1 AND goal_id = $2
		ORDER BY contribution_date DESC, id DESC
	`
	rows, err := p.DB.QueryContext(ctx, q, userID, goalID)
	if err != nil {
		return nil, fmt.Errorf("storage: list contributions: %w", err)
	}
	defer rows.Close()

	var out []domain.GoalContribution
	for rows.Next() {
		var (
			c         domain.GoalContribution
			amountRaw = new(decimalScanner)
			dateRaw   time.Time
		)
		if err := rows.Scan(&c.ID, &c.UserID, &c.GoalID, &c.ContributionID, amountRaw, &dateRaw, &c.Comment, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan contribution: %w", err)
		}
		c.Amount = domain.FromDecimal(amountRaw.d)
		c.ContributionDate = dateRaw
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteContribution удаляет запись по (userID, goalID, contribution row id).
func (p *Pool) DeleteContribution(ctx context.Context, userID, goalID, id int64) error {
	const q = `DELETE FROM goal_contributions WHERE id = $1 AND user_id = $2 AND goal_id = $3`
	res, err := p.DB.ExecContext(ctx, q, id, userID, goalID)
	if err != nil {
		return fmt.Errorf("storage: delete contribution: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete contribution rows: %w", err)
	}
	if n == 0 {
		return ErrContributionNotFound
	}
	return nil
}
```

Удалить временные строки `var _ = decimal.Zero` и `var _ time.Duration` из Task 8 (они были плейсхолдерами для компиляции до добавления contributions-методов; теперь decimal используется в коде). Проверить, что импорт `time` всё ещё нужен (используется в `time.Time`/`sql.NullTime`) — да. Если `decimal` больше не нужен — убрать импорт.

- [ ] **Step 2: Проверить сборку и vet**

Run: `"C:/Program Files/Go/bin/go.exe" build ./... && "C:/Program Files/Go/bin/go.exe" vet ./...`
Expected: BUILD_OK / VET_OK (без неиспользуемых импортов)

- [ ] **Step 3: Написать storage/goals_test.go — тесты CRUD целей**

`internal/storage/goals_test.go`:
```go
package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/shopspring/decimal"
)

func TestCreateGoal_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	yield := decimal.NewFromFloat(0.08)

	mock.ExpectQuery(q(`INSERT INTO goals`)).
		WithArgs(int64(7), "Машина", sqlmock.AnyArg(), sqlmock.AnyArg(), sql.NullString{}, sql.NullTime{}, yield).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), now, now))

	g, err := pool.CreateGoal(ctx, domain.Goal{
		UserID: 7, Name: "Машина",
		TargetAmount:  domain.MustParseMoney("1000000.00"),
		CurrentAmount: domain.Zero,
		ExpectedYield: yield,
	})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g.ID != 1 {
		t.Errorf("id = %d, want 1", g.ID)
	}
}

func TestGetGoal_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`SELECT .* FROM goals WHERE id`)).
		WithArgs(int64(99), int64(7)).
		WillReturnError(sql.ErrNoRows)
	_, err := pool.GetGoal(context.Background(), 7, 99)
	if !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

func TestDeleteGoal_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectExec(q(`UPDATE goals SET deleted_at`)).
		WithArgs(int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err := pool.DeleteGoal(context.Background(), 7, 99)
	if !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

func TestSumContributions_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`SELECT COALESCE\(SUM\(amount\), 0\) FROM goal_contributions`)).
		WithArgs(int64(7), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow("75000.00"))
	got, err := pool.SumContributions(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("SumContributions: %v", err)
	}
	if !got.Equal(domain.MustParseMoney("75000.00")) {
		t.Errorf("sum = %s, want 75000.00", got)
	}
}

func TestCreateContribution_Duplicate(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`INSERT INTO goal_contributions`)).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "goal_contributions_user_id_goal_id_contribution_id_key", Message: "dup"})
	_, err := pool.CreateContribution(context.Background(), domain.GoalContribution{
		UserID: 7, GoalID: 1, ContributionID: "abc",
		Amount: domain.MustParseMoney("5000.00"), ContributionDate: time.Now(),
	})
	if !errors.Is(err, ErrContributionExists) {
		t.Fatalf("expected ErrContributionExists, got %v", err)
	}
}

func TestCreateContribution_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(q(`INSERT INTO goal_contributions`)).
		WithArgs(int64(7), int64(1), "abc", sqlmock.AnyArg(), sqlmock.AnyArg(), "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(10), now))
	c, err := pool.CreateContribution(context.Background(), domain.GoalContribution{
		UserID: 7, GoalID: 1, ContributionID: "abc",
		Amount: domain.MustParseMoney("5000.00"), ContributionDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateContribution: %v", err)
	}
	if c.ID != 10 {
		t.Errorf("id = %d, want 10", c.ID)
	}
}

func TestDeleteContribution_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectExec(q(`DELETE FROM goal_contributions`)).
		WithArgs(int64(99), int64(7), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err := pool.DeleteContribution(context.Background(), 7, 1, 99)
	if !errors.Is(err, ErrContributionNotFound) {
		t.Fatalf("expected ErrContributionNotFound, got %v", err)
	}
}
```

- [ ] **Step 4: Запустить тесты storage**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/storage/`
Expected: PASS (все тесты, включая существующие для budget/operations/etc.)

- [ ] **Step 5: Commit**

```bash
git add internal/storage/goals.go internal/storage/goals_test.go
git commit -m "feat(goals): storage contributions CRUD + idempotency + sqlmock tests"
```

---

## Task 10: service/goals.go — Repo interface, Service, CRUD + идемпотентность

**Files:**
- Create: `internal/service/goals/goals.go`

- [ ] **Step 1: Создать service/goals/goals.go (без Projection/Simulate — это Task 11)**

`internal/service/goals/goals.go`:
```go
// Package goals implements BUSINESS_LOGIC.md ф.5 — savings-goal tracker with
// recurring contributions, ad-hoc top-ups, and a projection of status.
//
// Слои (симметрично budget): storage-agnostic через Repo interface → unit-
// тесты на in-memory fakeRepo, интеграция через *storage.Pool.
//
// Гибридная модель current_amount (см. spec §3.2):
//   effective_current = goal.current_amount (baseline) + Σ goal_contributions
//
// Money — decimal end-to-end; service не касается float64.
package goals

import (
	"context"
	"errors"
	"time"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/mathcore/goals"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// Repo — storage contract. Реализуется *storage.Pool; в тестах — fakeRepo.
type Repo interface {
	CreateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error)
	GetGoal(ctx context.Context, userID, id int64) (domain.Goal, error)
	ListGoals(ctx context.Context, userID int64) ([]domain.Goal, error)
	UpdateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error)
	DeleteGoal(ctx context.Context, userID, id int64) error
	SumContributions(ctx context.Context, userID, goalID int64) (domain.Money, error)
	CreateContribution(ctx context.Context, c domain.GoalContribution) (domain.GoalContribution, error)
	GetContributionByClientID(ctx context.Context, userID, goalID int64, contributionID string) (domain.GoalContribution, error)
	ListContributions(ctx context.Context, userID, goalID int64) ([]domain.GoalContribution, error)
	DeleteContribution(ctx context.Context, userID, goalID, id int64) error
}

// Service — бизнес-слой целей. Конструируется один раз, без per-request state.
type Service struct {
	repo Repo
	now  func() time.Time
}

// NewService возвращает Service. repo должен быть non-nil.
func NewService(repo Repo) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Sentinel errors.
var (
	ErrInvalidArgument = errors.New("goals: invalid argument")
	ErrNotFound        = errors.New("goals: not found")
)

// CreateInput — параметры создания цели.
type CreateInput struct {
	UserID             int64
	Name               string
	TargetAmount       domain.Money
	CurrentAmount      domain.Money
	MonthlyContribution *domain.Money
	TargetDate         *time.Time
	ExpectedYield      decimalYield
}
```

**ВАЖНО:** `decimalYield` выше — опечатка, нужен `decimal.Decimal`. Исправить:
```go
import "github.com/shopspring/decimal"

type CreateInput struct {
	UserID              int64
	Name                string
	TargetAmount        domain.Money
	CurrentAmount       domain.Money
	MonthlyContribution *domain.Money
	TargetDate          *time.Time
	ExpectedYield       decimal.Decimal
}
```

Добавить импорт `"github.com/shopspring/decimal"` в блок imports.

Продолжить файл (методы CRUD):
```go
// Create валидирует и сохраняет цель.
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Goal, error) {
	if in.UserID == 0 {
		return domain.Goal{}, errors.New("goals: user_id required")
	}
	if err := domain.ValidateGoal(in.Name, in.TargetAmount, in.ExpectedYield, in.TargetDate, in.MonthlyContribution, s.now()); err != nil {
		return domain.Goal{}, err
	}
	if in.CurrentAmount.IsNegative() {
		return domain.Goal{}, errors.New("goals: current_amount must be >= 0")
	}
	g, err := s.repo.CreateGoal(ctx, domain.Goal{
		UserID: in.UserID, Name: in.Name,
		TargetAmount:        in.TargetAmount,
		CurrentAmount:       in.CurrentAmount,
		MonthlyContribution: in.MonthlyContribution,
		TargetDate:          in.TargetDate,
		ExpectedYield:       in.ExpectedYield,
	})
	if err != nil {
		return domain.Goal{}, err
	}
	return g, nil
}

// Get возвращает цель по id, проверяя владельца.
func (s *Service) Get(ctx context.Context, userID, id int64) (domain.Goal, error) {
	g, err := s.repo.GetGoal(ctx, userID, id)
	if err != nil {
		return domain.Goal{}, mapStorageErr(err)
	}
	return g, nil
}

// List возвращает все цели пользователя.
func (s *Service) List(ctx context.Context, userID int64) ([]domain.Goal, error) {
	gs, err := s.repo.ListGoals(ctx, userID)
	if err != nil {
		return nil, mapStorageErr(err)
	}
	if gs == nil {
		gs = []domain.Goal{}
	}
	return gs, nil
}

// UpdateInput — параметры обновления.
type UpdateInput struct {
	UserID              int64
	ID                  int64
	Name                string
	TargetAmount        domain.Money
	CurrentAmount       domain.Money
	MonthlyContribution *domain.Money
	TargetDate          *time.Time
	ExpectedYield       decimal.Decimal
}

// Update валидирует и применяет мутацию.
func (s *Service) Update(ctx context.Context, in UpdateInput) (domain.Goal, error) {
	if in.UserID == 0 || in.ID == 0 {
		return domain.Goal{}, errors.New("goals: user_id and id required")
	}
	if err := domain.ValidateGoal(in.Name, in.TargetAmount, in.ExpectedYield, in.TargetDate, in.MonthlyContribution, s.now()); err != nil {
		return domain.Goal{}, err
	}
	g, err := s.repo.UpdateGoal(ctx, domain.Goal{
		ID: in.ID, UserID: in.UserID, Name: in.Name,
		TargetAmount:        in.TargetAmount,
		CurrentAmount:       in.CurrentAmount,
		MonthlyContribution: in.MonthlyContribution,
		TargetDate:          in.TargetDate,
		ExpectedYield:       in.ExpectedYield,
	})
	if err != nil {
		return domain.Goal{}, mapStorageErr(err)
	}
	return g, nil
}

// Delete мягко удаляет цель.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	if err := s.repo.DeleteGoal(ctx, userID, id); err != nil {
		return mapStorageErr(err)
	}
	return nil
}

// AddContributionInput — параметры внепланового пополнения.
type AddContributionInput struct {
	UserID         int64
	GoalID         int64
	ContributionID string
	Amount         domain.Money
	Date           time.Time
	Comment        string
}

// AddContribution записывает пополнение с идемпотентностью (стиль ф.1):
// при ErrContributionExists возвращает оригинал по (user, goal, contribution_id).
func (s *Service) AddContribution(ctx context.Context, in AddContributionInput) (domain.GoalContribution, bool, error) {
	if in.UserID == 0 || in.GoalID == 0 {
		return domain.GoalContribution{}, false, errors.New("goals: user_id and goal_id required")
	}
	if in.ContributionID == "" {
		return domain.GoalContribution{}, false, errors.New("goals: contribution_id required")
	}
	if !in.Amount.IsPositive() {
		return domain.GoalContribution{}, false, errors.New("goals: amount must be positive")
	}
	// Проверяем, что цель существует и принадлежит пользователю.
	if _, err := s.repo.GetGoal(ctx, in.UserID, in.GoalID); err != nil {
		return domain.GoalContribution{}, false, mapStorageErr(err)
	}
	date := in.Date
	if date.IsZero() {
		date = s.now()
	}
	c, err := s.repo.CreateContribution(ctx, domain.GoalContribution{
		UserID: in.UserID, GoalID: in.GoalID, ContributionID: in.ContributionID,
		Amount: in.Amount, ContributionDate: date, Comment: in.Comment,
	})
	if err != nil {
		if errors.Is(err, storage.ErrContributionExists) {
			// Идемпотентный ответ: вернуть оригинал.
			orig, gErr := s.repo.GetContributionByClientID(ctx, in.UserID, in.GoalID, in.ContributionID)
			if gErr != nil {
				return domain.GoalContribution{}, false, gErr
			}
			return orig, true, nil
		}
		return domain.GoalContribution{}, false, err
	}
	return c, false, nil
}

// ListContributions возвращает журнал пополнений цели.
func (s *Service) ListContributions(ctx context.Context, userID, goalID int64) ([]domain.GoalContribution, error) {
	cs, err := s.repo.ListContributions(ctx, userID, goalID)
	if err != nil {
		return nil, mapStorageErr(err)
	}
	if cs == nil {
		cs = []domain.GoalContribution{}
	}
	return cs, nil
}

// DeleteContribution удаляет пополнение.
func (s *Service) DeleteContribution(ctx context.Context, userID, goalID, id int64) error {
	if err := s.repo.DeleteContribution(ctx, userID, goalID, id); err != nil {
		return mapStorageErr(err)
	}
	return nil
}

func mapStorageErr(err error) error {
	switch {
	case errors.Is(err, storage.ErrGoalNotFound):
		return ErrNotFound
	case errors.Is(err, storage.ErrContributionNotFound):
		return ErrNotFound
	default:
		return err
	}
}

// Заглушка, чтобы пакет компилировался без использования goals-импорта
// (он понадобится в Task 11 для Projection/Simulate).
var _ = goals.ErrUnreachable
```

**ВАЖНО:** убрать последнюю строку-заглушку `var _ = goals.ErrUnreachable` если она создаёт неиспользуемый импорт после Task 11. В этом Task 10 оставить — она держит импорт `goals` до Task 11. Проверить, что импорт не дублируется.

- [ ] **Step 2: Проверить сборку**

Run: `"C:/Program Files/Go/bin/go.exe" build ./...`
Expected: BUILD_OK

- [ ] **Step 3: Commit**

```bash
git add internal/service/goals/goals.go
git commit -m "feat(goals): service Repo interface, CRUD, idempotent AddContribution"
```

---

## Task 11: service/goals — Projection + Simulate

**Files:**
- Modify: `internal/service/goals/goals.go` (append Projection, Simulate)

- [ ] **Step 1: Дописать типы и методы Projection/Simulate в goals.go**

Добавить в конец файла (удалить строку-заглушку `var _ = goals.ErrUnreachable`):
```go
// Projection — результат вычисления статуса цели (BUSINESS_LOGIC ф.5 "С").
type Projection struct {
	Goal             domain.Goal     `json:"goal"`
	EffectiveCurrent domain.Money    `json:"effective_current"` // baseline + Σ contributions
	TargetEffective  domain.Money    `json:"target_effective"`  // цель после инфляции (если применимо)
	Progress         decimal.Decimal `json:"progress"`          // effective / target [0..1+]
	MonthsLeft       int             `json:"months_left"`       // до target_date (0 если нет даты)
	RequiredMonthly  domain.Money    `json:"required_monthly"`  // требуемый взнос для достижения к сроку
	EstimatedMonths  int             `json:"estimated_months"`  // срок при текущем взносе (0 если нет взноса)
	Status           domain.GoalStatus `json:"status"`
	AsOfDate         time.Time       `json:"as_of"`
}

// Compute проекция для цели. Читает Σ contributions → effective → mathcore goals.
func (s *Service) Compute(ctx context.Context, userID, goalID int64) (Projection, error) {
	if userID == 0 || goalID == 0 {
		return Projection{}, errors.New("goals: user_id and id required")
	}
	g, err := s.repo.GetGoal(ctx, userID, goalID)
	if err != nil {
		return Projection{}, mapStorageErr(err)
	}
	sum, err := s.repo.SumContributions(ctx, userID, goalID)
	if err != nil {
		return Projection{}, err
	}
	now := s.now()
	return s.projectWith(g, sum, now, nil), nil
}

// SimulateInput — параметры what-if симуляции. stateless: не требует сохранения.
type SimulateInput struct {
	CurrentAmount       domain.Money
	TargetAmount        domain.Money
	MonthlyContribution *domain.Money
	TargetDate          *time.Time
	ExpectedYield       decimal.Decimal
	Inflation           decimal.Decimal // опционально; 0 = без корректировки
}

// Simulate считает проекцию по гипотезе (stateless). Если extraApplyToGoalID
// задан, можно комбинировать с сохранённой целью — здесь просто обёртка.
func (s *Service) Simulate(ctx context.Context, in SimulateInput) (Projection, error) {
	if !in.TargetAmount.IsPositive() {
		return Projection{}, errors.New("goals: target_amount must be positive")
	}
	now := s.now()
	g := domain.Goal{
		Name: "simulate", TargetAmount: in.TargetAmount, CurrentAmount: in.CurrentAmount,
		MonthlyContribution: in.MonthlyContribution, TargetDate: in.TargetDate,
		ExpectedYield: in.ExpectedYield,
	}
	return s.projectWith(g, domain.Zero, now, &in.Inflation), nil
}

// SimulateSaved комбинирует сохранённую цель с гипотезой (тело приоритетнее).
func (s *Service) SimulateSaved(ctx context.Context, userID, goalID int64, in SimulateInput) (Projection, error) {
	g, err := s.repo.GetGoal(ctx, userID, goalID)
	if err != nil {
		return Projection{}, mapStorageErr(err)
	}
	sum, err := s.repo.SumContributions(ctx, userID, goalID)
	if err != nil {
		return Projection{}, err
	}
	// Наложить гипотезу: ненулевые поля тела приоритетнее.
	if in.TargetAmount.IsPositive() {
		g.TargetAmount = in.TargetAmount
	}
	if in.MonthlyContribution != nil {
		g.MonthlyContribution = in.MonthlyContribution
	}
	if in.TargetDate != nil {
		g.TargetDate = in.TargetDate
	}
	if in.CurrentAmount.IsPositive() {
		g.CurrentAmount = in.CurrentAmount
	}
	inflation := in.Inflation
	return s.projectWith(g, sum, s.now(), &inflation), nil
}

// projectWith — чистая (почти) функция над (goal, sum, now, inflation).
// Inflation nil → без корректировки цели.
func (s *Service) projectWith(g domain.Goal, sum domain.Money, now time.Time, inflation *decimal.Decimal) Projection {
	effective := g.CurrentAmount.Add(sum)
	i := g.ExpectedYield.Div(decimal.NewFromInt(12)) // месячная ставка

	// Достигнута?
	if effective.GreaterThanOrEqual(g.TargetAmount) {
		return Projection{
			Goal: g, EffectiveCurrent: effective, TargetEffective: g.TargetAmount,
			Progress: decimal.NewFromInt(1), Status: domain.StatusGoalAchieved, AsOfDate: now,
		}
	}

	// Цель, скорректированная на инфляцию (если задан target_date).
	targetEff := g.TargetAmount
	if g.TargetDate != nil && inflation != nil && inflation.IsPositive() {
		months := monthsBetween(now, *g.TargetDate)
		if inflated, err := goals.InflateTarget(g.TargetAmount, *inflation, months); err == nil {
			targetEff = domain.FromDecimal(inflated)
		}
	}

	progress := effective.Decimal().Div(targetEff.Decimal())

	// С дедлайном: считаем требуемый взнос.
	if g.TargetDate != nil {
		monthsLeft := monthsBetween(now, *g.TargetDate)
		if monthsLeft <= 0 {
			return Projection{
				Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
				Progress: progress, MonthsLeft: 0, Status: domain.StatusGoalBehind, AsOfDate: now,
			}
		}
		req, err := goals.SolveContribution(effective.Decimal(), targetEff.Decimal(), i, monthsLeft)
		if err != nil {
			return Projection{
				Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
				Progress: progress, MonthsLeft: monthsLeft, Status: domain.StatusGoalBehind, AsOfDate: now,
			}
		}
		reqMoney := domain.FromDecimal(req)
		status := classifyStatus(g.MonthlyContribution, reqMoney)
		return Projection{
			Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
			Progress: progress, MonthsLeft: monthsLeft, RequiredMonthly: reqMoney,
			Status: status, AsOfDate: now,
		}
	}

	// Без дедлайна, но с взносом: считаем срок.
	if g.MonthlyContribution != nil && g.MonthlyContribution.IsPositive() {
		n, err := goals.SolveTerm(effective.Decimal(), targetEff.Decimal(), g.MonthlyContribution.Decimal(), i)
		if err != nil {
			return Projection{
				Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
				Progress: progress, Status: domain.StatusGoalBehind, AsOfDate: now,
			}
		}
		// Округляем вверх до целых месяцев.
		estMonths := int(n.Round(0).IntPart())
		if n.Sub(decimal.NewFromInt(int64(estMonths))).IsPositive() {
			estMonths++
		}
		return Projection{
			Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
			Progress: progress, EstimatedMonths: estMonths,
			Status: domain.StatusGoalOnTrack, AsOfDate: now,
		}
	}

	// Нет дедлайна, нет взноса.
	return Projection{
		Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
		Progress: progress, Status: domain.StatusGoalNoDeadline, AsOfDate: now,
	}
}

// classifyStatus сравнивает заданный взнос с требуемым.
// nil/0 → behind; >= required → on_track; >= required·0.9 → at_risk; иначе behind.
func classifyStatus(monthly *domain.Money, required domain.Money) domain.GoalStatus {
	if monthly == nil || !monthly.IsPositive() {
		if required.IsZero() {
			return domain.StatusGoalOnTrack
		}
		return domain.StatusGoalBehind
	}
	if monthly.GreaterThanOrEqual(required) {
		return domain.StatusGoalOnTrack
	}
	threshold := required.Mul(decimal.NewFromFloat(0.9))
	if monthly.GreaterThanOrEqual(threshold) {
		return domain.StatusGoalAtRisk
	}
	return domain.StatusGoalBehind
}

// monthsBetween — целое число месяцев между двумя датами (не отрицательное).
func monthsBetween(from, to time.Time) int {
	if !to.After(from) {
		return 0
	}
	months := (to.Year()-from.Year())*12 + int(to.Month()-from.Month())
	if !to.AddDate(0, -months, 0).Before(from) {
		// поправка: если день to раньше дня from, месяц не засчитан полностью
	}
	if months < 0 {
		months = 0
	}
	return months
}
```

- [ ] **Step 2: Проверить build + vet**

Run: `"C:/Program Files/Go/bin/go.exe" build ./... && "C:/Program Files/Go/bin/go.exe" vet ./...`
Expected: BUILD_OK / VET_OK

- [ ] **Step 3: Commit**

```bash
git add internal/service/goals/goals.go
git commit -m "feat(goals): Projection + Simulate (stateless) + classifyStatus"
```

---

## Task 12: service/goals — fakeRepo и тесты

**Files:**
- Create: `internal/service/goals/goals_test.go`

- [ ] **Step 1: Написать fakeRepo и тесты**

`internal/service/goals/goals_test.go`:
```go
package goals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
	"github.com/shopspring/decimal"
)

// fakeRepo — in-memory Repo для unit-тестов без БД (стиль budget_test.go).
type fakeRepo struct {
	goals         map[int64]domain.Goal
	contributions map[int64]domain.GoalContribution // keyed by row id
	byClientID    map[string]int64                  // "userID:goalID:contributionID" → row id
	nextGoalID    int64
	nextContrID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		goals:         make(map[int64]domain.Goal),
		contributions: make(map[int64]domain.GoalContribution),
		byClientID:    make(map[string]int64),
	}
}

func (f *fakeRepo) CreateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	f.nextGoalID++
	g.ID = f.nextGoalID
	g.CreatedAt = time.Now()
	g.UpdatedAt = g.CreatedAt
	f.goals[g.ID] = g
	return g, nil
}
func (f *fakeRepo) GetGoal(_ context.Context, userID, id int64) (domain.Goal, error) {
	g, ok := f.goals[id]
	if !ok || g.UserID != userID {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	return g, nil
}
func (f *fakeRepo) ListGoals(_ context.Context, userID int64) ([]domain.Goal, error) {
	var out []domain.Goal
	for _, g := range f.goals {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}
func (f *fakeRepo) UpdateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	ex, ok := f.goals[g.ID]
	if !ok || ex.UserID != g.UserID {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	f.goals[g.ID] = g
	return g, nil
}
func (f *fakeRepo) DeleteGoal(_ context.Context, userID, id int64) error {
	g, ok := f.goals[id]
	if !ok || g.UserID != userID {
		return storage.ErrGoalNotFound
	}
	delete(f.goals, id)
	return nil
}
func (f *fakeRepo) SumContributions(_ context.Context, userID, goalID int64) (domain.Money, error) {
	sum := domain.Zero
	for _, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID {
			sum = sum.Add(c.Amount)
		}
	}
	return sum, nil
}
func (f *fakeRepo) CreateContribution(_ context.Context, c domain.GoalContribution) (domain.GoalContribution, error) {
	key := c.ClientKey()
	if _, exists := f.byClientID[key]; exists {
		return domain.GoalContribution{}, storage.ErrContributionExists
	}
	f.nextContrID++
	c.ID = f.nextContrID
	c.CreatedAt = time.Now()
	f.contributions[c.ID] = c
	f.byClientID[key] = c.ID
	return c, nil
}
func (f *fakeRepo) GetContributionByClientID(_ context.Context, userID, goalID int64, contributionID string) (domain.GoalContribution, error) {
	key := clientKey(userID, goalID, contributionID)
	id, ok := f.byClientID[key]
	if !ok {
		return domain.GoalContribution{}, storage.ErrContributionNotFound
	}
	return f.contributions[id], nil
}
func (f *fakeRepo) ListContributions(_ context.Context, userID, goalID int64) ([]domain.GoalContribution, error) {
	var out []domain.GoalContribution
	for _, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeRepo) DeleteContribution(_ context.Context, userID, goalID, id int64) error {
	c, ok := f.contributions[id]
	if !ok || c.UserID != userID || c.GoalID != goalID {
		return storage.ErrContributionNotFound
	}
	delete(f.contributions, id)
	delete(f.byClientID, c.ClientKey())
	return nil
}

// helpers для ключа идемпотентности.
func clientKey(userID, goalID int64, contributionID string) string {
	return string(userID) + ":" + string(goalID) + ":" + contributionID
}
func (c domain.GoalContribution) ClientKey() string { return clientKey(c.UserID, c.GoalID, c.ContributionID) }
```

**ВАЖНО:** `string(int64)` не работает в Go. Использовать `strconv.FormatInt`. Исправить:
```go
import "strconv"

func clientKey(userID, goalID int64, contributionID string) string {
	return strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(goalID, 10) + ":" + contributionID
}
```
И убрать `"string(userID) + ..."` версию. Метод `ClientKey()` тоже должен использовать `clientKey(...)`.

Также: методы `ClientKey` на `domain.GoalContribution` определить нельзя в package `goals` (это чужой пакет). Убрать метод, использовать функцию `clientKey(c.UserID, c.GoalID, c.ContributionID)` напрямую в `CreateContribution`/`DeleteContribution`:
```go
// В CreateContribution:
key := clientKey(c.UserID, c.GoalID, c.ContributionID)
// В DeleteContribution:
c := f.contributions[id]
delete(f.byClientID, clientKey(c.UserID, c.GoalID, c.ContributionID))
```

Дописать тесты после исправлений:
```go
func svcWithFixedNow(repo Repo, t time.Time) *Service {
	return &Service{repo: repo, now: func() time.Time { return t }}
}

func TestCreate_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	g, err := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "Машина",
		TargetAmount:  domain.MustParseMoney("1000000.00"),
		CurrentAmount: domain.Zero,
		ExpectedYield: decimal.NewFromFloat(0.08),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.ID == 0 {
		t.Errorf("expected non-zero id")
	}
}

func TestCreate_RejectsInvalid(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	cases := []struct {
		name string
		in   CreateInput
	}{
		{"empty name", CreateInput{UserID: 7, TargetAmount: domain.MustParseMoney("1.00")}},
		{"zero target", CreateInput{UserID: 7, Name: "X"}},
		{"neg yield", CreateInput{UserID: 7, Name: "X", TargetAmount: domain.MustParseMoney("1.00"), ExpectedYield: decimal.NewFromInt(-1)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := svc.Create(context.Background(), c.in); err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func TestAddContribution_Idempotent(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	repo.CreateGoal(context.Background(), domain.Goal{UserID: 7, ID: 1, Name: "X", TargetAmount: domain.MustParseMoney("1000.00")})

	in := AddContributionInput{UserID: 7, GoalID: 1, ContributionID: "abc", Amount: domain.MustParseMoney("500.00")}
	c1, dup1, err := svc.AddContribution(context.Background(), in)
	if err != nil || dup1 {
		t.Fatalf("first Add: dup=%v err=%v", dup1, err)
	}
	c2, dup2, err := svc.AddContribution(context.Background(), in)
	if err != nil || !dup2 {
		t.Fatalf("second Add: expected dup=true, got dup=%v err=%v", dup2, err)
	}
	if c1.ID != c2.ID || !c1.Amount.Equal(c2.Amount) {
		t.Errorf("idempotent response mismatch: c1=%+v c2=%+v", c1, c2)
	}
}

func TestCompute_Achieved(t *testing.T) {
	repo := newFakeRepo()
	g, _ := repo.CreateGoal(context.Background(), domain.Goal{
		UserID: 7, Name: "X",
		TargetAmount:  domain.MustParseMoney("1000.00"),
		CurrentAmount: domain.MustParseMoney("1500.00"), // уже больше цели
	})
	svc := NewService(repo)
	proj, err := svc.Compute(context.Background(), 7, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.Status != domain.StatusGoalAchieved {
		t.Errorf("status = %s, want achieved", proj.Status)
	}
}

func TestCompute_WithDeadline_OnTrack(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC) // 12 месяцев
	monthly := domain.MustParseMoney("100000.00")
	g, _ := repo.CreateGoal(context.Background(), domain.Goal{
		UserID: 7, Name: "X",
		TargetAmount:        domain.MustParseMoney("1200000.00"),
		CurrentAmount:       domain.Zero,
		MonthlyContribution: &monthly,
		TargetDate:          &deadline,
		ExpectedYield:       decimal.Zero,
	})
	svc := svcWithFixedNow(repo, now)
	proj, err := svc.Compute(context.Background(), 7, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	// Требуется 100000/мес при 12 мес — задано 100000 → on_track
	if proj.Status != domain.StatusGoalOnTrack {
		t.Errorf("status = %s, want on_track (req=%s)", proj.Status, proj.RequiredMonthly)
	}
	if proj.MonthsLeft != 12 {
		t.Errorf("months_left = %d, want 12", proj.MonthsLeft)
	}
}

func TestSimulate_Stateless(t *testing.T) {
	repo := newFakeRepo() // не используется в Simulate, но требуется для конструктора
	svc := NewService(repo)
	deadline := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	monthly := domain.MustParseMoney("100000.00")
	proj, err := svc.Simulate(context.Background(), SimulateInput{
		TargetAmount:        domain.MustParseMoney("1200000.00"),
		CurrentAmount:       domain.Zero,
		MonthlyContribution: &monthly,
		TargetDate:          &deadline,
		ExpectedYield:       decimal.Zero,
	})
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if proj.Status != domain.StatusGoalOnTrack {
		t.Errorf("status = %s, want on_track", proj.Status)
	}
}

func TestCompute_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	_, err := svc.Compute(context.Background(), 7, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
```

- [ ] **Step 2: Запустить тесты service**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/service/goals/`
Expected: PASS (все тесты)

- [ ] **Step 3: Commit**

```bash
git add internal/service/goals/goals_test.go
git commit -m "test(goals): fakeRepo + service unit tests (CRUD, idempotency, projection, simulate)"
```

---

## Task 13: transport/http/goals.go — handler

**Files:**
- Create: `internal/transport/http/goals.go`

- [ ] **Step 1: Написать handler**

`internal/transport/http/goals.go`:
```go
package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	applog "github.com/RedEye472-afk/FinHelper/internal/log"
	"github.com/RedEye472-afk/FinHelper/internal/service/goals"
)

// GoalHandler wires the goal-tracker REST endpoints (BUSINESS_LOGIC.md ф.5).
type GoalHandler struct {
	svc    *goals.Service
	logger *slog.Logger
}

func NewGoalHandler(svc *goals.Service, logger *slog.Logger) *GoalHandler {
	if svc == nil || logger == nil {
		panic("http: NewGoalHandler requires non-nil deps")
	}
	return &GoalHandler{svc: svc, logger: logger}
}

func (h *GoalHandler) Register(r chi.Router) {
	r.Post("/goals", h.Create)
	r.Get("/goals", h.List)
	r.Get("/goals/{id}", h.Get)
	r.Patch("/goals/{id}", h.Update)
	r.Delete("/goals/{id}", h.Delete)
	r.Get("/goals/{id}/projection", h.Projection)
	r.Post("/goals/{id}/contributions", h.AddContribution)
	r.Get("/goals/{id}/contributions", h.ListContributions)
	r.Delete("/goals/{id}/contributions/{cid}", h.DeleteContribution)
	r.Post("/goals/{id}/simulate", h.SimulateSaved)
	r.Post("/calc/goal", h.Simulate)
}

// ---- response shapes ----

type goalResponse struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	TargetAmount        string `json:"target_amount"`
	CurrentAmount       string `json:"current_amount"`
	MonthlyContribution string `json:"monthly_contribution,omitempty"`
	TargetDate          string `json:"target_date,omitempty"`
	ExpectedYield       string `json:"expected_yield"`
}

type contributionResponse struct {
	ID               int64  `json:"id"`
	GoalID           int64  `json:"goal_id"`
	ContributionID   string `json:"contribution_id"`
	Amount           string `json:"amount"`
	ContributionDate string `json:"contribution_date"`
	Comment          string `json:"comment,omitempty"`
}

type projectionResponse struct {
	Goal             goalResponse `json:"goal"`
	EffectiveCurrent string       `json:"effective_current"`
	TargetEffective  string       `json:"target_effective"`
	Progress         string       `json:"progress"`
	MonthsLeft       int          `json:"months_left"`
	RequiredMonthly  string       `json:"required_monthly"`
	EstimatedMonths  int          `json:"estimated_months"`
	Status           string       `json:"status"`
	AsOfDate         string       `json:"as_of"`
}

func toGoalResponse(g domain.Goal) goalResponse {
	resp := goalResponse{
		ID: g.ID, Name: g.Name,
		TargetAmount:  g.TargetAmount.String(),
		CurrentAmount: g.CurrentAmount.String(),
		ExpectedYield: g.ExpectedYield.String(),
	}
	if g.MonthlyContribution != nil {
		resp.MonthlyContribution = g.MonthlyContribution.String()
	}
	if g.TargetDate != nil {
		resp.TargetDate = g.TargetDate.Format("2006-01-02")
	}
	return resp
}

func toContributionResponse(c domain.GoalContribution) contributionResponse {
	return contributionResponse{
		ID: c.ID, GoalID: c.GoalID, ContributionID: c.ContributionID,
		Amount:           c.Amount.String(),
		ContributionDate: c.ContributionDate.Format("2006-01-02"),
		Comment:          c.Comment,
	}
}

// ---- request shapes ----

type createGoalRequest struct {
	Name                string `json:"name"`
	TargetAmount        string `json:"target_amount"`
	CurrentAmount       string `json:"current_amount"`
	MonthlyContribution string `json:"monthly_contribution"`
	TargetDate          string `json:"target_date"`
	ExpectedYield       string `json:"expected_yield"`
}

type updateGoalRequest struct {
	Name                string `json:"name"`
	TargetAmount        string `json:"target_amount"`
	CurrentAmount       string `json:"current_amount"`
	MonthlyContribution string `json:"monthly_contribution"`
	TargetDate          string `json:"target_date"`
	ExpectedYield       string `json:"expected_yield"`
}

type addContributionRequest struct {
	ContributionID string `json:"contribution_id"`
	Amount         string `json:"amount"`
	Date           string `json:"date"`
	Comment        string `json:"comment"`
}

type simulateRequest struct {
	CurrentAmount       string `json:"current_amount"`
	TargetAmount        string `json:"target_amount"`
	MonthlyContribution string `json:"monthly_contribution"`
	TargetDate          string `json:"target_date"`
	ExpectedYield       string `json:"expected_yield"`
	Inflation           string `json:"inflation"`
}

// parseGoalCommon разбирает общие поля create/update запросов.
// monthlyEmpty/zero → nil (нет регулярного взноса). dateEmpty → nil.
func parseGoalCommon(req createGoalRequest, userID int64, now time.Time) (goals.CreateInput, error) {
	target, err := domain.ParseMoney(req.TargetAmount)
	if err != nil {
		return goals.CreateInput{}, errors.New("goal: invalid target_amount")
	}
	current := domain.Zero
	if req.CurrentAmount != "" {
		c, err := domain.ParseMoney(req.CurrentAmount)
		if err != nil {
			return goals.CreateInput{}, errors.New("goal: invalid current_amount")
		}
		current = c
	}
	in := goals.CreateInput{
		UserID: userID, Name: req.Name, TargetAmount: target, CurrentAmount: current,
	}
	if req.MonthlyContribution != "" {
		m, err := domain.ParseMoney(req.MonthlyContribution)
		if err != nil {
			return goals.CreateInput{}, errors.New("goal: invalid monthly_contribution")
		}
		in.MonthlyContribution = &m
	}
	if req.TargetDate != "" {
		t, err := time.Parse("2006-01-02", req.TargetDate)
		if err != nil {
			return goals.CreateInput{}, errors.New("goal: invalid target_date (use YYYY-MM-DD)")
		}
		in.TargetDate = &t
	}
	if req.ExpectedYield != "" {
		y, err := decimal.NewFromString(req.ExpectedYield)
		if err != nil {
			return goals.CreateInput{}, errors.New("goal: invalid expected_yield")
		}
		in.ExpectedYield = y
	}
	return in, nil
}

// ---- handlers ----

func (h *GoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	var req createGoalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goal.invalid_body", err.Error())
		return
	}
	in, err := parseGoalCommon(req, userID, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "goal.invalid_field", err.Error())
		return
	}
	g, err := h.svc.Create(ctx, in)
	if err != nil {
		h.writeServiceError(w, err, "create")
		return
	}
	applog.Info(ctx, h.logger, "goal created", "goal_id", g.ID)
	writeJSON(w, http.StatusCreated, toGoalResponse(g))
}

func (h *GoalHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	gs, err := h.svc.List(ctx, userID)
	if err != nil {
		h.writeServiceError(w, err, "list")
		return
	}
	out := make([]goalResponse, 0, len(gs))
	for _, g := range gs {
		out = append(out, toGoalResponse(g))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *GoalHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	g, err := h.svc.Get(ctx, userID, id)
	if err != nil {
		h.writeServiceError(w, err, "get")
		return
	}
	writeJSON(w, http.StatusOK, toGoalResponse(g))
}

func (h *GoalHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	var req createGoalRequest // та же shape, что у create
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goal.invalid_body", err.Error())
		return
	}
	in, err := parseGoalCommon(req, userID, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "goal.invalid_field", err.Error())
		return
	}
	g, err := h.svc.Update(ctx, goals.UpdateInput{
		UserID: userID, ID: id, Name: in.Name, TargetAmount: in.TargetAmount,
		CurrentAmount: in.CurrentAmount, MonthlyContribution: in.MonthlyContribution,
		TargetDate: in.TargetDate, ExpectedYield: in.ExpectedYield,
	})
	if err != nil {
		h.writeServiceError(w, err, "update")
		return
	}
	writeJSON(w, http.StatusOK, toGoalResponse(g))
}

func (h *GoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	if err := h.svc.Delete(ctx, userID, id); err != nil {
		h.writeServiceError(w, err, "delete")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *GoalHandler) Projection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	proj, err := h.svc.Compute(ctx, userID, id)
	if err != nil {
		h.writeServiceError(w, err, "projection")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionResponse(proj))
}

func (h *GoalHandler) AddContribution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	goalID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || goalID <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	var req addContributionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "contribution.invalid_body", err.Error())
		return
	}
	amount, err := domain.ParseMoney(req.Amount)
	if err != nil || !amount.IsPositive() {
		writeError(w, http.StatusBadRequest, "contribution.invalid_amount", "amount must be positive")
		return
	}
	var date time.Time
	if req.Date != "" {
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			writeError(w, http.StatusBadRequest, "contribution.invalid_date", "use YYYY-MM-DD")
			return
		}
	}
	c, dup, err := h.svc.AddContribution(ctx, goals.AddContributionInput{
		UserID: userID, GoalID: goalID, ContributionID: req.ContributionID,
		Amount: amount, Date: date, Comment: req.Comment,
	})
	if err != nil {
		h.writeServiceError(w, err, "add_contribution")
		return
	}
	status := http.StatusCreated
	if dup {
		status = http.StatusOK // идемпотентный ответ
	}
	applog.Info(ctx, h.logger, "contribution added", "goal_id", goalID, "duplicate", dup)
	writeJSON(w, status, toContributionResponse(c))
}

func (h *GoalHandler) ListContributions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	goalID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || goalID <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	cs, err := h.svc.ListContributions(ctx, userID, goalID)
	if err != nil {
		h.writeServiceError(w, err, "list_contributions")
		return
	}
	out := make([]contributionResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, toContributionResponse(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *GoalHandler) DeleteContribution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	goalID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || goalID <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	cid, err := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	if err != nil || cid <= 0 {
		writeError(w, http.StatusBadRequest, "contribution.invalid_id", "cid must be a positive integer")
		return
	}
	if err := h.svc.DeleteContribution(ctx, userID, goalID, cid); err != nil {
		h.writeServiceError(w, err, "delete_contribution")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *GoalHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := MustUserID(ctx); !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	proj, err := h.parseAndSimulate(ctx, r, false, 0)
	if err != nil {
		h.writeServiceError(w, err, "simulate")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionResponse(proj))
}

func (h *GoalHandler) SimulateSaved(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := MustUserID(ctx)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	goalID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || goalID <= 0 {
		writeError(w, http.StatusBadRequest, "goal.invalid_id", "id must be a positive integer")
		return
	}
	var req simulateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goal.invalid_body", err.Error())
		return
	}
	in := h.parseSimulateInput(req)
	proj, err := h.svc.SimulateSaved(ctx, userID, goalID, in)
	if err != nil {
		h.writeServiceError(w, err, "simulate_saved")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionResponse(proj))
}

func (h *GoalHandler) parseAndSimulate(ctx context.Context, r *http.Request, _ bool, _ int64) (goals.Projection, error) {
	var req simulateRequest
	if err := decodeJSON(r, &req); err != nil {
		return goals.Projection{}, err
	}
	return h.svc.Simulate(ctx, h.parseSimulateInput(req))
}

func (h *GoalHandler) parseSimulateInput(req simulateRequest) goals.SimulateInput {
	in := goals.SimulateInput{}
	if req.CurrentAmount != "" {
		if c, err := domain.ParseMoney(req.CurrentAmount); err == nil {
			in.CurrentAmount = c
		}
	}
	if req.TargetAmount != "" {
		if t, err := domain.ParseMoney(req.TargetAmount); err == nil {
			in.TargetAmount = t
		}
	}
	if req.MonthlyContribution != "" {
		if m, err := domain.ParseMoney(req.MonthlyContribution); err == nil {
			in.MonthlyContribution = &m
		}
	}
	if req.TargetDate != "" {
		if t, err := time.Parse("2006-01-02", req.TargetDate); err == nil {
			in.TargetDate = &t
		}
	}
	if req.ExpectedYield != "" {
		if y, err := decimal.NewFromString(req.ExpectedYield); err == nil {
			in.ExpectedYield = y
		}
	}
	if req.Inflation != "" {
		if pi, err := decimal.NewFromString(req.Inflation); err == nil {
			in.Inflation = pi
		}
	}
	return in
}

func toProjectionResponse(p goals.Projection) projectionResponse {
	return projectionResponse{
		Goal:             toGoalResponse(p.Goal),
		EffectiveCurrent: p.EffectiveCurrent.String(),
		TargetEffective:  p.TargetEffective.String(),
		Progress:         p.Progress.String(),
		MonthsLeft:       p.MonthsLeft,
		RequiredMonthly:  p.RequiredMonthly.String(),
		EstimatedMonths:  p.EstimatedMonths,
		Status:           string(p.Status),
		AsOfDate:         p.AsOfDate.Format("2006-01-02"),
	}
}

func (h *GoalHandler) writeServiceError(w http.ResponseWriter, err error, op string) {
	switch {
	case errors.Is(err, goals.ErrNotFound):
		writeError(w, http.StatusNotFound, "goal.not_found", "goal not found")
	default:
		h.logger.Error("goals: "+op, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
	}
}
```

**ВАЖНО:** в `parseAndSimulate` параметр `ctx context.Context` требует импорта `context`. Добавить `"context"` в imports. Если `parseAndSimulate` оказывается избыточной обёрткой — упростить: в `Simulate` напрямую декодировать и звать `h.svc.Simulate`. См. шаг рефакторинга ниже.

- [ ] **Step 2: Упростить parseAndSimulate — убрать лишнюю обёртку**

Заменить метод `Simulate` и удалить `parseAndSimulate`:
```go
func (h *GoalHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := MustUserID(ctx); !ok {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
		return
	}
	var req simulateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "goal.invalid_body", err.Error())
		return
	}
	proj, err := h.svc.Simulate(ctx, h.parseSimulateInput(req))
	if err != nil {
		h.writeServiceError(w, err, "simulate")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionResponse(proj))
}
```

После этого импорт `"context"` НЕ нужен — убрать его.

- [ ] **Step 3: Проверить build + vet**

Run: `"C:/Program Files/Go/bin/go.exe" build ./... && "C:/Program Files/Go/bin/go.exe" vet ./...`
Expected: BUILD_OK / VET_OK (без неиспользуемых импортов)

- [ ] **Step 4: Commit**

```bash
git add internal/transport/http/goals.go
git commit -m "feat(goals): HTTP handler — CRUD, projection, contributions, simulate"
```

---

## Task 14: transport/http — wiring в router.go

**Files:**
- Modify: `internal/transport/http/router.go`

- [ ] **Step 1: Добавить поле Goals в Deps и монтаж**

В `router.go`:

1. Добавить импорт:
```go
"github.com/RedEye472-afk/FinHelper/internal/service/goals"
```
(в алфавитном порядке между `dashboard` и `operations`)

2. Добавить поле в `Deps` (после `Budget *budget.Service`):
```go
	// Goals is the savings-goal tracker service for ф.5. nil = skip mounting
	// /goals routes.
	Goals *goals.Service
```

3. В `r.Group(...)` добавить (после блока `if deps.Budget != nil`):
```go
			if deps.Goals != nil {
				NewGoalHandler(deps.Goals, deps.Logger).Register(r)
			}
```

- [ ] **Step 2: Проверить build**

Run: `"C:/Program Files/Go/bin/go.exe" build ./...`
Expected: BUILD_OK

- [ ] **Step 3: Commit**

```bash
git add internal/transport/http/router.go
git commit -m "feat(goals): wire Goals service into router Deps"
```

---

## Task 15: cmd/server/main.go — wiring

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Добавить сервис целей и передать в Deps**

В `main.go`:

1. Добавить импорт:
```go
"github.com/RedEye472-afk/FinHelper/internal/service/goals"
```

2. После `budgetSvc := budget.NewService(pool)` добавить:
```go
		// Goals service (BUSINESS_LOGIC ф.5) — savings-goal tracker.
		goalsSvc := goals.NewService(pool)
```

3. В вызове `transporthttp.NewRouter(transporthttp.Deps{...})` добавить поле:
```go
			Goals:            goalsSvc,
```

- [ ] **Step 2: Полная проверка build/vet/test**

Run: `"C:/Program Files/Go/bin/go.exe" build ./... && "C:/Program Files/Go/bin/go.exe" vet ./...`
Expected: BUILD_OK / VET_OK

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(goals): wire goals service into main"
```

---

## Task 16: transport/http — HTTP-тесты

**Files:**
- Create: `internal/transport/http/goals_test.go`

- [ ] **Step 1: Написать HTTP-тесты (fakeRepo + httptest, стиль budgets_test.go)**

`internal/transport/http/goals_test.go`:
```go
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/service/goals"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// goalFakeRepo — минимальный in-memory Repo для HTTP-тестов.
type goalFakeRepo struct {
	goal          domain.Goal
	contributions map[string]domain.GoalContribution // by contributionID
	nextContrID   int64
}

func newGoalFakeRepo() *goalFakeRepo {
	return &goalFakeRepo{contributions: make(map[string]domain.GoalContribution)}
}

func (f *goalFakeRepo) CreateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	g.ID = 1
	f.goal = g
	return g, nil
}
func (f *goalFakeRepo) GetGoal(_ context.Context, userID, id int64) (domain.Goal, error) {
	if f.goal.ID == 0 || f.goal.UserID != userID || f.goal.ID != id {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	return f.goal, nil
}
func (f *goalFakeRepo) ListGoals(_ context.Context, userID int64) ([]domain.Goal, error) {
	if f.goal.UserID == userID {
		return []domain.Goal{f.goal}, nil
	}
	return []domain.Goal{}, nil
}
func (f *goalFakeRepo) UpdateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	if f.goal.ID != g.ID {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	f.goal = g
	return g, nil
}
func (f *goalFakeRepo) DeleteGoal(_ context.Context, _, _ int64) error {
	f.goal = domain.Goal{}
	return nil
}
func (f *goalFakeRepo) SumContributions(_ context.Context, _, _ int64) (domain.Money, error) {
	sum := domain.Zero
	for _, c := range f.contributions {
		sum = sum.Add(c.Amount)
	}
	return sum, nil
}
func (f *goalFakeRepo) CreateContribution(_ context.Context, c domain.GoalContribution) (domain.GoalContribution, error) {
	if _, exists := f.contributions[c.ContributionID]; exists {
		return domain.GoalContribution{}, storage.ErrContributionExists
	}
	f.nextContrID++
	c.ID = f.nextContrID
	f.contributions[c.ContributionID] = c
	return c, nil
}
func (f *goalFakeRepo) GetContributionByClientID(_ context.Context, _, _ int64, contributionID string) (domain.GoalContribution, error) {
	c, ok := f.contributions[contributionID]
	if !ok {
		return domain.GoalContribution{}, storage.ErrContributionNotFound
	}
	return c, nil
}
func (f *goalFakeRepo) ListContributions(_ context.Context, _, _ int64) ([]domain.GoalContribution, error) {
	out := make([]domain.GoalContribution, 0, len(f.contributions))
	for _, c := range f.contributions {
		out = append(out, c)
	}
	return out, nil
}
func (f *goalFakeRepo) DeleteContribution(_ context.Context, _, _, _ int64) error {
	return nil
}

func newGoalTestEnv(t *testing.T) (*httptest.Server, *auth.JWTIssuer, *goalFakeRepo) {
	t.Helper()
	issuer, err := auth.NewJWTIssuer(
		"access-test-secret-must-be-32+-chars-long",
		"refresh-test-secret-must-be-32+-chars-long",
		15*time.Minute, time.Hour,
	)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	repo := newGoalFakeRepo()
	svc := goals.NewService(repo)
	mw := NewAuthMiddleware(issuer, logger)
	h := NewGoalHandler(svc, logger)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(mw.Wrap)
		h.Register(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, issuer, repo
}

func doGoalReq(t *testing.T, method, url, token string, body any) (*http.Response, []byte) {
	t.Helper()
	var req *http.Request
	if body != nil {
		buf, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, url, bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp.Body)
	return resp, out.Bytes()
}

func TestGoal_Create_Success(t *testing.T) {
	srv, issuer, repo := newGoalTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")
	resp, body := doGoalReq(t, http.MethodPost, srv.URL+"/goals", tok,
		map[string]any{"name": "Машина", "target_amount": "1000000.00", "expected_yield": "0.08"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	if repo.goal.Name != "Машина" {
		t.Errorf("repo name = %s", repo.goal.Name)
	}
}

func TestGoal_Create_RejectsInvalid(t *testing.T) {
	srv, issuer, _ := newGoalTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")
	resp, _ := doGoalReq(t, http.MethodPost, srv.URL+"/goals", tok,
		map[string]any{"name": "", "target_amount": "0"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestGoal_Unauthorized(t *testing.T) {
	srv, _, _ := newGoalTestEnv(t)
	resp, _ := doGoalReq(t, http.MethodGet, srv.URL+"/goals", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGoal_Projection_Achieved(t *testing.T) {
	srv, issuer, repo := newGoalTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")
	repo.goal = domain.Goal{
		ID: 1, UserID: 7, Name: "X",
		TargetAmount:  domain.MustParseMoney("1000.00"),
		CurrentAmount: domain.MustParseMoney("1500.00"),
	}
	resp, body := doGoalReq(t, http.MethodGet, srv.URL+"/goals/1/projection", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var out projectionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Status != string(domain.StatusGoalAchieved) {
		t.Errorf("status = %s, want achieved", out.Status)
	}
}

func TestGoal_AddContribution_Idempotent(t *testing.T) {
	srv, issuer, repo := newGoalTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")
	repo.goal = domain.Goal{
		ID: 1, UserID: 7, Name: "X", TargetAmount: domain.MustParseMoney("1000.00"),
	}
	body := map[string]any{"contribution_id": "abc", "amount": "500.00", "date": "2026-06-01"}
	resp1, _ := doGoalReq(t, http.MethodPost, srv.URL+"/goals/1/contributions", tok, body)
	if resp1.StatusCode != http.StatusCreated {
		t.Errorf("first: status = %d, want 201", resp1.StatusCode)
	}
	resp2, _ := doGoalReq(t, http.MethodPost, srv.URL+"/goals/1/contributions", tok, body)
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("second (idempotent): status = %d, want 200", resp2.StatusCode)
	}
}

func TestGoal_CalcGoal_Stateless(t *testing.T) {
	srv, issuer, _ := newGoalTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")
	resp, body := doGoalReq(t, http.MethodPost, srv.URL+"/calc/goal", tok,
		map[string]any{
			"target_amount": "1200000.00", "current_amount": "0",
			"monthly_contribution": "100000.00", "target_date": "2027-06-01", "expected_yield": "0",
		})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var out projectionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Status != string(domain.StatusGoalOnTrack) {
		t.Errorf("status = %s, want on_track", out.Status)
	}
}
```

- [ ] **Step 2: Запустить все тесты**

Run: `"C:/Program Files/Go/bin/go.exe" test ./...`
Expected: PASS (все пакеты зелёные)

- [ ] **Step 3: Commit**

```bash
git add internal/transport/http/goals_test.go
git commit -m "test(goals): HTTP integration tests via httptest + fakeRepo"
```

---

## Task 17: dashboard.go — прибавить Σ contributions

**Files:**
- Modify: `internal/storage/dashboard.go` (GoalProgresses query)

- [ ] **Step 1: Прочитать текущий GoalProgresses и обновить SQL**

Найти функцию `GoalProgresses` в `internal/storage/dashboard.go`. Текущий SQL
выбирает `(current_amount, target_amount)` из `goals`. Нужно прибавить
`COALESCE(SUM(goal_contributions.amount), 0)` через LEFT JOIN.

Заменить запрос (адаптировать к фактической структуре, сохранить псевдонимы):
```sql
SELECT g.id, g.name,
       g.current_amount + COALESCE(SUM(gc.amount), 0) AS current_amount,
       g.target_amount,
       CASE WHEN g.target_amount = 0 THEN 0
            ELSE (g.current_amount + COALESCE(SUM(gc.amount), 0)) / g.target_amount
       END AS progress
FROM goals g
LEFT JOIN goal_contributions gc ON gc.goal_id = g.id AND gc.user_id = g.user_id
WHERE g.user_id = $1 AND g.deleted_at IS NULL
GROUP BY g.id, g.name, g.current_amount, g.target_amount
```

**Важно:** перед заменой прочитать фактический SQL в файле и сохранить его
псевдонимы/порядок колонок — scan-код в `dashboard.go` зависит от порядка.
Адаптировать SQL так, чтобы порядок и имена возвращаемых колонок совпали с
существующими. Если текущий код сканирует `progress` как decimal через
`decimalScanner` — оставить как есть.

- [ ] **Step 2: Запустить существующие тесты dashboard — убедиться что не сломано**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/storage/`
Expected: PASS (тесты dashboard_test.go используют sqlmock — обновить
ожидаемые SQL-паттерны в тестах, если они матчат старый запрос без JOIN)

- [ ] **Step 3: Обновить sqlmock-ожидания в dashboard_test.go**

Если тесты `TestGoalProgresses*` падают из-за изменённого SQL — обновить
`mock.ExpectQuery(q(...))` паттерны так, чтобы они матчили новый запрос
(добавить `LEFT JOIN goal_contributions`). Сами тест-данные остаются те же
(при отсутствии contributions SUM = 0, результат идентичный).

- [ ] **Step 4: Запустить все storage-тесты**

Run: `"C:/Program Files/Go/bin/go.exe" test ./internal/storage/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/storage/dashboard.go internal/storage/dashboard_test.go
git commit -m "feat(goals): dashboard GoalProgresses includes contributions sum (hybrid model)"
```

---

## Task 18: Полная верификация + PROGRESS.md

**Files:**
- Modify: `PROGRESS.md`

- [ ] **Step 1: Полная сборка + vet + тесты всего проекта**

Run:
```bash
cd C:/Users/user/ZCodeProject/FinHelper/backend
"C:/Program Files/Go/bin/go.exe" build ./...
"C:/Program Files/Go/bin/go.exe" vet ./...
"C:/Program Files/Go/bin/go.exe" test ./...
```
Expected: BUILD_OK / VET_OK / все пакеты зелёные

- [ ] **Step 2: Smoke-тест бинарника (без БД)**

Run (в git bash, как в PROGRESS.md):
```bash
JWT_ACCESS_SECRET=$(printf 'a%.0s' {1..40}) \
JWT_REFRESH_SECRET=$(printf 'b%.0s' {1..40}) \
USER_HASH_SALT=salt HTTP_ADDR=:18080 \
./bin/finhelper.exe &
curl -s http://localhost:18080/healthz   # → {"status":"ok"}
curl -s -o /dev/null -w "%{http_code}" http://localhost:18080/api/v1/goals  # → 404 (нет БД → API не смонтирован)
```
Expected: `/healthz` → 200; `/api/v1/goals` → 404 (graceful без БД)
Остановить сервер после проверки.

- [ ] **Step 3: Обновить PROGRESS.md — добавить секцию "Фича 5"**

Добавить новую секцию после "✅ Этап 3 / Фича 4" (перед "✅ ЭТАП 3 ЗАКРЫТ"),
следуя формату существующих секций (Что сделано, Файлы, Верификация, Хорошо,
Плохо, КРИТИЧЕСКОЕ РЕШЕНИЕ). Краткое содержание:

```markdown
## ✅ Этап 4 / Фича 5 — Трекер целей (ВЫПОЛНЕН + ВЕРИФИЦИРОВАН)

**Цель:** BUSINESS_LOGIC.md ф.5 — трекер целей с регулярными взносами,
внеплановыми пополнениями, проекцией статуса и what-if симуляцией — достигнута.

**Файлы созданы:**
- internal/mathcore/goals/{doc,sinkingfund,sinkingfund_test}.go — sinking-fund
  формулы (SolveFutureValue/SolveContribution/SolveTerm/InflateTarget), strictly
  decimal, без новых float64-bridges
- internal/domain/goal.go — Goal, GoalContribution, GoalStatus, ValidateGoal
- migrations/0003_goals_contributions.sql — журнал пополнений + идемпотентность
- internal/storage/goals.go — CRUD goals + contributions, SumContributions
- internal/service/goals/goals.go — Service + Repo interface, Projection, Simulate
- internal/transport/http/goals.go — REST handler (CRUD + projection + contributions + simulate)

**Верификация (дата):** BUILD_OK / VET_OK / `go test ./...` → все пакеты зелёные
- mathcore/goals: N тестов (golden + edge-cases)
- storage: N sqlmock-тестов
- service/goals: N тестов (CRUD, идемпотентность, projection, simulate)
- transport/http: N интеграционных тестов

**Хорошо:**
- Гибридная модель current_amount: baseline + Σ contributions, self-healing
- Идемпотентность пополнений по contribution_id (консистентно с ф.1)
- Stateless Simulate для фичи 12 «Что если» — фундамент заложен
- 0 новых float64-bridges — sinking fund решается аналитически

**Плохо / тех. долг:**
- E2E через реальный Postgres по-прежнему отложен
- Вначале PROGRESS называл ф.5 «переиспользует credit» — по факту потребовался
  новый mathcore/goals (зафиксировано в spec)

**КРИТИЧЕСКОЕ РЕШЕНИЕ (зафиксировано):**
effective_current = goals.current_amount + Σ goal_contributions. Projection
всегда читает гибридную сумму, не только current_amount. Прямой PATCH
current_amount разрешён — это baseline «стартового капитала».
```

- [ ] **Step 4: Commit**

```bash
git add PROGRESS.md
git commit -m "docs(этап 4): PROGRESS.md — фича 5 (трекер целей) завершена"
```

---

## Self-Review (выполнено автором плана)

**1. Spec coverage:**
- §2 архитектура (mathcore/goals + service/goals + storage/goals + http/goals) → Tasks 1-17 ✅
- §3.1 миграция goal_contributions → Task 7 ✅
- §3.2 гибридная модель → Tasks 8 (SumContributions) + 11 (projectWith) + 17 (dashboard) ✅
- §3.3 domain типы → Task 6 ✅
- §4.1-4.2 mathcore формулы + edge-cases → Tasks 2-5 ✅
- §4.4 Projection алгоритм → Task 11 ✅
- §5 REST API (все 11 эндпоинтов) → Tasks 13-15 ✅
- §5 идемпотентность в стиле ф.1 → Task 10 (AddContribution) + 9 (storage) ✅
- §6 тестирование (3 уровня) → Tasks 2-5 (mathcore), 9 (storage), 12 (service), 16 (http) ✅
- §6 DoD (build/vet/test + smoke) → Task 18 ✅
- dashboard интеграция → Task 17 ✅

**2. Placeholder scan:** Несколько мест в плане честно отмечены
«пересчитать эталон» (Task 2) и «адаптировать к фактическому SQL» (Task 17) —
это не TODO-дыры, а необходимость свериться с кодом в момент реализации
(в стиле принципа проекта «doc↔код не должен расходиться»). Конкретные числа
для golden-тестов должны браться из калькулятора/Excel, не выдумываться.

**3. Type consistency:**
- `goals.CreateInput` / `UpdateInput` / `AddContributionInput` / `SimulateInput` — имена консистентны между service и http
- `domain.GoalStatus` константы `StatusGoalOnTrack` etc. — используется и в service.classifyStatus, и в http.toProjectionResponse
- `Repo` interface в service совпадает с методами `*storage.Pool` (Create/Get/List/Update/Delete Goal + 4 contribution метода + SumContributions)
- `decimal.Decimal` для yield/inflation, `domain.Money` для сумм — консистентно с domain.Money

План готов.
