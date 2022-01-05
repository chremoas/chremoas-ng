package sets

import (
	"github.com/chremoas/chremoas-ng/internal/payloads"
)

type ChremoasRoleSet struct {
	Set map[int64]payloads.Role
}

func NewChremoasRoleSet() *ChremoasRoleSet {
	return &ChremoasRoleSet{make(map[int64]payloads.Role)}
}

func (set *ChremoasRoleSet) Add(s payloads.Role) bool {
	if s == (payloads.Role{}) {
		return false
	}

	_, found := set.Set[s.ChatID]
	set.Set[s.ChatID] = s
	return !found
}

func (set *ChremoasRoleSet) Remove(s payloads.Role) {
	delete(set.Set, s.ChatID)
}

func (set *ChremoasRoleSet) Merge(newSet *ChremoasRoleSet) {
	set.FromSlice(newSet.ToSlice())
}

func (set *ChremoasRoleSet) Contains(s payloads.Role) bool {
	_, found := set.Set[s.ChatID]
	return found
}

func (set *ChremoasRoleSet) Len() int {
	return len(set.Set)
}

func (set *ChremoasRoleSet) FromSlice(slice []payloads.Role) {
	for s := range slice {
		set.Add(slice[s])
	}
}

func (set *ChremoasRoleSet) FromPtrSlice(slice []*payloads.Role) {
	for s := range slice {
		set.Add(*slice[s])
	}
}

func (set *ChremoasRoleSet) ToSlice() (slice []payloads.Role) {
	for _, s := range set.Set {
		slice = append(slice, s)
	}

	return slice
}

func (set *ChremoasRoleSet) Intersection(set1 *ChremoasRoleSet) *ChremoasRoleSet {
	var output = NewChremoasRoleSet()

	for _, s := range set.Set {
		if set1.Contains(s) {
			output.Add(s)
		}
	}

	return output
}

func (set *ChremoasRoleSet) Difference(set1 *ChremoasRoleSet) *ChremoasRoleSet {
	var output = NewChremoasRoleSet()

	for _, s := range set.Set {
		if !set1.Contains(s) {
			output.Add(s)
		}
	}

	return output
}
