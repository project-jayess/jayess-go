package tooling

import "time"

type PhaseMeasurement struct {
	Name     string
	Duration time.Duration
}

type PhaseFunc func() error

func MeasurePhases(phases map[string]PhaseFunc, order []string) ([]PhaseMeasurement, error) {
	measurements := make([]PhaseMeasurement, 0, len(order))
	for _, name := range order {
		phase := phases[name]
		start := time.Now()
		if phase != nil {
			if err := phase(); err != nil {
				return measurements, err
			}
		}
		measurements = append(measurements, PhaseMeasurement{Name: name, Duration: time.Since(start)})
	}
	return measurements, nil
}
