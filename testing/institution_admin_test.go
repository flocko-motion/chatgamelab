package testing

import (
	"cgl/api/routes"
	"cgl/obj"
)

// TestAdminLocksAndUnlocksOrganisation: a lock stops the organisation's members, keeps their data,
// and is fully reversible; admins see lock and consent in the overview.
func (s *InstitutionRegistrationSuite) TestAdminLocksAndUnlocksOrganisation() {
	admin := s.DevUser()
	head, req, res := s.register("sperre@jz-test.de")
	instID := res.Institution.ID.String()
	ws := Must(head.CreateWorkshop(instID, "Workshop vor der Sperre"))

	// Only admins can lock.
	MustFail(head.Post("institutions/"+instID+"/disable", routes.LockInstitutionRequest{}, nil))

	var locked obj.Institution
	MustSucceed(admin.Post("institutions/"+instID+"/disable", routes.LockInstitutionRequest{Reason: "Test"}, &locked))
	s.NotNil(locked.DisabledAt)

	// The head is stopped everywhere except /users/me, which reports the lock.
	_, err := head.GetInstitution(instID)
	s.Require().Error(err)
	s.Contains(err.Error(), obj.ErrCodeInstitutionDisabled)
	err = head.Get("workshops/"+ws.ID.String(), nil)
	s.Require().Error(err)
	s.Contains(err.Error(), obj.ErrCodeInstitutionDisabled)
	me := Must(head.GetMe())
	s.True(me.InstitutionDisabled)

	// Members of other organisations are not affected.
	other := s.DevUserClient(devHead1ID, "head-1")
	s.False(Must(other.GetMe()).InstitutionDisabled)
	MustSucceed(other.Get("institutions/"+devInstitutionID, nil))

	// Admin overview shows the lock, its reason and the consent.
	var all []obj.Institution
	MustSucceed(admin.Get("institutions", &all))
	var found *obj.Institution
	for i := range all {
		if all[i].ID == res.Institution.ID {
			found = &all[i]
		}
	}
	s.Require().NotNil(found)
	s.NotNil(found.DisabledAt)
	s.Require().NotNil(found.DisabledReason)
	s.Equal("Test", *found.DisabledReason)
	s.Require().NotNil(found.Declaration)
	s.False(found.Declaration.Pending)
	s.Require().NotNil(found.Operator)
	s.Equal(req.Operator, *found.Operator)

	// Unlock: everything works again and the workshop is still there.
	MustFail(head.Post("institutions/"+instID+"/enable", nil, nil))
	MustSucceed(admin.Post("institutions/"+instID+"/enable", nil, nil))
	s.False(Must(head.GetMe()).InstitutionDisabled)
	again := Must(head.GetWorkshop(ws.ID.String()))
	s.Equal("Workshop vor der Sperre", again.Name)
}

// TestAdminSeesApplications: admins list all registrations, others may not.
func (s *InstitutionRegistrationSuite) TestAdminSeesApplications() {
	person, req, res := s.register("antrag-offen@jz-test.de")

	var apps []obj.InstitutionApplication
	MustSucceed(s.DevUser().Get("institution-applications", &apps))
	var found bool
	for _, a := range apps {
		if a.InstitutionName == req.InstitutionName {
			found = true
			// Stage 1: confirmed right away, linked to the new organisation.
			s.Equal(obj.InstitutionApplicationConfirmed, a.Status)
			s.Require().NotNil(a.InstitutionID)
			s.Equal(res.Institution.ID, *a.InstitutionID)
			s.Require().NotNil(a.Email)
			s.Equal(req.Email, *a.Email)
		}
	}
	s.True(found, "the registration is listed")

	MustFail(person.Get("institution-applications", nil))
	MustFail(s.DevUserClient(devHead1ID, "head-1").Get("institution-applications", nil))
}

// TestAdminCreatesOrganisationWithDetails: manual creation still works and stores operator and address.
func (s *InstitutionRegistrationSuite) TestAdminCreatesOrganisationWithDetails() {
	var inst obj.Institution
	MustSucceed(s.DevUser().Post("institutions", routes.CreateInstitutionRequest{
		Name: "Von Hand angelegt", Operator: "Landkreis", City: "Kleinstadt",
	}, &inst))
	s.Require().NotNil(inst.Operator)
	s.Equal("Landkreis", *inst.Operator)
	got := Must(s.DevUser().GetInstitution(inst.ID.String()))
	s.Require().NotNil(got.City)
	s.Equal("Kleinstadt", *got.City)
	s.Nil(got.Street)
}
