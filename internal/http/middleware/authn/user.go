package authn

type User struct {
	Email       string
	Provider    string
	Subject     string
	DisplayName string
	OrgID       string
	TokenID     string
}
