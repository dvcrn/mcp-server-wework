package wework

func (w Workspace) ReservableTypeName() string {
	if w.Reservable == nil {
		return ""
	}
	if w.IsHybridSpace {
		return "HybridSpace"
	}
	if w.IsAffiliateCoworking {
		return "AffiliateCoworking"
	}
	if w.IsFranchiseCoworking {
		return "FranchiseCoworking"
	}
	return "Workspace"
}

func (w Workspace) ReservableName() string {
	return w.Location.Name
}

func (w Workspace) ReservableFloorName() string {
	return ""
}
