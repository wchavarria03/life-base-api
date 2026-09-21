package services

import (
	"context"
	"fmt"
	"strings"

	"life-base-api/app/internal/models"
)

func NewClassificationService(rules ClassificationRuleRepository) *ClassificationService {
	return &ClassificationService{rules: rules}
}

// Apply fetches classification rules and applies them to the transactions, returning
// the updated slice. Rules are matched by bank name, code, and description pattern;
// the highest-priority matching rule wins.
func (s *ClassificationService) Apply(ctx context.Context, bankName string, txs []models.Transaction) ([]models.Transaction, error) {
	allRules, err := s.rules.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch classification rules: %w", err)
	}
	if len(allRules) == 0 {
		return txs, nil
	}

	result := make([]models.Transaction, len(txs))
	copy(result, txs)

	for i, tx := range result {
		best := findBestRule(allRules, bankName, tx)
		if best == nil {
			continue
		}
		if best.TypeOverride != nil {
			result[i].Type = *best.TypeOverride
		}
	}

	return result, nil
}

func findBestRule(rules []models.ClassificationRule, bankName string, tx models.Transaction) *models.ClassificationRule {
	var best *models.ClassificationRule
	for i, r := range rules {
		if r.BankName != nil && *r.BankName != bankName {
			continue
		}
		if r.Code != nil && *r.Code != tx.Code {
			continue
		}
		if r.DescriptionPattern != nil && !strings.Contains(
			strings.ToUpper(tx.Description),
			strings.ToUpper(*r.DescriptionPattern),
		) {
			continue
		}
		if best == nil || r.Priority > best.Priority {
			best = &rules[i]
		}
	}
	return best
}
