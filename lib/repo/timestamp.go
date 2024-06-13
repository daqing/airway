package repo

import (
	"fmt"
	"time"
)

type Timestamp time.Time

func (ts Timestamp) MarshalJSON() ([]byte, error) {
	t := time.Time(ts)
	str := fmt.Sprintf("%d", t.Unix())

	return []byte(str), nil
}
