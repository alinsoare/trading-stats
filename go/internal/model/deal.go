// Package model defines the core data structures shared across the app:
// raw deal rows, rebuilt closed positions, and aggregated KPI summaries.
package model

import "time"

// Deal is one normalized row from a deals_<login>.csv export.
type Deal struct {
	Ticket     int64
	Time       time.Time
	HasTime    bool
	Type       int
	Entry      int
	Symbol     string
	Volume     float64
	Price      float64
	Commission float64
	Swap       float64
	Profit     float64
	Magic      int64
	Comment    string
	PositionID int64
	Reason     int
	Login      int64
	Server     string
	IsNonTrade bool

	AccountLabel string
	SourceFile   string
}

// ClosedPosition is one rebuilt round-trip (or single-ticket fallback).
type ClosedPosition struct {
	AccountLabel  string
	PosKey        string
	NetPnL        float64
	ExitTime      time.Time
	HasExit       bool
	Symbol        string
	Magic         int64
	CommentSample string
	NLegs         int
}

// KPI holds the aggregated statistics for a set of closed positions.
type KPI struct {
	Label        string
	Trades       int
	Wins         int
	Losses       int
	Breakeven    int
	WinRate      float64
	NetPnL       float64
	GrossWin     float64
	GrossLoss    float64
	ProfitFactor float64
	Expectancy   float64
	AvgWin       float64
	AvgLoss      float64
	Payoff       float64
	MaxDDCumPnL  float64
	BestTrade    float64
	WorstTrade   float64
}
