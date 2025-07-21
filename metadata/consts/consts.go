package consts

// AccessModifier represents the access level of an entity in the CTI system.
type AccessModifier string

const (
	// AccessModifierPublic indicates that the entity is allowed to be referenced by anyone.
	AccessModifierPublic AccessModifier = "public"
	// AccessModifierProtected indicates that the entity is allowed to be referenced within any package of the same vendor.
	AccessModifierProtected AccessModifier = "protected"
	// AccessModifierPrivate indicates that the entity is allowed to be referenced only by the same package.
	AccessModifierPrivate AccessModifier = "private"
)

// Integer returns the integer representation of the AccessModifier which can be used to simplify the comparison.
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
