package api

import (
	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/waker"
)

// doctorWakerAdapter adapts doctor.Service to waker.DoctorRunner (DX13).
type doctorWakerAdapter struct {
	svc *doctor.Service
}

func (a doctorWakerAdapter) RunSuite(suite string) (*waker.DoctorReport, error) {
	if a.svc == nil {
		return nil, waker.ErrDoctorUnavailable
	}
	rep, err := a.svc.RunSuite(suite)
	if err != nil {
		return nil, err
	}
	out := &waker.DoctorReport{}
	if rep == nil {
		return out, nil
	}
	for _, c := range rep.Results {
		out.Cases = append(out.Cases, waker.DoctorCaseResult{ID: c.ID, Status: c.Status})
	}
	return out, nil
}
