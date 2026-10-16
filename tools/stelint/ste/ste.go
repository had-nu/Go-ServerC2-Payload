// Package ste contains the mechanical rules that stelint checks.
//
// The rules come from ASD-STE100 Simplified Technical English, Issue 9
// (2025-01-15), Part 1. This file states a condensed form of the rules that a
// program can check. The standard is the only authority.
package ste

import "strings"

// The word counts that the standard uses. Rule 8.6 counts a number, a number
// with a unit, an abbreviation, an alphanumeric identifier, quoted text, a
// title, and a proper name as one word. Rule 8.5 counts text in parentheses as
// one word. Rule 8.7 counts a hyphenated word as one word.
const (
	MaxWordsProcedure        = 20 // rule 5.1
	MaxWordsDescriptive      = 25 // rule 6.3
	MaxWordsNote             = 25 // rule 5.5, notes do not obey rule 5.1
	MaxSentencesPerParagraph = 6  // rule 6.6
	MaxWordsMultiWordNoun    = 3  // rule 2.1
)

// ModalVerbs lists the modals that the dictionary approves. MAY, MIGHT, COULD,
// SHALL, and SHOULD have no entry. The standard maps them to CAN, CANNOT, and
// MUST.
var ModalVerbs = []string{"BE", "CAN", "CANNOT", "MUST", "WILL"}

// UnapprovedModals lists the words that the dictionary does not approve, with
// the approved alternative.
var UnapprovedModals = map[string]string{
	"MAY":    "CAN",
	"MIGHT":  "CAN",
	"COULD":  "CAN",
	"SHALL":  "MUST",
	"SHOULD": "MUST",
	"OUGHT":  "MUST",
}

// NounOnlyVerbs lists the verbs that the first version of this project listed as
// approved. The dictionary gives each of them only as a noun.
var NounOnlyVerbs = map[string]string{
	"CHECK": "CHECK (n), used as 'do a check of'",
	"TEST":  "TEST (n), used as 'do the test'",
}

// ApprovedVerbs is the list of approved verbs from the dictionary, page 2-0-19.
// The list has 208 entries. The letters J, N, Q, V, X, Y, and Z have no entry.
//
// The entries use "|" as the separator. Four approved verbs have two words:
// COME ON, GO OFF, MAKE SURE, and PUT ON. A space as the separator would split
// those four into eight entries and give a count of 212.
var ApprovedVerbs = newVerbSet(
	"ABSORB|ACCEPT|ACTIVATE|ADAPT|ADD|ADJUST|AGREE|ALIGN|APPLY|ARM|ASSEMBLE|ATTACH",
	"BALANCE|BE|BECOME|BEND|BLEED|BLOW|BOND|BREAK|BREATHE|BURN|BYPASS",
	"CALCULATE|CALIBRATE|CAN|CANCEL|CANNOT|CATCH|CAUSE|CHANGE|CHARGE|CLEAN|CLOSE|COLLECT|COME|COME ON|COMPARE|COMPLETE|COMPRESS|CONNECT|CONTACT|CONTAIN|CONTINUE|CONTROL|CORRECT|COUNT|CUT",
	"DEACTIVATE|DECREASE|DE-ENERGIZE|DEFLATE|DEFUEL|DEPLOY|DISARM|DISASSEMBLE|DISCARD|DISCONNECT|DISENGAGE|DIVIDE|DO|DRAIN|DRINK|DRY",
	"EAT|EJECT|ENERGIZE|ENGAGE|ERASE|EXAMINE|EXPAND|EXTEND|EXTINGUISH",
	"FALL|FEATHER|FEEL|FILL|FIND|FIRE|FLASH|FLOW|FLUSH|FOLD|FOLLOW|FREEZE",
	"GET|GIVE|GO|GO OFF|GROUND",
	"HANG|HAVE|HEAR|HELP|HIT|HOLD",
	"IDENTIFY|IGNORE|ILLUMINATE|INCLUDE|INCREASE|INFLATE|INSTALL|INTERCHANGE|ISOLATE",
	"KEEP|KILL|KNOW",
	"LATCH|LET|LIFT|LISTEN|LOCK|LOOK|LOOSEN|LOWER|LUBRICATE",
	"MAKE|MAKE SURE|MEASURE|MELT|MIX|MONITOR|MOOR|MOVE|MULTIPLY|MUST",
	"OBEY|OCCUR|OPEN|OPERATE|OVERRIDE",
	"PAINT|PARK|POINT|POLISH|PREPARE|PRESSURIZE|PREVENT|PROTRUDE|PULL|PUSH|PUT|PUT ON",
	"READ|RECEIVE|RECOMMEND|RECORD|RECYCLE|REFER|REFUEL|REJECT|RELEASE|REMOVE|REPAIR|REPLACE|RETRACT|RUB",
	"SAFETY|SCHEDULE|SEAL|SEE|SELECT|SEND|SENSE|SET|SHAKE|SHOW|SIMULATE|SMELL|SMOKE|SOAK|SPEAK|SPILL|SPRAY|START|STAY|STOP|STOW|SUBTRACT|SUPPLY|SWALLOW",
	"TAG|TAP|TELL|THINK|TIGHTEN|TILT|TORQUE|TOUCH|TOW|TRANSMIT|TRY|TUNE|TURN|TWIST",
	"UNFOLD|UNLOCK|UNWIND|USE",
	"WAIT|WALK|WANT|WEAR|WEIGH|WILL|WIND|WRITE",
)

func newVerbSet(lines ...string) map[string]bool {
	set := make(map[string]bool)
	for _, line := range lines {
		for _, v := range strings.Split(line, "|") {
			set[strings.TrimSpace(v)] = true
		}
	}
	return set
}

// SafetyWords are the words that rule 7.1 accepts for a level of risk. Other
// sectors use other words, for example "danger", "attention", and "notice".
var SafetyWords = []string{"WARNING", "CAUTION"}

// NoteWord marks a note. A note gives information only.
const NoteWord = "NOTE"
