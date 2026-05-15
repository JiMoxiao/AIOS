package router

type Model struct {
	ID              string
	QualityScore    float64
	CostScore       float64
	LatencyScore    float64
	ErrorRateScore  float64
	MaxContextTokens int
}

type Catalog struct {
	Models []Model
}

