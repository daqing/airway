package membership_plugin

import "github.com/daqing/airway/lib/repo"

type MembershipResp struct {
	Name      string
	ExpiredAt repo.Timestamp
}

func (r MembershipResp) Fields() []string {
	return []string{"name", "expired_at"}
}
