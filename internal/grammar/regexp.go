package grammar

// Grammar: https://github.com/docker/distribution/blob/2800ab02245e2eafc10e338939511dd1aeb5e135/reference/reference.go#L4-L24
const (
	Reference = Name + (`(?::` + Tag + `)?`) + (`(?:@` + Digest + `)?`)

	Name            = (`(?:` + Domain + `\/)?`) + PathComponent + (`(?:\/` + PathComponent + `)*`)
	Domain          = DomainComponent + (`(?:\.` + DomainComponent + `)*`) + ("(?::" + PortNumber + ")?")
	DomainComponent = `(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])`
	PortNumber      = `[0-9]+`
	Tag             = `[\w][\w.-]{0,127}`
	PathComponent   = AlphaNumeric + (`(?:` + Separator + AlphaNumeric + `)*`)
	AlphaNumeric    = `[a-z0-9]+`
	Separator       = `[_.]|__|[-]*`

	Digest                   = DigestAlgorithm + ":" + DigestHex
	DigestAlgorithm          = DigestAlgorithmComponent + (`(?:` + DigestAlgorithmSeparator + DigestAlgorithmComponent + `)*`)
	DigestAlgorithmSeparator = `[+.-_]`
	DigestAlgorithmComponent = `[A-Za-z][A-Za-z0-9]*`
	DigestHex                = `[0-9a-fA-F]{32,}`
)
