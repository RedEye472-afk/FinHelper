// Package deposit implements pure-math formulas for a deposit (вклад) calculator.
//
// Design principle (CLAUDE.md §1 "Детерминизм"): all monetary calculations use
// decimal.Decimal — never float64. Rounding is to scale 2 (kopecks) using
// RoundHalfUp, which for the non-negative values in deposit math is equivalent
// to ROUND_HALF_AWAY_FROM_ZERO.
//
// Coverage:
//   - Simple interest  (CapMaturity):   S = P * (1 + i * t), t = months/12
//   - Compound interest (monthly/quarterly/annual cap): S = P * (1 + i/m)^(m*t)
//   - Effective annual rate: i_eff = (1 + i/m)^m - 1
//   - Fisher real return:  r_real = (1 + r_nom)/(1 + pi) - 1
//   - Monthly projection for charts
//
// Sources: Копнова Г.П. "Финансовая математика" Гл. 1-2; MATH_FORMULAS.md §1.1–1.4.
package deposit
