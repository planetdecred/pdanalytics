package types

// AgendaStatusType defines the various agenda statuses.
type AgendaStatusType int8

const (
	// InitialAgendaStatus is the agenda status when the agenda is not yet up for
	// voting and the votes tally is not also available.
	InitialAgendaStatus AgendaStatusType = iota

	// StartedAgendaStatus is the agenda status when the agenda is up for voting.
	StartedAgendaStatus

	// FailedAgendaStatus is the agenda status set when the votes tally does not
	// attain the minimum threshold set. Activation height is not set for such an
	// agenda.
	FailedAgendaStatus

	// LockedInAgendaStatus is the agenda status when the agenda is considered to
	// have passed after attaining the minimum set threshold. This agenda will
	// have its activation height set.
	LockedInAgendaStatus

	// ActivatedAgendaStatus is the agenda status chaincfg.Params.RuleChangeActivationInterval
	// blocks (e.g. 8064 blocks = 2016 * 4 for 4 weeks on mainnet) after
	// LockedInAgendaStatus ("lockedin") that indicates when the rule change is to
	// be effected. https://docs.decred.org/glossary/#rule-change-interval-rci.
	ActivatedAgendaStatus

	// UnknownStatus is used when a status string is not recognized.
	UnknownStatus
)

func (a AgendaStatusType) String() string {
	switch a {
	case InitialAgendaStatus:
		return "upcoming"
	case StartedAgendaStatus:
		return "in progress"
	case LockedInAgendaStatus:
		return "locked in"
	case FailedAgendaStatus:
		return "failed"
	case ActivatedAgendaStatus:
		return "finished"
	default:
		return "unknown"
	}
}

// Ensure at compile time that AgendaStatusType satisfies interface json.Marshaller.
var _ json.Marshaler = (*AgendaStatusType)(nil)

// MarshalJSON is AgendaStatusType default marshaller.
func (a AgendaStatusType) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON is the default unmarshaller for AgendaStatusType.
func (a *AgendaStatusType) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	*a = AgendaStatusFromStr(str)
	return nil
}

// AgendaStatusFromStr creates an agenda status from a string. If "UnknownStatus"
// is returned then an invalid status string has been passed.
func AgendaStatusFromStr(status string) AgendaStatusType {
	switch strings.ToLower(status) {
	case "defined", "upcoming":
		return InitialAgendaStatus
	case "started", "in progress":
		return StartedAgendaStatus
	case "failed":
		return FailedAgendaStatus
	case "lockedin", "locked in":
		return LockedInAgendaStatus
	case "active", "finished":
		return ActivatedAgendaStatus
	default:
		return UnknownStatus
	}
}