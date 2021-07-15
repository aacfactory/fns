package fns

type AuthCredentials interface {
	Transfer(target AuthCredentials) (err error)
}

type User interface {
	Expired() (expired bool)
	Attributes() (attributes UserAttributes)
	Principal() (principal UserPrincipal)
}

type UserAttributes interface {

}

type UserPrincipal interface {

}