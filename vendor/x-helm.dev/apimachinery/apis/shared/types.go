// +kubebuilder:object:generate=true
package shared

type MediaType string

const (
	MediaLogo        MediaType = "logo"
	MediaLogoWhite   MediaType = "logo_white"
	MediaIcon        MediaType = "icon"
	MediaIcon192_192 MediaType = "icon_192x192"
	MediaHeroImage   MediaType = "hero_image"
	MediaIntroVideo  MediaType = "intro_video"
)

type LinkType string

const (
	LinkWebsite         LinkType = "website"
	LinkSupportDesk     LinkType = "support_desk"
	LinkFacebook        LinkType = "facebook"
	LinkLinkedIn        LinkType = "linkedin"
	LinkTwitter         LinkType = "twitter"
	LinkTwitterID       LinkType = "twitter_id"
	LinkYouTube         LinkType = "youtube"
	LinkSourceRepo      LinkType = "src_repo"
	LinkStarRepo        LinkType = "star_repo"
	LinkDocsRepo        LinkType = "docs_repo"
	LinkDatasheetFormID LinkType = "datasheet_form_id"
)

// ImageSpec contains information about an image used as an icon.
type ImageSpec struct {
	// The source for image represented as either an absolute URL to the image or a Data URL containing
	// the image. Data URLs are defined in RFC 2397.
	Source string `json:"src"`

	// (optional) The size of the image in pixels (e.g., 25x25).
	Size string `json:"size,omitempty"`

	// (optional) The mine type of the image (e.g., "image/png").
	Type string `json:"type,omitempty"`
}

// MediaSpec contains information about an image/video.
type MediaSpec struct {
	// Description is human readable content explaining the purpose of the link.
	Description MediaType `json:"description,omitempty"`

	ImageSpec `json:",inline"`
}

// ContactData contains information about an individual or organization.
type ContactData struct {
	// Name is the descriptive name.
	Name string `json:"name,omitempty"`

	// Url could typically be a website address.
	URL string `json:"url,omitempty"`

	// Email is the email address.
	Email string `json:"email,omitempty"`
}

// Link contains information about an URL to surface documentation, dashboards, etc.
type Link struct {
	// Description is human readable content explaining the purpose of the link.
	Description string `json:"description,omitempty"`

	// Url typically points at a website address.
	URL string `json:"url,omitempty"`
}
