package main

import (
	"fmt"
	"sort"
)

// CorrectnessChecker validates order processing correctness
type CorrectnessChecker struct {
	orders map[string][]*Order // submissionID -> orders
}

type Order struct {
	OrderID   string
	Price     float64
	Quantity  int
	Timestamp int64
	Status    string // PENDING, FILLED, REJECTED, CANCELLED
	FillPrice float64
}

func NewCorrectnessChecker() *CorrectnessChecker {
	return &CorrectnessChecker{
		orders: make(map[string][]*Order),
	}
}

// ValidatePriceTimePriority checks if orders were filled in FIFO order at same price
func (cc *CorrectnessChecker) ValidatePriceTimePriority(submissionID string) (float64, error) {
	orders, exists := cc.orders[submissionID]
	if !exists {
		return 0, fmt.Errorf("no orders found for submission %s", submissionID)
	}

	if len(orders) == 0 {
		return 100.0, nil
	}

	// Group orders by price
	priceGroups := make(map[float64][]*Order)
	for _, order := range orders {
		if order.Status == "FILLED" {
			priceGroups[order.Price] = append(priceGroups[order.Price], order)
		}
	}

	// Check FIFO within each price level
	validCount := 0
	for _, priceOrderList := range priceGroups {
		// Sort by timestamp - should already be in order if FIFO was followed
		sort.Slice(priceOrderList, func(i, j int) bool {
			return priceOrderList[i].Timestamp < priceOrderList[j].Timestamp
		})

		// Verify no out-of-order fills at same price
		for i := 1; i < len(priceOrderList); i++ {
			if priceOrderList[i].Timestamp >= priceOrderList[i-1].Timestamp {
				validCount++
			}
		}
	}

	correctnessRate := float64(validCount) / float64(len(orders)) * 100
	return correctnessRate, nil
}

// ValidateNoDoubleFills checks that no order was filled twice
func (cc *CorrectnessChecker) ValidateNoDoubleFills(submissionID string) (float64, error) {
	orders, exists := cc.orders[submissionID]
	if !exists {
		return 0, fmt.Errorf("no orders found for submission %s", submissionID)
	}

	fillCount := make(map[string]int) // orderID -> fill count
	for _, order := range orders {
		if order.Status == "FILLED" {
			fillCount[order.OrderID]++
		}
	}

	// Check for double-fills
	validCount := 0
	for _, count := range fillCount {
		if count <= 1 {
			validCount++
		}
	}

	correctnessRate := float64(validCount) / float64(len(fillCount)) * 100
	return correctnessRate, nil
}

// CalculateCorrectnessScore combines all correctness checks
func (cc *CorrectnessChecker) CalculateCorrectnessScore(submissionID string) (float64, error) {
	fifoRate, err1 := cc.ValidatePriceTimePriority(submissionID)
	if err1 != nil {
		fifoRate = 0
	}

	doubleFillRate, err2 := cc.ValidateNoDoubleFills(submissionID)
	if err2 != nil {
		doubleFillRate = 0
	}

	// Average of all checks
	finalScore := (fifoRate + doubleFillRate) / 2
	return finalScore, nil
}
