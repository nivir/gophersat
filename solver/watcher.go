package solver

import "sort"

type watcher struct {
	other  Lit // Another lit from the clause
	clause *Clause
}

// A watcherList is a structure used to store clauses and propagate unit literals efficiently.
type watcherList struct {
	nbOriginal int         // Original # of clauses
	nbLearned  int         // # of learned clauses
	nbMax      int         // Max # of learned clauses at current moment
	idxReduce  int         // # of calls to reduce + 1
	wlistBin   [][]watcher // For each literal, a list of binary clauses where its negation appears
	wlist      [][]*Clause // For each literal, a list of non-binary clauses where its negation appears at first or second position
	clauses    []*Clause   // All the clauses
}

// initWatcherList makes a new watcherList for the solver.
func (s *Solver) initWatcherList(clauses []*Clause) {
	nbMax := initNbMaxClauses
	newClauses := make([]*Clause, len(clauses), len(clauses)*2) // Make room for future learned clauses
	copy(newClauses, clauses)
	s.wl = watcherList{
		nbOriginal: len(clauses),
		nbMax:      nbMax,
		idxReduce:  1,
		wlistBin:   make([][]watcher, s.nbVars*2),
		wlist:      make([][]*Clause, s.nbVars*2),
		clauses:    newClauses,
	}
	for _, c := range clauses {
		s.watchClause(c)
	}
}

// bumpNbMax increases the max nb of clauses used.
// It is typically called after a restart.
func (s *Solver) bumpNbMax() {
	s.wl.nbMax += incrNbMaxClauses
}

// postponeNbMax increases the max nb of clauses used.
// It is typically called when too many good clauses were learned and a cleaning was expected.
func (s *Solver) postponeNbMax() {
	s.wl.nbMax += incrPostponeNbMax
}

// Utilities for sorting according to clauses' LBD and activities.
func (wl *watcherList) Len() int { return wl.nbLearned }

func (wl *watcherList) Less(i, j int) bool {
	idxI := i + wl.nbOriginal
	idxJ := j + wl.nbOriginal
	lbdI := wl.clauses[idxI].lbd()
	lbdJ := wl.clauses[idxJ].lbd()
	// Sort by lbd, break ties by activity
	return lbdI > lbdJ || (lbdI == lbdJ && wl.clauses[idxI].activity < wl.clauses[idxJ].activity)
}

func (wl *watcherList) Swap(i, j int) {
	idxI := i + wl.nbOriginal
	idxJ := j + wl.nbOriginal
	wl.clauses[idxI], wl.clauses[idxJ] = wl.clauses[idxJ], wl.clauses[idxI]
}

// Watches the provided clause.
func (s *Solver) watchClause(c *Clause) {
	first := c.First()
	second := c.Second()
	neg0 := first.Negation()
	neg1 := second.Negation()
	if c.Len() == 2 {
		s.wl.wlistBin[neg0] = append(s.wl.wlistBin[neg0], watcher{clause: c, other: second})
		s.wl.wlistBin[neg1] = append(s.wl.wlistBin[neg1], watcher{clause: c, other: first})
	} else {
		s.wl.wlist[neg0] = append(s.wl.wlist[neg0], c)
		s.wl.wlist[neg1] = append(s.wl.wlist[neg1], c)
	}
}

func (s *Solver) unwatchClause(c *Clause) {
	if c.Len() == 2 {
		for i := 0; i < 2; i++ {
			neg := c.Get(i).Negation()
			j := 0
			length := len(s.wl.wlistBin[neg])
			// We're looking for the index of the clause.
			// This will panic if c is not in wlist[neg], but this shouldn't happen.
			for s.wl.wlistBin[neg][j].clause != c {
				j++
			}
			s.wl.wlistBin[neg][j] = s.wl.wlistBin[neg][length-1]
			s.wl.wlistBin[neg] = s.wl.wlistBin[neg][:length-1]
		}
	} else {
		for i := 0; i < 2; i++ {
			neg := c.Get(i).Negation()
			j := 0
			length := len(s.wl.wlist[neg])
			// We're looking for the index of the clause.
			// This will panic if c is not in wlist[neg], but this shouldn't happen.
			for s.wl.wlist[neg][j] != c {
				j++
			}
			s.wl.wlist[neg][j] = s.wl.wlist[neg][length-1]
			s.wl.wlist[neg] = s.wl.wlist[neg][:length-1]
		}
	}
}

// reduceLearned removes a few learned clauses that are deemed useless.
func (s *Solver) reduceLearned() {
	sort.Sort(&s.wl)
	length := s.wl.nbLearned / 2
	if s.wl.clauses[s.wl.nbOriginal+length].lbd() <= 3 { // Lots of good clauses, postpone reduction
		s.postponeNbMax()
	}
	nbRemoved := 0
	for i := 0; i < length; i++ {
		idx := i + s.wl.nbOriginal
		c := s.wl.clauses[idx]
		if c.lbd() <= 2 || c.isLocked() {
			continue
		}
		nbRemoved++
		s.Stats.NbDeleted++
		s.wl.clauses[idx] = s.wl.clauses[len(s.wl.clauses)-nbRemoved]
		s.unwatchClause(c)
	}
	s.wl.clauses = s.wl.clauses[:len(s.wl.clauses)-nbRemoved]
	s.wl.nbLearned -= nbRemoved
}

// Adds the given clause and updates watchers.
// If too many clauses have been learned yet, one will be removed.
func (s *Solver) addClause(c *Clause) {
	s.wl.nbLearned++
	s.wl.clauses = append(s.wl.clauses, c)
	s.watchClause(c)
	s.clauseBumpActivity(c)
}

// If l is negative, -lvl is returned. Else, lvl is returned.
func lvlToSignedLvl(l Lit, lvl decLevel) decLevel {
	if l.IsPositive() {
		return lvl
	}
	return -lvl
}

// Removes the first occurrence of c from lst.
// The element *must* be present into lst.
func removeFrom(lst []*Clause, c *Clause) []*Clause {
	i := 0
	for lst[i] != c {
		i++
	}
	last := len(lst) - 1
	lst[i] = lst[last]
	return lst[:last]
}

// Unifies the given literal and returns a conflict clause, or nil if no conflict arose.
func (s *Solver) unifyLiteral(lit Lit, lvl decLevel) *Clause {
	s.model[lit.Var()] = lvlToSignedLvl(lit, lvl)
	ptr := len(s.trail)
	s.trail = append(s.trail, lit)
	for ptr < len(s.trail) {
		lit := s.trail[ptr]
		for _, w := range s.wl.wlistBin[lit] {
			v2 := w.other.Var()
			if assign := s.model[v2]; assign == 0 { // Other was unbounded: propagate
				s.reason[v2] = w.clause
				w.clause.lock()
				s.model[v2] = lvlToSignedLvl(w.other, lvl)
				s.trail = append(s.trail, w.other)
			} else if (assign > 0) != w.other.IsPositive() { // Conflict here
				return w.clause
			}
		}
		for _, c := range s.wl.wlist[lit] {
			res, unit := s.simplifyClause(c)
			switch res {
			case Unsat: // A conflict was met in current clause
				return c
			case Unit:
				v := unit.Var()
				s.reason[v] = c
				c.lock()
				s.model[v] = lvlToSignedLvl(unit, lvl)
				s.trail = append(s.trail, unit)
			}
		}
		ptr++
	}
	// No unsat clause was met
	return nil
}

// Used by simplifyClause. Indexes of the 1st & 2nd free lits, if any.
var freeIdx [2]int

// simplifyClause simplifies the given clause according to current binding.
// It returns a new status, and a potential unit literal.
// TODO: improve quality of code.
func (s *Solver) simplifyClause(clause *Clause) (Status, Lit) {
	nbFound := 0
	len := clause.Len()
	for i := 0; i < len; i++ {
		lit := clause.Get(i)
		if assign := s.model[lit.Var()]; assign == 0 {
			freeIdx[nbFound] = i
			nbFound++
			if nbFound == 2 {
				break
			}
		} else if (assign > 0) == lit.IsPositive() {
			return Sat, -1
		}
	}
	switch nbFound {
	case 0:
		return Unsat, -1
	case 1:
		return Unit, clause.Get(freeIdx[0])
	}
	// 2 lits are known to be unbounded
	switch freeIdx[0] {
	case 0: // c[0] is not removed, c[1] is
		n1 := &s.wl.wlist[clause.Second().Negation()]
		nf1 := &s.wl.wlist[clause.Get(freeIdx[1]).Negation()]
		clause.swapSecondWith(freeIdx[1])
		*n1 = removeFrom(*n1, clause)
		*nf1 = append(*nf1, clause)
	case 1: // c[0] is removed, not c[1]
		n0 := &s.wl.wlist[clause.First().Negation()]
		nf1 := &s.wl.wlist[clause.Get(freeIdx[1]).Negation()]
		clause.swapFirstWith(freeIdx[1])
		*n0 = removeFrom(*n0, clause)
		*nf1 = append(*nf1, clause)
	default: // Both c[0] & c[1] are removed
		n0 := &s.wl.wlist[clause.First().Negation()]
		n1 := &s.wl.wlist[clause.Second().Negation()]
		nf0 := &s.wl.wlist[clause.Get(freeIdx[0]).Negation()]
		nf1 := &s.wl.wlist[clause.Get(freeIdx[1]).Negation()]
		clause.swapFirstWith(freeIdx[0])
		clause.swapSecondWith(freeIdx[1])
		*n0 = removeFrom(*n0, clause)
		*n1 = removeFrom(*n1, clause)
		*nf0 = append(*nf0, clause)
		*nf1 = append(*nf1, clause)
	}
	return Many, -1
}
