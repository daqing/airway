package membership_api

import "github.com/daqing/airway/lib/pg_repo"

type MembershipResp struct {
	Name      string
	ExpiredAt pg_repo.Timestamp
}

func (r MembershipResp) Fields() []string {
	return []string{"name", "expired_at"}
}
