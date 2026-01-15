package platform

import (
	"math"
	"strings"

	"github.com/tuusuario/dev-env-snapshots/internal/core"
)

// WindowMatcher maneja el matching inteligente de ventanas
type WindowMatcher struct {
	// Configuración de scoring
	ExactTitleScore   int
	PartialTitleScore int
	SameAppScore      int
	SameSizeScore     int
	MinimumScore      int
}

// DefaultMatcher retorna un matcher con configuración por defecto
func DefaultMatcher() *WindowMatcher {
	return &WindowMatcher{
		ExactTitleScore:   100,
		PartialTitleScore: 50,
		SameAppScore:      50,
		SameSizeScore:     10,
		MinimumScore:      60, // Threshold mínimo para considerar match
	}
}

// MatchResult representa el resultado de un matching
type MatchResult struct {
	Window core.Window
	Score  int
}

// FindBestMatch encuentra la mejor ventana candidata para restaurar
func (m *WindowMatcher) FindBestMatch(target core.Window, candidates []core.Window) *MatchResult {
	var bestMatch *MatchResult

	for _, candidate := range candidates {
		score := m.calculateScore(target, candidate)

		if score >= m.MinimumScore {
			if bestMatch == nil || score > bestMatch.Score {
				bestMatch = &MatchResult{
					Window: candidate,
					Score:  score,
				}
			}
		}
	}

	return bestMatch
}

// calculateScore calcula el score de similitud entre dos ventanas
func (m *WindowMatcher) calculateScore(target, candidate core.Window) int {
	score := 0

	// 1. Title matching (más importante)
	score += m.scoreTitleMatch(target.WindowTitle, candidate.WindowTitle)

	// 2. App name matching
	if target.AppName == candidate.AppName {
		score += m.SameAppScore
	}

	// 3. Size similarity (menos importante pero útil)
	if m.isSimilarSize(target, candidate) {
		score += m.SameSizeScore
	}

	return score
}

// scoreTitleMatch calcula score basado en similitud de títulos
func (m *WindowMatcher) scoreTitleMatch(target, candidate string) int {
	// Exact match
	if target == candidate {
		return m.ExactTitleScore
	}

	// Normalize for comparison
	targetLower := strings.ToLower(target)
	candidateLower := strings.ToLower(candidate)

	// Exact match (case-insensitive)
	if targetLower == candidateLower {
		return m.ExactTitleScore
	}

	// Partial match - candidate contiene target
	if strings.Contains(candidateLower, targetLower) {
		return m.PartialTitleScore
	}

	// Partial match - target contiene candidate
	if strings.Contains(targetLower, candidateLower) {
		return m.PartialTitleScore
	}

	// Fuzzy matching usando Jaccard similarity
	similarity := m.stringSimilarity(targetLower, candidateLower)
	if similarity > 0.7 { // 70% similar
		return int(float64(m.PartialTitleScore) * similarity)
	}

	// Token-based matching (útil para títulos como "file.go - Project - VSCode")
	targetTokens := strings.Fields(target)
	candidateTokens := strings.Fields(candidate)

	commonTokens := m.countCommonTokens(targetTokens, candidateTokens)
	if commonTokens > 0 {
		tokenScore := (commonTokens * m.PartialTitleScore) / len(targetTokens)
		return tokenScore
	}

	return 0
}

// stringSimilarity calcula similitud entre strings (0.0 a 1.0)
// Implementación simple usando Jaccard similarity
func (m *WindowMatcher) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Convertir a sets de caracteres
	set1 := make(map[rune]bool)
	set2 := make(map[rune]bool)

	for _, c := range s1 {
		set1[c] = true
	}
	for _, c := range s2 {
		set2[c] = true
	}

	// Calcular intersección
	intersection := 0
	for c := range set1 {
		if set2[c] {
			intersection++
		}
	}

	// Calcular unión
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// countCommonTokens cuenta tokens comunes entre dos listas
func (m *WindowMatcher) countCommonTokens(tokens1, tokens2 []string) int {
	set := make(map[string]bool)
	for _, t := range tokens1 {
		set[strings.ToLower(t)] = true
	}

	count := 0
	for _, t := range tokens2 {
		if set[strings.ToLower(t)] {
			count++
		}
	}
	return count
}

// isSimilarSize verifica si dos ventanas tienen tamaño similar
func (m *WindowMatcher) isSimilarSize(w1, w2 core.Window) bool {
	// Tolerancia del 10%
	tolerance := 0.1

	widthDiff := math.Abs(float64(w1.Width-w2.Width)) / float64(w1.Width)
	heightDiff := math.Abs(float64(w1.Height-w2.Height)) / float64(w1.Height)

	return widthDiff <= tolerance && heightDiff <= tolerance
}

// MatchWindows encuentra matches para múltiples ventanas
func (m *WindowMatcher) MatchWindows(targets []core.Window, candidates []core.Window) map[string]*MatchResult {
	results := make(map[string]*MatchResult)

	// Crear una copia de candidates para ir marcando las ya usadas
	availableCandidates := make([]core.Window, len(candidates))
	copy(availableCandidates, candidates)

	for _, target := range targets {
		match := m.FindBestMatch(target, availableCandidates)
		if match != nil {
			// Usar título como key (podría ser ID en el futuro)
			results[target.WindowTitle] = match

			// Remover el candidato usado para evitar matches duplicados
			availableCandidates = m.removeWindow(availableCandidates, match.Window)
		}
	}

	return results
}

// removeWindow remueve una ventana de la lista
func (m *WindowMatcher) removeWindow(windows []core.Window, toRemove core.Window) []core.Window {
	result := make([]core.Window, 0, len(windows))
	for _, w := range windows {
		if w.WindowTitle != toRemove.WindowTitle || w.AppName != toRemove.AppName {
			result = append(result, w)
		}
	}
	return result
}
