package testutil

import "cgl/apiclient"

// The list endpoints with their query: filters, search, order and page.

// ListQuery is what a test asks of a list; zero values are left out.
type ListQuery struct {
	Filter  []string
	Search  string
	SortBy  string
	SortDir string
	Limit   int
	Offset  int
}

// MemberPage is one page of an organisation's members and the total.
type MemberPage = page[User]

// ParticipantPage is one page of a workshop's participants and the total.
type ParticipantPage = page[apiclient.WorkshopParticipantView]

// ListMembersPage is a page of an institution's members.
func (c caller) ListMembersPage(institutionID string, q ListQuery) (MemberPage, error) {
	return decode[MemberPage](c.api.GetApiInstitutionsIdMembersWithResponse(ctx(), institutionID, &apiclient.GetApiInstitutionsIdMembersParams{
		Filter:  nonEmpty(q.Filter),
		Search:  someText(q.Search),
		SortBy:  (*apiclient.MemberSort)(someText(q.SortBy)),
		SortDir: (*apiclient.SortDir)(someText(q.SortDir)),
		Limit:   someCount(q.Limit),
		Offset:  someCount(q.Offset),
	}))
}

// ListParticipantsPage is a page of a workshop's participants.
func (c caller) ListParticipantsPage(workshopID string, q ListQuery) (ParticipantPage, error) {
	return decode[ParticipantPage](c.api.GetApiWorkshopsIdParticipantsWithResponse(ctx(), workshopID, &apiclient.GetApiWorkshopsIdParticipantsParams{
		Filter:  nonEmpty(q.Filter),
		Search:  someText(q.Search),
		SortBy:  (*apiclient.ParticipantSort)(someText(q.SortBy)),
		SortDir: (*apiclient.SortDir)(someText(q.SortDir)),
		Limit:   someCount(q.Limit),
		Offset:  someCount(q.Offset),
	}))
}

func someText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func someCount(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

func nonEmpty(list []string) *[]string {
	if len(list) == 0 {
		return nil
	}
	return &list
}
