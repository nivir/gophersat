package solver

import "fmt"

// A Clause is a list of Lit, associated with possible data (for learned clauses).
type Clause struct {
	lits     []Lit
	lbdValue uint32 // contains value for lbd (30 lowest bytes) but also flags to indicate whether clause is learned or not, and locked or not
	activity float32
}

const (
	learnedMask uint32 = 1 << 31
	lockedMask  uint32 = 1 << 30
	bothMasks   uint32 = learnedMask | lockedMask
)

// NewClause returns a clause whose lits are given as an argument.
func NewClause(lits []Lit) *Clause {
	return &Clause{lits: lits}
}

// NewLearnedClause returns a new clause marked as learned.
func NewLearnedClause(lits []Lit) *Clause {
	return &Clause{lits: lits, lbdValue: learnedMask}
}

// Learned returns true iff c was a learned clause.
func (c *Clause) Learned() bool {
	return c.lbdValue&learnedMask == learnedMask
}

func (c *Clause) lock() {
	// c.locked = true
	c.lbdValue = c.lbdValue | lockedMask
}

func (c *Clause) unlock() {
	// c.locked = false
	c.lbdValue = c.lbdValue & ^lockedMask
}

func (c *Clause) lbd() int {
	return int(c.lbdValue & ^bothMasks)
}

func (c *Clause) setLbd(lbd int) {
	c.lbdValue = (c.lbdValue & bothMasks) | uint32(lbd)
}

func (c *Clause) incLbd() {
	c.lbdValue++
}

func (c *Clause) isLocked() bool {
	// return c.learned && c.locked
	return c.lbdValue&bothMasks == bothMasks
}

// Len returns the nb of lits in the clause.
func (c *Clause) Len() int {
	return len(c.lits)
}

// First returns the first lit from the clause.
func (c *Clause) First() Lit {
	return c.lits[0]
}

// Second returns the second lit from the clause.
func (c *Clause) Second() Lit {
	return c.lits[1]
}

// Get returns the ith literal from the clause.
func (c *Clause) Get(i int) Lit {
	return c.lits[i]
}

// Set sets the ith literal of the clause.
func (c *Clause) Set(i int, l Lit) {
	c.lits[i] = l
}

// swapFirstWith swaps the first and ith lits from the clause.
func (c *Clause) swapFirstWith(i int) {
	c.lits[0], c.lits[i] = c.lits[i], c.lits[0]
}

// swapSecondWith swaps the second and ith lits from the clause.
func (c *Clause) swapSecondWith(i int) {
	c.lits[1], c.lits[i] = c.lits[i], c.lits[1]
}

// Shrink reduces the length of the clauses, by removing all lits
// starting from position newLen.
func (c *Clause) Shrink(newLen int) {
	c.lits = c.lits[:newLen]
}

// CNF returns a DIMACS CNF representation of the clause.
func (c *Clause) CNF() string {
	res := ""
	for _, lit := range c.lits {
		res += fmt.Sprintf("%d ", lit.Int())
	}
	return fmt.Sprintf("%s0", res)
}

// OutputClause displays a clause on stdout.
func OutputClause(c *Clause) {
	fmt.Printf("[")
	for i, l := range c.lits {
		if i < c.Len()-1 {
			fmt.Printf("%d, ", l.Int())
		} else {
			fmt.Printf("%d", l.Int())
		}
	}
	fmt.Printf("]\n")
}
