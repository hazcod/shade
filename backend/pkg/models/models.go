package models

type EnrolledUser struct {
	Username string
	ID       string
	Hostname string
	IP       string
	LastSeen string
}

type DashboardStats struct {
	TotalUsers           int
	TotalDomains         int
	DuplicatePasswords   int
	CompromisedPasswords int
	UsersWithoutMFA      int
}

type DuplicatePasswordEntry struct {
	User    string
	Domains []string
}
