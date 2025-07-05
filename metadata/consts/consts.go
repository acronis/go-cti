package consts

type AccessModifier string

const (
	AccessModifierPublic    AccessModifier = "public"
	AccessModifierPrivate   AccessModifier = "private"
	AccessModifierProtected AccessModifier = "protected"
)

func (a AccessModifier) Integer() int {
	switch a {
	case AccessModifierPublic:
		return 0
	case AccessModifierProtected:
		return 1
	case AccessModifierPrivate:
		return 2
	default:
		return -1
	}
}

const (
	CTI           = "cti.cti"
	Final         = "cti.final"
	Access        = "cti.access"
	AccessField   = "cti.access_field"
	Resilient     = "cti.resilient"
	ID            = "cti.id"
	L10n          = "cti.l10n"
	DisplayName   = "cti.display_name"
	Description   = "cti.description"
	Asset         = "cti.asset"
	Overridable   = "cti.overridable"
	Reference     = "cti.reference"
	Schema        = "cti.schema"
	Meta          = "cti.meta"
	PropertyNames = "cti.propertyNames"
)

const (
	Traits = "cti-traits"
)
