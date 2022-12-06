package common

const (
	DotSeparator      = "."
	ColonSeparator    = ":"
	CommaSeparator    = ","
	SpecialSeparator  = "#@"
	JsonMarshalPrefix = ""
	JsonMarshalIndent = "  "
	Hyphen            = "-"
	Underscore        = "_"
	Slash             = "/"
)

func GenerateKeyBy(namespace, name string) string {
	return namespace + Slash + name
}
