package internal_xray

func (r *Rule) stale(now int64) bool {
	return r.MatchedRequests != 0 && now >= r.Reservoir.RefreshedAt+r.Reservoir.interval
}

// snapshot takes a snapshot of the sampling statistics counters, returning
// samplingStatisticsDocument. It also resets statistics counters.
func (r *Rule) snapshot() *samplingStatisticsDocument {
	clock := &DefaultClock{}
	now := clock.now().Unix()

	name := r.RuleProperties.RuleName

	requests, sampled, borrows := r.MatchedRequests, r.SampledRequests, r.BorrowedRequests

	r.mu.Lock()
	r.MatchedRequests, r.SampledRequests, r.BorrowedRequests = 0, 0, 0
	r.mu.Unlock()

	return &samplingStatisticsDocument{
		RequestCount: &requests,
		SampledCount: &sampled,
		BorrowCount:  &borrows,
		RuleName:     &name,
		Timestamp:    &now,
	}
}
