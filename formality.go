package dragoman

const (
	// FormalityUnspecified indicates the absence of a specified formality level in
	// translation or language settings.
	FormalityUnspecified Formality = ""

	// FormalityFormal represents the use of formal language and address forms,
	// applicable across all languages where such distinctions exist.
	FormalityFormal Formality = "formal"

	// FormalityInformal specifies the use of informal language and address forms,
	// applicable across various languages where distinctions between formality
	// levels exist.
	FormalityInformal Formality = "informal"
)

// Formality represents the level of formality in language, ranging from formal
// to informal, providing contextual cues for language translation or usage. It
// supports checking if a specific formality has been set and converts to its
// string representation. Formality also guides language adjustments based on
// the desired tone and social context.
type Formality string

// IsSpecified reports whether a [Formality] instance has a specified value
// other than the default unspecified state.
func (f Formality) IsSpecified() bool {
	return f != FormalityUnspecified
}

// String returns the string representation of the formal language setting
// encapsulated by the [Formality] type.
func (f Formality) String() string {
	return string(f)
}

func (f Formality) instruction() string {
	if f == FormalityInformal {
		return "Use informal language and address forms, applicable across all languages where such distinctions exist."
	}
	if f == FormalityFormal {
		return "Use formal language and address forms, applicable across all languages where such distinctions exist."
	}
	return ""
}
