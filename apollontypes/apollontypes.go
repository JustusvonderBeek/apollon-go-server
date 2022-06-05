package apollontypes

import "net"

type User struct {
	Username   string
	UserId     uint32
	Connection net.Conn
}
