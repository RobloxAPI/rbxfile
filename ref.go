package rbxfile

import (
	"encoding/hex"
	"github.com/satori/go.uuid"
	"strings"
)

// GetReference gets a reference from an Instance, using refs to check
// for duplicates. If the instance's reference already exists in refs, then a
// new reference is generated and applied to the instance. The instance's
// reference is then added to refs.
func GetReference(instance *Instance, refs map[string]*Instance) (ref string) {
	ref = string(instance.Reference)
	// If the reference is not empty, or if the reference is not marked, or
	// the marked reference already refers to the current instance, then do
	// nothing.
	if IsEmptyReference(ref) || refs[ref] != nil && refs[ref] != instance {
		// Otherwise, regenerate the reference until it is not a duplicate.
		for {
			// If a generated reference matches a reference that was not yet
			// traversed, then the latter reference will be regenerated, which
			// may not match Roblox's implementation. It is difficult to
			// discern whether this is correct because it is extremely
			// unlikely that a duplicate will be generated.
			ref = GenerateReference()
			if _, ok := refs[ref]; !ok {
				instance.Reference = []byte(ref)
				break
			}
		}
	}
	// Mark reference as taken.
	refs[ref] = instance
	return ref
}

// IsEmptyReference returns whether a reference string is considered "empty",
// and therefore does not have a referent.
func IsEmptyReference(ref string) bool {
	switch ref {
	case "", "null", "nil":
		return true
	default:
		return false
	}
}

// GenerateReference generates a unique string that can be used as a reference
// to an Instance.
func GenerateReference() string {
	return "RBX" + strings.ToUpper(hex.EncodeToString(uuid.NewV4().Bytes()))
}
