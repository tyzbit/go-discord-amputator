package goamputate

// A request to pass to the Amputator Bot API.
type AmputationRequest struct {
	options map[string]string
	urls    []string
}

// An instance of the AmputatorBot
type AmputatorBot struct{}

type AmputationResponseObject struct {
	AmpCanonical Canonical   `json:"amp_canonical"`
	Canonical    Canonical   `json:"canonical"`
	Canonicals   []Canonical `json:"canonicals"`
	Origin       Origin      `json:"origin"`
}

type Canonical struct {
	Domain        string  `json:"domain"`
	IsAlt         bool    `json:"is_alt"`
	IsAmp         bool    `json:"is_amp"`
	IsCached      bool    `json:"is_cached"`
	IsValid       bool    `json:"is_valid"`
	Type          string  `json:"type"`
	Url           string  `json:"url"`
	UrlSimilarity float64 `json:"url_similarity"`
}

type Origin struct {
	Domain   string `json:"domain"`
	IsAmp    bool   `json:"is_amp"`
	IsCached bool   `json:"is_cached"`
	IsValid  bool   `json:"is_valid"`
	Url      string `json:"url"`
}
