// Package scheduler handles the feasbility of the task
// 1. Determine a set of candidate workers on which a task could run
// 2. Score the candidate worker from best to worst
// 3. Pick the worker with the best score
package scheduler

type Scheduler interface {
	SelectCandidateNodes()
	Score()
	Pick()
}
