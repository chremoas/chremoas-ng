package sets

import (
	"github.com/bwmarrin/discordgo"
)

type DiscordRoleSet struct {
	Set map[discordgo.Role]bool
}

func NewDiscordRoleSet() *DiscordRoleSet {
	return &DiscordRoleSet{make(map[discordgo.Role]bool)}
}

func (set *DiscordRoleSet) Add(s discordgo.Role) bool {
	if s == (discordgo.Role{}) {
		return false
	}

	_, found := set.Set[s]
	set.Set[s] = true
	return !found
}

func (set *DiscordRoleSet) Remove(s discordgo.Role) {
	delete(set.Set, s)
}

func (set *DiscordRoleSet) Contains(s discordgo.Role) bool {
	_, found := set.Set[s]
	return found
}

func (set *DiscordRoleSet) Len() int {
	return len(set.Set)
}

func (set *DiscordRoleSet) FromSlice(slice []discordgo.Role) {
	for s := range slice {
		set.Add(slice[s])
	}
}

func (set *DiscordRoleSet) FromPtrSlice(slice []*discordgo.Role) {
	for s := range slice {
		set.Add(*slice[s])
	}
}

func (set *DiscordRoleSet) ToSlice() (slice []discordgo.Role) {
	for s := range set.Set {
		slice = append(slice, s)
	}

	return slice
}

func (set *DiscordRoleSet) ToStringSlice() (slice []string) {
	for s := range set.Set {
		slice = append(slice, s.Name)
	}

	return slice
}

func (set *DiscordRoleSet) Intersection(set1 *DiscordRoleSet) *DiscordRoleSet {
	var output = NewDiscordRoleSet()

	for s := range set.Set {
		if set1.Contains(s) {
			output.Add(s)
		}
	}

	return output
}

func (set *DiscordRoleSet) Difference(set1 *DiscordRoleSet) *DiscordRoleSet {
	var output = NewDiscordRoleSet()

	for s := range set.Set {
		if !set1.Contains(s) {
			output.Add(s)
		}
	}

	return output
}