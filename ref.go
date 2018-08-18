package rbxfile

import (
	"crypto/rand"
	"io"
)

// PropRef specifies the property of an instance that is a reference, which is
// to be resolved into its referent at a later time.
type PropRef struct {
	Instance  *Instance
	Property  string
	Reference string
}

// References is a mapping of reference strings to Instances.
type References map[string]*Instance

// Resolve resolves a PropRef and sets the value of the property using
// References. If the referent does not exist, and the reference is not empty,
// then false is returned. True is returned otherwise.
func (refs References) Resolve(propRef PropRef) bool {
	if refs == nil {
		return false
	}
	if propRef.Instance == nil {
		return false
	}
	referent := refs[propRef.Reference]
	propRef.Instance.Properties[propRef.Property] = ValueReference{
		Instance: referent,
	}
	return referent != nil && !IsEmptyReference(propRef.Reference)
}

// Get gets a reference from an Instance, using References to check for
// duplicates. If the instance's reference already exists in References, then
// a new reference is generated and applied to the instance. The instance's
// reference is then added to References.
func (refs References) Get(instance *Instance) (ref string) {
	if instance == nil {
		return ""
	}

	ref = instance.Reference
	if refs == nil {
		return ref
	}
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
				instance.Reference = ref
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

func generateUUID() string {
	var buf [32]byte
	if _, err := io.ReadFull(rand.Reader, buf[:16]); err != nil {
		panic(err)
	}
	buf[6] = (buf[6] & 0x0F) | 0x40 // Version 4       ; 0100XXXX
	buf[8] = (buf[8] & 0x3F) | 0x80 // Variant RFC4122 ; 10XXXXXX
	const hextable = "0123456789ABCDEF"
	for i := len(buf)/2 - 1; i >= 0; i-- {
		buf[i*2+1] = hextable[buf[i]&0x0f]
		buf[i*2] = hextable[buf[i]>>4]
	}
	return string(buf[:])
}

// GenerateReference generates a unique string that can be used as a reference
// to an Instance.
func GenerateReference() string {
	return "RBX" + generateUUID()
}
