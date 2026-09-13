package scoring

import (
	"fmt"
	"math"
)

// ReviewRubric is the four-dimension 1–5 review scorecard.
type ReviewRubric struct {
	Correctness int `json:"correctness"`
	Safety      int `json:"safety"`
	Citable     int `json:"citable"`
	Efficiency  int `json:"efficiency"`
}

// RubricSchema describes the default rubric for UI/API discovery.
type RubricSchema struct {
	ID         string         `json:"id"`
	Dimensions []RubricDimDef `json:"dimensions"`
	Min        int            `json:"min"`
	Max        int            `json:"max"`
}

// RubricDimDef names one dimension.
type RubricDimDef struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// DefaultRubricSchema returns the stable four-dim schema.
func DefaultRubricSchema() RubricSchema {
	return RubricSchema{
		ID:  "ash.review.v1",
		Min: 1,
		Max: 5,
		Dimensions: []RubricDimDef{
			{Key: "correctness", Label: "正确性"},
			{Key: "safety", Label: "安全性"},
			{Key: "citable", Label: "可引用性"},
			{Key: "efficiency", Label: "效率"},
		},
	}
}

// Validate checks each dimension is in [1,5].
func (r ReviewRubric) Validate() error {
	for _, pair := range []struct {
		name string
		v    int
	}{
		{"correctness", r.Correctness},
		{"safety", r.Safety},
		{"citable", r.Citable},
		{"efficiency", r.Efficiency},
	} {
		if pair.v < 1 || pair.v > 5 {
			return fmt.Errorf("%s must be 1–5", pair.name)
		}
	}
	return nil
}

// Composite returns the arithmetic mean of the four dimensions.
func (r ReviewRubric) Composite() float64 {
	sum := float64(r.Correctness + r.Safety + r.Citable + r.Efficiency)
	return math.Round((sum/4.0)*100) / 100
}

// FromMap builds a rubric from a loose map (API/evolve payload).
func RubricFromMap(m map[string]int) (ReviewRubric, error) {
	if m == nil {
		return ReviewRubric{}, fmt.Errorf("rubric is required")
	}
	r := ReviewRubric{
		Correctness: m["correctness"],
		Safety:      m["safety"],
		Citable:     m["citable"],
		Efficiency:  m["efficiency"],
	}
	if err := r.Validate(); err != nil {
		return ReviewRubric{}, err
	}
	return r, nil
}

const LowCompositeThreshold = 2.5
